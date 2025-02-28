package proto

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

const stringEmptyMask byte = 0x01 << 4

func EncodeString(s string) []byte {
	header := byte(TypeString)
	strLen := len(s)
	if strLen == 0 {
		return []byte{header | stringEmptyMask}
	}

	encodedLen := encodeUint64(uint64(strLen))
	return bytes.Join([][]byte{
		{header},
		encodedLen,
		[]byte(s),
	}, []byte{})
}

func decodeString(header byte, b io.Reader) (string, error) {
	if header&stringEmptyMask == stringEmptyMask {
		return "", nil
	}

	v, err := decodeUint64(b)
	if err != nil {
		return "", err
	}

	strBytes := make([]byte, int(v))
	if _, err := b.Read(strBytes); err != nil {
		return "", err
	}

	return strings.Clone(string(strBytes)), nil
}

func DecodeString(r io.Reader) (string, error) {
	pt, b, err := readType(r)
	if err != nil {
		return "", err
	}
	if pt != TypeString {
		return "", fmt.Errorf("expected type '%s', got '%s'", TypeString, pt)
	}

	return decodeString(b, r)
}
