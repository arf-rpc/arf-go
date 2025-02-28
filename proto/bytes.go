package proto

import (
	"bytes"
	"fmt"
	"io"
)

const bytesEmptyMask = byte(0x01) << 4

func EncodeBytes(b []byte) []byte {
	bLen := len(b)
	if bLen == 0 {
		return []byte{byte(TypeBytes) | bytesEmptyMask}
	}

	return bytes.Join([][]byte{
		{byte(TypeBytes)},
		encodeUint64(uint64(bLen)),
		b,
	}, nil)
}

func decodeBytes(header byte, r io.Reader) ([]byte, error) {
	if header&bytesEmptyMask == bytesEmptyMask {
		return nil, nil
	}

	size, err := decodeUint64(r)
	if err != nil {
		return nil, err
	}

	data := make([]byte, size)
	if _, err = r.Read(data); err != nil {
		return nil, err
	}

	return data, nil
}

func DecodeBytes(r io.Reader) ([]byte, error) {
	pt, b, err := readType(r)
	if err != nil {
		return nil, err
	}
	if pt != TypeBytes {
		return nil, fmt.Errorf("expected type '%s', got '%s'", TypeBytes, pt)
	}

	return decodeBytes(b, r)
}
