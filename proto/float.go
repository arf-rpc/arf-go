package proto

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
)

const float64Mask = 0x01 << 4
const floatEmptyMask = 0x01 << 5

func encodeFloat32(value float32) []byte {
	header := uint8(TypeFloat)
	if value == 0 {
		return []byte{header | floatEmptyMask}
	}
	v := math.Float32bits(value)
	tmp := make([]byte, 4)
	binary.BigEndian.PutUint32(tmp, v)
	return bytes.Join([][]byte{
		{header},
		tmp,
	}, nil)
}

func encodeFloat64(value float64) []byte {
	header := uint8(TypeFloat) | float64Mask
	if value == 0 {
		return []byte{header | floatEmptyMask}
	}

	v := math.Float64bits(value)
	tmp := make([]byte, 8)
	binary.BigEndian.PutUint64(tmp, v)
	return bytes.Join([][]byte{
		{header},
		tmp,
	}, nil)
}

func decodeFloat(header byte, reader io.Reader) (bits int, value float64, err error) {
	bits = 32
	if header&float64Mask == float64Mask {
		bits = 64
	}

	if header&floatEmptyMask == floatEmptyMask {
		return
	}
	if bits == 32 {
		data := make([]byte, 4)
		if _, err := io.ReadFull(reader, data); err != nil {
			return 0, 0, err
		}
		value = float64(math.Float32frombits(binary.BigEndian.Uint32(data)))
	} else {
		data := make([]byte, 8)
		if _, err := io.ReadFull(reader, data); err != nil {
			return 0, 0, err
		}
		value = math.Float64frombits(binary.BigEndian.Uint64(data))
	}
	return
}
