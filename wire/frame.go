package wire

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

func encodeUint16(v uint16) []byte {
	tmp := make([]byte, 2)
	binary.BigEndian.PutUint16(tmp, v)
	return tmp
}
func encodeUint32(v uint32) []byte {
	tmp := make([]byte, 4)
	binary.BigEndian.PutUint32(tmp, v)
	return tmp
}
func decodeUint32(b []byte) uint32 { return binary.BigEndian.Uint32(b) }
func decodeUint16(b []byte) uint16 { return binary.BigEndian.Uint16(b) }

type Frame struct {
	StreamID  uint32
	FrameKind FrameKind
	Flags     uint8
	Length    uint16
	Payload   []byte
}

var magic = []byte("arf")
var magicLen = len(magic)

func (f *Frame) Bytes(method CompressionMethod) []byte {
	f.Payload = method.Compress(f.Payload)
	f.Length = uint16(len(f.Payload))

	return bytes.Join([][]byte{
		magic,
		encodeUint32(f.StreamID),
		{byte(f.FrameKind), f.Flags},
		encodeUint16(f.Length),
		f.Payload,
	}, nil)
}

func (f *Frame) ValidateKind(expected FrameKind, associated bool) error {
	if f.FrameKind != expected {
		return &FrameTypeMismatchError{expected, f.FrameKind}
	}

	if associated && f.StreamID == 0x00 {
		return &UnexpectedUnassociatedFrameError{Kind: f.FrameKind}
	} else if !associated && f.StreamID != 0x00 {
		return &UnexpectedAssociatedFrameError{Kind: f.FrameKind}
	}

	return nil
}

func (f *Frame) ValidateSize(s int) error {
	if f.Length != uint16(s) {
		return &InvalidFrameLengthError{
			fmt.Sprintf("invalid length for frame %s: %d bytes are required, received %d", f.FrameKind, s, f.Length)}
	}

	return nil
}

func (f *Frame) Decompress(method CompressionMethod) (err error) {
	f.Payload, err = method.Decompress(f.Payload)
	if err == nil {
		f.Length = uint16(len(f.Payload))
	}
	return
}

type Framer interface {
	IntoFrame() *Frame
	FromFrame(f *Frame) error
	FrameKind() FrameKind
}

type HelloFrame struct {
	Ack             bool
	CompressionGZip bool

	MaxConcurrentStreams uint32
}

func (*HelloFrame) FrameKind() FrameKind { return FrameKindHello }

func (s *HelloFrame) IntoFrame() *Frame {
	flags := byte(0x00)
	if s.Ack {
		flags |= 0x01 << 0
	}
	if s.CompressionGZip {
		flags |= 0x01 << 1
	}

	return &Frame{
		FrameKind: FrameKindHello,
		Flags:     flags,
		Length:    4,
		Payload:   encodeUint32(s.MaxConcurrentStreams),
	}
}

func (s *HelloFrame) FromFrame(f *Frame) error {
	if err := f.ValidateKind(FrameKindHello, false); err != nil {
		return err
	}

	s.Ack = f.Flags&(0x01<<0) != 0
	s.CompressionGZip = f.Flags&(0x01<<1) != 0

	if f.Length != 0 && f.Length != 4 {
		return &InvalidFrameLengthError{fmt.Sprintf("invalid length %d for frame HELLO, expected either 0 or 4 bytes", f.Length)}
	}

	if f.Length != 0 {
		s.MaxConcurrentStreams = decodeUint32(f.Payload)
	}

	if s.MaxConcurrentStreams != 0 && !s.Ack {
		return &InvalidFrameError{message: "received non-ack HELLO with non-zero MaxConcurrentStreams"}
	}

	return nil
}

type PingFrame struct {
	Ack     bool
	Payload []byte
}

func (*PingFrame) FrameKind() FrameKind { return FrameKindPing }

func (p *PingFrame) IntoFrame() *Frame {
	flags := uint8(0x00)
	if p.Ack {
		flags |= 0x01 << 2
	}
	return &Frame{
		FrameKind: FrameKindPing,
		Flags:     flags,
		Length:    uint16(len(p.Payload)),
		Payload:   p.Payload,
	}
}

func (p *PingFrame) FromFrame(f *Frame) error {
	if err := f.ValidateKind(FrameKindPing, false); err != nil {
		return err
	}
	if err := f.ValidateSize(8); err != nil {
		return err
	}

	p.Ack = f.Flags&(0x01<<2) != 0
	p.Payload = f.Payload

	return nil
}

type GoAwayFrame struct {
	LastStreamID   uint32
	ErrorCode      ErrorCode
	AdditionalData []byte
}

func (*GoAwayFrame) FrameKind() FrameKind { return FrameKindGoAway }

func (g *GoAwayFrame) IntoFrame() *Frame {
	payload := bytes.Join([][]byte{
		encodeUint32(g.LastStreamID),
		encodeUint32(uint32(g.ErrorCode)),
		g.AdditionalData,
	}, nil)

	return &Frame{
		FrameKind: FrameKindGoAway,
		Length:    uint16(len(payload)),
		Payload:   payload,
	}
}

func (g *GoAwayFrame) FromFrame(f *Frame) error {
	if err := f.ValidateKind(FrameKindGoAway, false); err != nil {
		return err
	}

	if f.Length < 8 {
		return &InvalidFrameLengthError{"invalid length for frame GOAWAY: at least 8 bytes are required"}
	}

	g.LastStreamID = decodeUint32(f.Payload)
	g.ErrorCode = ErrorCode(decodeUint32(f.Payload[4:]))
	g.AdditionalData = f.Payload[8:]

	return nil
}

type MakeStreamFrame struct {
	StreamID uint32
}

func (*MakeStreamFrame) FrameKind() FrameKind { return FrameKindMakeStream }

func (r *MakeStreamFrame) IntoFrame() *Frame {
	return &Frame{
		StreamID:  r.StreamID,
		FrameKind: FrameKindMakeStream,
	}
}

func (r *MakeStreamFrame) FromFrame(f *Frame) error {
	if err := f.ValidateKind(FrameKindMakeStream, true); err != nil {
		return err
	}

	if err := f.ValidateSize(0); err != nil {
		return err
	}

	r.StreamID = f.StreamID

	return nil
}

type ResetStreamFrame struct {
	StreamID  uint32
	ErrorCode ErrorCode
}

func (*ResetStreamFrame) FrameKind() FrameKind { return FrameKindResetStream }

func (r *ResetStreamFrame) IntoFrame() *Frame {
	return &Frame{
		StreamID:  r.StreamID,
		FrameKind: FrameKindResetStream,
		Length:    4,
		Payload:   encodeUint32(uint32(r.ErrorCode)),
	}
}

func (r *ResetStreamFrame) FromFrame(f *Frame) error {
	if err := f.ValidateKind(FrameKindResetStream, true); err != nil {
		return err
	}

	if err := f.ValidateSize(4); err != nil {
		return err
	}

	r.StreamID = f.StreamID
	r.ErrorCode = ErrorCode(decodeUint32(f.Payload))

	return nil
}

type DataFrame struct {
	StreamID uint32

	EndData   bool
	EndStream bool

	Payload []byte
}

func (*DataFrame) FrameKind() FrameKind { return FrameKindData }

func (d *DataFrame) IntoFrame() *Frame {
	flags := byte(0x00)
	if d.EndStream {
		flags |= 0x01 << 0
	}
	if d.EndData {
		flags |= 0x01 << 1
	}

	return &Frame{
		StreamID:  d.StreamID,
		FrameKind: FrameKindData,
		Flags:     flags,
		Length:    uint16(len(d.Payload)),
		Payload:   d.Payload,
	}
}

func (d *DataFrame) FromFrame(f *Frame) error {
	if err := f.ValidateKind(FrameKindData, true); err != nil {
		return err
	}

	d.StreamID = f.StreamID
	d.EndStream = f.Flags&(0x01<<0) != 0
	d.EndData = f.Flags&(0x01<<1) != 0
	d.Payload = f.Payload

	return nil
}
