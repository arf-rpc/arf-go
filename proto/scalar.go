package proto

import (
	"io"
)

type Numeric interface {
	uint8 | uint16 | uint32 | uint64 | int8 | int16 | int32 | int64
}

const numericSignedMask byte = 0x01 << 4
const numericZeroMask byte = 0x01 << 5
const numericNegativeMask byte = 0x01 << 6

func encodeUint64(x uint64) []byte {
	var buf []byte
	for x >= 0x80 {
		buf = append(buf, byte(x)|0x80)
		x >>= 7
	}
	return append(buf, byte(x))
}

func decodeUint64(r io.Reader) (value uint64, err error) {
	var x uint64
	var s uint
	buf := []byte{0x00}
	for {
		if _, err = r.Read(buf); err != nil {
			return
		}
		b := buf[0]
		if b < 0x80 {
			return x | uint64(b)<<s, nil
		}
		x |= uint64(b&0x7f) << s
		s += 7
	}
}

func encodeScalar[T Numeric](v T) []byte {
	anyV := any(v)
	typeByte := byte(TypeScalar)

	switch anyV.(type) {
	case int8, int16, int32, int64:
		typeByte |= numericSignedMask
	}

	switch anyV.(type) {
	case int8:
		if v < 0 {
			typeByte |= numericNegativeMask
			v *= any(int8(-1)).(T)
		}
	case int16:
		if v < 0 {
			typeByte |= numericNegativeMask
			v *= any(int16(-1)).(T)
		}
	case int32:
		if v < 0 {
			typeByte |= numericNegativeMask
			v *= any(int32(-1)).(T)
		}
	case int64:
		if v < 0 {
			typeByte |= numericNegativeMask
			v *= any(int64(-1)).(T)
		}
	}

	if v == 0 {
		typeByte |= numericZeroMask
		return []byte{typeByte}
	}

	data := encodeUint64(uint64(v))
	return append([]byte{typeByte}, data...)
}

func decodeScalar(header byte, reader io.Reader) (signed bool, negative bool, value uint64, err error) {
	signed = header&numericSignedMask == numericSignedMask
	negative = header&numericNegativeMask == numericNegativeMask
	if header&numericZeroMask == numericZeroMask {
		return
	}

	value, err = decodeUint64(reader)
	return
}
