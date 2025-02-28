package rpc

import (
	"encoding/binary"
	"io"
)

var encodeUint16 = func(v uint16) []byte {
	var b = make([]byte, 2)
	binary.BigEndian.PutUint16(b, v)
	return b
}

var decodeUint16 = binary.BigEndian.Uint16

func decodeUint16FromReader(r io.Reader) (uint16, error) {
	var b = make([]byte, 2)
	if _, err := io.ReadFull(r, b); err != nil {
		return 0, err
	}
	return decodeUint16(b), nil
}

func decodeUint8FromReader(r io.Reader) (byte, error) {
	b := []byte{0x00}
	if _, err := io.ReadFull(r, b); err != nil {
		return 0, err
	}
	return b[0], nil
}
