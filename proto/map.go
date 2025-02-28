package proto

import (
	"bytes"
	"io"
	"reflect"
)

const emptyMapMask = 0x01 << 4

type encodedMap struct {
	keys, values []any
}

var reflectedMapValue = reflect.TypeOf(&encodedMap{})

func encodeMap(v reflect.Value) ([]byte, error) {
	pairsLen := v.Len()
	if pairsLen == 0 {
		return []byte{byte(TypeMap) | emptyMapMask}, nil
	}

	var keys, values []byte

	for _, k := range v.MapKeys() {
		key, err := Encode(k.Interface())
		if err != nil {
			return nil, err
		}
		value, err := Encode(v.MapIndex(k).Interface())
		if err != nil {
			return nil, err
		}

		keys = append(keys, key...)
		values = append(values, value...)
	}

	encodedPairsLen := encodeUint64(uint64(pairsLen))

	return bytes.Join([][]byte{
		{byte(TypeMap)},
		encodeUint64(uint64(len(keys) + len(values) + len(encodedPairsLen))),
		encodedPairsLen,
		keys,
		values,
	}, nil), nil
}

func decodeMap(header byte, r io.Reader) (*encodedMap, error) {
	if header&emptyMapMask == emptyMapMask {
		return &encodedMap{}, nil
	}

	_, err := decodeUint64(r)
	if err != nil {
		return nil, err
	}
	pairsLen, err := decodeUint64(r)
	if err != nil {
		return nil, err
	}

	keys := make([]any, pairsLen)
	values := make([]any, pairsLen)

	for i := range pairsLen {
		keys[i], err = DecodeAny(r)
		if err != nil {
			return nil, err
		}
	}

	for i := range pairsLen {
		values[i], err = DecodeAny(r)
		if err != nil {
			return nil, err
		}
	}

	return &encodedMap{keys, values}, nil
}
