package proto

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMap(t *testing.T) {
	theMap := map[string]string{
		`en`: `Hello, arf!`,
		`ja`: `こんにちは、arf！`,
		`it`: `Ciao, arf!`,
	}

	theComplexMap := map[string]SubStruct{
		`en`: {`Hello, arf!`},
		`ja`: {`こんにちは、arf！`},
		`it`: {`Ciao, arf!`},
	}

	t.Run("Encode/decode", func(t *testing.T) {
		encoded, err := Encode(theMap)
		require.NoError(t, err)
		decoded, err := decodeMap(encoded[0], bytes.NewReader(encoded[1:]))
		require.NoError(t, err)
		require.NotNil(t, decoded)
	})

	t.Run("Encode/decode complex", func(t *testing.T) {
		resetRegistry()
		RegisterMessage(SubStruct{})

		encoded, err := Encode(theComplexMap)
		require.NoError(t, err)
		decoded, err := decodeMap(encoded[0], bytes.NewReader(encoded[1:]))
		require.NoError(t, err)
		require.NotNil(t, decoded)
	})
}
