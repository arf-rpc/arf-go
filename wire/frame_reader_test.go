package wire

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

func readerFrom(data ...byte) io.Reader {
	return bytes.NewReader(data)
}

func TestFrameReader(t *testing.T) {
	t.Run("decode hello", func(t *testing.T) {
		buf := readerFrom(0x61, 0x72, 0x66, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00)
		r := NewFrameReader(buf)
		f, err := r.Read()
		require.NoError(t, err)
		assert.Equal(t, uint32(0x00), f.StreamID)
		assert.Equal(t, byte(0x00), f.Flags)
		assert.Equal(t, FrameKindHello, f.FrameKind)
		assert.Equal(t, uint16(0), f.Length)
		assert.Nil(t, f.Payload)

		s := HelloFrame{}
		err = s.FromFrame(f)
		require.NoError(t, err)

		assert.Equal(t, false, s.CompressionGZip)
		assert.Equal(t, false, s.Ack)
		assert.Zero(t, s.MaxConcurrentStreams)
	})

	t.Run("decode hello ack", func(t *testing.T) {
		buf := readerFrom(0x61, 0x72, 0x66, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00)
		r := NewFrameReader(buf)
		f, err := r.Read()
		require.NoError(t, err)
		assert.Equal(t, uint32(0x00), f.StreamID)
		assert.Equal(t, byte(0x01), f.Flags)
		assert.Equal(t, FrameKindHello, f.FrameKind)
		assert.Equal(t, uint16(4), f.Length)
		assert.Equal(t, []byte{0x00, 0x00, 0x00, 0x00}, f.Payload)

		s := HelloFrame{}
		err = s.FromFrame(f)
		require.NoError(t, err)

		assert.Equal(t, false, s.CompressionGZip)
		assert.Equal(t, true, s.Ack)
		assert.Zero(t, s.MaxConcurrentStreams)
	})
}
