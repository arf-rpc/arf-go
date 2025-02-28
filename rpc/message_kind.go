package rpc

import (
	"fmt"
	"io"
)

type MessageKind uint8

func (m MessageKind) String() string {
	if v, ok := messageNames[m]; ok {
		return v
	}
	return "UNKNOWN"
}

const (
	MessageKindInvalid        MessageKind = 0x00
	MessageKindRequest        MessageKind = 0x01
	MessageKindResponse       MessageKind = 0x02
	MessageKindStartStream    MessageKind = 0x03
	MessageKindStreamItem     MessageKind = 0x04
	MessageKindStreamMetadata MessageKind = 0x05
	MessageKindEndStream      MessageKind = 0x06
	MessageKindStreamError    MessageKind = 0x07
)

var messageNames = map[MessageKind]string{
	MessageKindInvalid:        "Invalid",
	MessageKindRequest:        "Request",
	MessageKindResponse:       "Response",
	MessageKindStartStream:    "StartStream",
	MessageKindStreamItem:     "StreamItem",
	MessageKindStreamMetadata: "StreamMetadata",
	MessageKindEndStream:      "EndStream",
	MessageKindStreamError:    "StreamError",
}

var messageKindFromByte = map[byte]MessageKind{
	byte(MessageKindInvalid):        MessageKindInvalid,
	byte(MessageKindRequest):        MessageKindRequest,
	byte(MessageKindResponse):       MessageKindResponse,
	byte(MessageKindStartStream):    MessageKindStartStream,
	byte(MessageKindStreamItem):     MessageKindStreamItem,
	byte(MessageKindStreamMetadata): MessageKindStreamMetadata,
	byte(MessageKindEndStream):      MessageKindEndStream,
	byte(MessageKindStreamError):    MessageKindStreamError,
}

func MessageKindFromByte(b byte) MessageKind {
	k, ok := messageKindFromByte[b]
	if !ok {
		return MessageKindInvalid
	}
	return k
}

func MessageKindFromReader(r io.Reader) (MessageKind, error) {
	var b [1]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return MessageKindInvalid, err
	}
	return MessageKindFromByte(b[0]), nil
}

func InitializeMessageKind(k MessageKind) (Message, error) {
	switch k {
	case MessageKindRequest:
		return &Request{}, nil
	case MessageKindResponse:
		return &Response{}, nil
	case MessageKindStartStream:
		return &StartStream{}, nil
	case MessageKindStreamItem:
		return &StreamItem{}, nil
	case MessageKindStreamMetadata:
		return &StreamMetadata{}, nil
	case MessageKindEndStream:
		return &EndStream{}, nil
	case MessageKindStreamError:
		return &StreamError{}, nil
	}

	return nil, fmt.Errorf("cannot initialize Invalid message kind")
}
