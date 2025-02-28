package wire

import (
	"bytes"
	"fmt"
	"io"
	"slices"
)

type UnknownFrameKindError struct {
	ReceivedKind byte
}

func (e *UnknownFrameKindError) Error() string {
	return fmt.Sprintf("FrameReader: unknown frame kind 0x%02x", e.ReceivedKind)
}

type FrameReader struct {
	tmp []byte
	r   io.Reader
}

func NewFrameReader(r io.Reader) *FrameReader {
	return &FrameReader{
		tmp: make([]byte, 4),
		r:   r,
	}
}

func (f *FrameReader) readSize(n int) ([]byte, error) {
	if n == 0 {
		return nil, nil
	}

	f.tmp = slices.Grow(f.tmp, n)
	if _, err := io.ReadFull(f.r, f.tmp[:n]); err != nil {
		return nil, err
	}

	return f.tmp[:n], nil
}

var streamIDOffset = magicLen
var frameKindOffset = streamIDOffset + 4
var flagsOffset = frameKindOffset + 1
var lengthOffset = flagsOffset + 1
var payloadOffset = lengthOffset + 2

func (f *FrameReader) Read() (*Frame, error) {
	buf, err := f.readSize(magicLen + 4 + 1 + 1 + 2)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(magic, buf[:magicLen]) {
		return nil, fmt.Errorf("magic number mismatch")
	}

	fk := buf[frameKindOffset:flagsOffset][0]
	if _, ok := frameFromByte[fk]; !ok {
		return nil, &UnknownFrameKindError{fk}
	}

	fr := &Frame{
		StreamID:  decodeUint32(buf[streamIDOffset:frameKindOffset]),
		FrameKind: FrameKind(fk),
		Flags:     buf[flagsOffset:lengthOffset][0],
		Length:    decodeUint16(buf[lengthOffset:payloadOffset]),
	}

	payload, err := f.readSize(int(fr.Length))
	if err != nil {
		return nil, err
	}
	if payload != nil {
		fr.Payload = bytes.Clone(payload)
	}
	return fr, nil
}
