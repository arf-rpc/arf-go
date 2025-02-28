package proto

import (
	"io"
)

func DecodeAny(r io.Reader) (any, error) {
	t, b, err := readType(r)
	if err != nil {
		return nil, err
	}

	switch t {
	case TypeVoid:
		return nil, nil
	case TypeScalar:
		_, _, v, err := decodeScalar(b, r)
		return v, err
	case TypeBoolean:
		return decodeBoolean(b), nil
	case TypeFloat:
		_, v, err := decodeFloat(b, r)
		return v, err
	case TypeString:
		return decodeString(b, r)
	case TypeBytes:
		return decodeBytes(b, r)
	case TypeArray:
		return decodeArray(b, r)
	case TypeMap:
		return decodeMap(b, r)
	case TypeStruct:
		return decodeStruct(r)
	default:
		panic("unreachable")
	}
}
