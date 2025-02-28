package proto

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestString(t *testing.T) {
	str := "こんにちは、arf！"
	encodedStr := []byte{0x04, 0x18, 0xe3, 0x81, 0x93, 0xe3, 0x82, 0x93, 0xe3, 0x81, 0xab, 0xe3, 0x81, 0xa1, 0xe3, 0x81, 0xaf, 0xe3, 0x80, 0x81, 0x61, 0x72, 0x66, 0xef, 0xbc, 0x81}

	t.Run("Encode empty", func(t *testing.T) {
		b := EncodeString("")
		assert.Equal(t, []byte{byte(TypeString) | stringEmptyMask}, b)
	})

	t.Run("Encode", func(t *testing.T) {
		b := EncodeString(str)
		assert.Equal(t, byte(TypeString), b[0])
		assert.Equal(t, encodedStr, b)
	})

	t.Run("Decode", func(t *testing.T) {
		b := EncodeString(str)
		dec, err := decodeString(b[0], bytes.NewReader(b[1:]))
		require.NoError(t, err)
		assert.Equal(t, str, dec)
	})
}
