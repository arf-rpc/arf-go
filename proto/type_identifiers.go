package proto

import (
	"fmt"
	"io"
)

//go:generate stringer -type=PrimitiveType -output=type_identifiers_string.go

type PrimitiveType byte

const (
	TypeVoid    PrimitiveType = 0b0000
	TypeScalar  PrimitiveType = 0b0001
	TypeBoolean PrimitiveType = 0b0010
	TypeFloat   PrimitiveType = 0b0011
	TypeString  PrimitiveType = 0b0100
	TypeBytes   PrimitiveType = 0b0101
	TypeArray   PrimitiveType = 0b0110
	TypeMap     PrimitiveType = 0b0111
	TypeStruct  PrimitiveType = 0b1000
)

var allPrimitives = map[PrimitiveType]bool{
	TypeVoid:    true,
	TypeScalar:  true,
	TypeBoolean: true,
	TypeFloat:   true,
	TypeString:  true,
	TypeBytes:   true,
	TypeArray:   true,
	TypeMap:     true,
	TypeStruct:  true,
}

func readType(r io.Reader) (PrimitiveType, byte, error) {
	buf := []byte{0}
	_, err := r.Read(buf)
	if err != nil {
		return 0, 0, err
	}

	decoded := PrimitiveType(buf[0] & 0xF)
	if _, ok := allPrimitives[decoded]; ok {
		return decoded, buf[0], nil
	}

	return 0, 0, fmt.Errorf("unknown type 0x%02x", decoded)
}
