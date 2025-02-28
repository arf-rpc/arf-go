package proto

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestArray(t *testing.T) {
	t.Run("Encode", func(t *testing.T) {
		b, err := Encode([]int64{1, 2, 3})
		require.NoError(t, err)
		require.Equal(t, []byte{
			0x6, 0x3, 0x11, 0x1, 0x11, 0x2, 0x11, 0x3,
		}, b)
	})

	t.Run("decode", func(t *testing.T) {
		b, err := Encode([]uint64{1, 2, 3})
		require.NoError(t, err)

		arr, err := decodeArray(b[0], bytes.NewReader(b[1:]))
		require.NoError(t, err)
		require.NotEmpty(t, arr)

		assert.Equal(t, uint64(1), arr[0])
		assert.Equal(t, uint64(2), arr[1])
		assert.Equal(t, uint64(3), arr[2])
	})

	t.Run("Encode empty", func(t *testing.T) {
		b, err := Encode([]uint16{})
		require.NoError(t, err)
		require.Equal(t, []byte{0x16}, b)
	})

	t.Run("decode empty", func(t *testing.T) {
		data := []byte{0x16}
		arr, err := decodeArray(data[0], bytes.NewReader(data[1:]))
		require.NoError(t, err)
		require.Empty(t, arr)
	})
}
