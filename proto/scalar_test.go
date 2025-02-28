package proto

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)
import "github.com/stretchr/testify/assert"

func TestScalar(t *testing.T) {
	t.Run("zero uint", func(t *testing.T) {
		b := encodeScalar(uint8(0))
		assert.Equal(t, []byte{0x21}, b)

		pt, bv, err := readType(bytes.NewReader(b))
		require.NoError(t, err)
		assert.Equal(t, TypeScalar, pt)
		assert.Equal(t, byte(0x21), bv)

		s, n, v, err := decodeScalar(b[0], bytes.NewReader([]byte{}))
		require.NoError(t, err)
		require.False(t, s)
		require.False(t, n)
		require.Zero(t, v)
	})

	t.Run("zero int", func(t *testing.T) {
		b := encodeScalar(int8(0))
		assert.Equal(t, []byte{0x31}, b)

		pt, bv, err := readType(bytes.NewReader(b))
		require.NoError(t, err)
		assert.Equal(t, TypeScalar, pt)
		assert.Equal(t, byte(0x31), bv)

		s, n, v, err := decodeScalar(b[0], bytes.NewReader([]byte{}))
		require.NoError(t, err)
		require.True(t, s)
		require.False(t, n)
		require.Zero(t, v)
	})

	t.Run("ten int", func(t *testing.T) {
		b := encodeScalar(int8(-10))
		assert.Equal(t, []byte{0x51, 0xa}, b)

		pt, bv, err := readType(bytes.NewReader(b))
		require.NoError(t, err)
		assert.Equal(t, TypeScalar, pt)
		assert.Equal(t, byte(0x51), bv)

		s, n, v, err := decodeScalar(b[0], bytes.NewReader([]byte{b[1]}))
		require.NoError(t, err)
		require.True(t, s)
		require.True(t, n)
		require.Equal(t, uint64(10), v)
	})

	t.Run("ten uint", func(t *testing.T) {
		b := encodeScalar(uint8(10))
		assert.Equal(t, []byte{0x1, 0xa}, b)

		pt, bv, err := readType(bytes.NewReader(b))
		require.NoError(t, err)
		assert.Equal(t, TypeScalar, pt)
		assert.Equal(t, byte(0x1), bv)

		s, n, v, err := decodeScalar(b[0], bytes.NewReader([]byte{b[1]}))
		require.NoError(t, err)
		require.False(t, s)
		require.False(t, n)
		require.Equal(t, uint64(10), v)
	})

	t.Run("uint scalars", func(t *testing.T) {
		for i := 0; i <= 1024; i++ {
			t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
				buf := encodeScalar(uint64(i))
				s, n, v, err := decodeScalar(buf[0], bytes.NewReader(buf[1:]))
				require.NoError(t, err)
				require.False(t, s)
				require.False(t, n)
				assert.Equal(t, uint64(i), v, "Buffer data is %#v", buf)
			})
		}
	})

	t.Run("int scalars", func(t *testing.T) {
		for i := 1024; i >= -1024; i-- {
			t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
				buf := encodeScalar(int64(i))
				s, n, v, err := decodeScalar(buf[0], bytes.NewReader(buf[1:]))
				require.NoError(t, err)
				require.True(t, s)
				expected := i
				if i >= 0 {
					require.False(t, n)
				} else {
					require.True(t, n)
					expected = -i
				}
				assert.Equal(t, uint64(expected), v, "Buffer data is %#v", buf)
			})
		}
	})
}
