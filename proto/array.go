package proto

import (
	"bytes"
	"io"
	"reflect"
)

const arrayEmptyMask = byte(0x01) << 4

func encodeArray(v reflect.Value) ([]byte, error) {
	if v.Len() == 0 {
		return []byte{byte(TypeArray) | arrayEmptyMask}, nil
	}

	var buffer []byte
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		data, err := Encode(item)
		if err != nil {
			return nil, err
		}
		buffer = append(buffer, data...)
	}
	return bytes.Join([][]byte{
		{byte(TypeArray)},
		encodeUint64(uint64(v.Len())),
		buffer,
	}, nil), nil
}

func decodeArray(header byte, r io.Reader) ([]any, error) {
	if header&arrayEmptyMask == arrayEmptyMask {
		return nil, nil
	}

	arrLen, err := decodeUint64(r)
	if err != nil {
		return nil, err
	}

	arr := make([]any, arrLen)
	for i := range arrLen {
		if arr[i], err = DecodeAny(r); err != nil {
			return nil, err
		}
	}

	return arr, nil
}
