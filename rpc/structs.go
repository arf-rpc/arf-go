package rpc

import (
	"bytes"
	"fmt"
	proto2 "github.com/arf-rpc/arf-go/proto"
	"github.com/arf-rpc/arf-go/status"
	"io"
	"reflect"
)

type Message interface {
	FromReader(r io.Reader) error
	Kind() MessageKind
	Encode() ([]byte, error)
	Wrap() ([]byte, error)
}

func MessageFromReader(r io.Reader) (Message, error) {
	rawKind, err := decodeUint8FromReader(r)
	if err != nil {
		return nil, err
	}

	inst, err := InitializeMessageKind(MessageKindFromByte(rawKind))
	if err != nil {
		return nil, err
	}

	if err = inst.FromReader(r); err != nil {
		return nil, err
	}

	return inst, nil
}

func MessageTFromReader[T Message](r io.Reader) (msg T, err error) {
	var k MessageKind
	k, err = MessageKindFromReader(r)
	if err != nil {
		return
	}
	msg = reflect.New(reflect.TypeOf(msg).Elem()).Interface().(T)
	if k != msg.Kind() {
		err = &MessageKindMismatchError{
			Expected: msg.Kind(),
			Received: k,
		}
		return
	}
	err = msg.FromReader(r)
	return
}

func wrapMessage(m Message) ([]byte, error) {
	buf, err := m.Encode()
	if err != nil {
		return nil, err
	}
	return append([]byte{byte(m.Kind())}, buf...), nil
}

type Request struct {
	Service   string
	Method    string
	Streaming bool
	Metadata  Metadata
	Params    []any
}

func (r *Request) Encode() ([]byte, error) {
	flags := byte(0x00)
	if r.Streaming {
		flags |= 0x01 << 0x00
	}
	pLen := len(r.Params)
	params := make([][]byte, pLen)

	var err error
	for i, v := range r.Params {
		params[i], err = proto2.Encode(v)
		if err != nil {
			return nil, err
		}
	}

	data := [][]byte{
		proto2.EncodeString(r.Service),
		proto2.EncodeString(r.Method),
		{flags},
		r.Metadata.Encode(),
		encodeUint16(uint16(pLen)),
		bytes.Join(params, nil),
	}

	return bytes.Join(data, nil), nil
}

func (r *Request) FromReader(read io.Reader) error {
	var err error

	if r.Service, err = proto2.DecodeString(read); err != nil {
		return err
	}

	if r.Method, err = proto2.DecodeString(read); err != nil {
		return err
	}

	flags, err := decodeUint8FromReader(read)
	if err != nil {
		return err
	}

	r.Streaming = flags&(0x01<<0) == (0x01 << 0)

	if r.Metadata, err = MetadataFromReader(read); err != nil {
		return err
	}

	paramsLen, err := decodeUint16FromReader(read)
	if err != nil {
		return err
	}

	r.Params = make([]any, paramsLen)

	for i := range paramsLen {
		r.Params[i], err = proto2.DecodeAny(read)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Request) Kind() MessageKind { return MessageKindRequest }

func (r *Request) Wrap() ([]byte, error) { return wrapMessage(r) }

type Response struct {
	Status    uint16
	Streaming bool
	Metadata  Metadata
	Params    []any
}

func (r *Response) Encode() ([]byte, error) {
	flags := byte(0x00)
	if r.Streaming {
		flags |= 0x01 << 0x00
	}
	pLen := len(r.Params)
	params := make([][]byte, pLen)

	var err error
	for i, v := range r.Params {
		params[i], err = proto2.Encode(v)
		if err != nil {
			return nil, err
		}
	}

	data := [][]byte{
		encodeUint16(r.Status),
		{flags},
		r.Metadata.Encode(),
		encodeUint16(uint16(pLen)),
		bytes.Join(params, nil),
	}

	return bytes.Join(data, nil), nil
}

func (r *Response) FromReader(read io.Reader) error {
	var err error

	r.Status, err = decodeUint16FromReader(read)
	if err != nil {
		return err
	}

	flags, err := decodeUint8FromReader(read)
	if err != nil {
		return err
	}

	r.Streaming = flags&(0x01<<0) == (0x01 << 0)

	if r.Metadata, err = MetadataFromReader(read); err != nil {
		return err
	}

	paramsLen, err := decodeUint16FromReader(read)
	if err != nil {
		return err
	}

	r.Params = make([]any, paramsLen)

	for i := range paramsLen {
		r.Params[i], err = proto2.DecodeAny(read)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Response) Kind() MessageKind { return MessageKindResponse }

func (r *Response) Wrap() ([]byte, error) { return wrapMessage(r) }

func (r *Response) Result() ([]any, error) {
	if r.Status == uint16(status.OK) {
		return r.Params, nil
	}

	st := status.Status(r.Status)
	respStatus, ok := r.Metadata.LookupString("arf-status-description")
	if !ok {
		respStatus = st.Error()
	}
	return nil, &status.BadStatus{
		Code:    st,
		Message: respStatus,
	}
}

type StartStream struct{}

func (s *StartStream) FromReader(io.Reader) error { return nil }

func (s *StartStream) Kind() MessageKind { return MessageKindStartStream }

func (s *StartStream) Encode() ([]byte, error) { return nil, nil }

func (s *StartStream) Wrap() ([]byte, error) { return wrapMessage(s) }

type StreamItem struct {
	Value any
}

func (s *StreamItem) Encode() ([]byte, error) { return proto2.Encode(s.Value) }

func (s *StreamItem) FromReader(r io.Reader) error {
	val, err := proto2.DecodeAny(r)
	if err != nil {
		return err
	}
	s.Value = val
	return nil
}

func (s *StreamItem) Kind() MessageKind { return MessageKindStreamItem }

func (s *StreamItem) Wrap() ([]byte, error) { return wrapMessage(s) }

type StreamMetadata struct {
	Metadata Metadata
}

func (s *StreamMetadata) Encode() ([]byte, error) {
	return s.Metadata.Encode(), nil
}

func (s *StreamMetadata) FromReader(r io.Reader) (err error) {
	if s.Metadata, err = MetadataFromReader(r); err != nil {
		return
	}
	return
}

func (s *StreamMetadata) Kind() MessageKind { return MessageKindStreamMetadata }

func (s *StreamMetadata) Wrap() ([]byte, error) { return wrapMessage(s) }

type EndStream struct{}

func (e *EndStream) Encode() ([]byte, error) { return nil, nil }

func (e *EndStream) FromReader(io.Reader) error { return nil }

func (e *EndStream) Kind() MessageKind { return MessageKindEndStream }

func (e *EndStream) Wrap() ([]byte, error) { return wrapMessage(e) }

type StreamError struct {
	Status   uint16
	Metadata Metadata
}

func (s *StreamError) Encode() ([]byte, error) {
	data := [][]byte{
		encodeUint16(s.Status),
		s.Metadata.Encode(),
	}
	return bytes.Join(data, nil), nil
}

func (s *StreamError) FromReader(r io.Reader) error {
	var err error

	if s.Status, err = decodeUint16FromReader(r); err != nil {
		return err
	}

	if s.Metadata, err = MetadataFromReader(r); err != nil {
		return err
	}

	return nil
}

func (s *StreamError) Kind() MessageKind { return MessageKindStreamError }

func (s *StreamError) Wrap() ([]byte, error) { return wrapMessage(s) }

func (s *StreamError) Error() string {
	return fmt.Sprintf("stream error: status=%d", s.Status)
}
