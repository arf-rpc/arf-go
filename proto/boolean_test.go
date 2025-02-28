package proto

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBoolean(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		b := encodeBoolean(true)
		assert.Equal(t, []byte{0x12}, b)

		pt, bv, err := readType(bytes.NewBuffer(b))
		require.NoError(t, err)
		assert.Equal(t, TypeBoolean, pt)
		assert.Equal(t, byte(0x12), bv)

		v := decodeBoolean(b[0])
		require.True(t, v)
	})

	t.Run("false", func(t *testing.T) {
		b := encodeBoolean(false)
		assert.Equal(t, []byte{0x02}, b)

		pt, bv, err := readType(bytes.NewBuffer(b))
		require.NoError(t, err)
		assert.Equal(t, TypeBoolean, pt)
		assert.Equal(t, byte(0x02), bv)

		v := decodeBoolean(b[0])
		require.False(t, v)
	})
}
