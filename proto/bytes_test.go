package proto

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBytes(t *testing.T) {
	t.Run("Encode empty", func(t *testing.T) {
		b := EncodeBytes(nil)
		assert.Equal(t, []byte{byte(TypeBytes) | bytesEmptyMask}, b)
	})

	t.Run("decode empty", func(t *testing.T) {
		b := EncodeBytes(nil)

		v, err := decodeBytes(b[0], bytes.NewReader(b[1:]))
		require.NoError(t, err)
		require.Nil(t, v)
	})

	t.Run("Encode/Decode", func(t *testing.T) {
		data := []byte{0x01, 0x02, 0x03, 0x04}
		b := EncodeBytes(data)
		fmt.Println(hex.EncodeToString(b))
		v, err := decodeBytes(b[0], bytes.NewReader(b[1:]))
		require.NoError(t, err)
		assert.Equal(t, data, v)
	})
}
