package wire

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
	"slices"
	"testing"
)

func frameFromBytes(t *testing.T, data []byte) *Frame {
	t.Helper()

	b := bytes.NewReader(data)
	fr := NewFrameReader(b)
	f, err := fr.Read()
	require.NoError(t, err)
	return f
}

func incompatibleFrameFor(t *testing.T, f *Frame) Framer {
	t.Helper()

	knownFrames := []FrameKind{
		FrameKindHello, FrameKindPing, FrameKindGoAway,
		FrameKindMakeStream, FrameKindResetStream, FrameKindData,
	}
	incompatible := []FrameKind{
		FrameKindData, FrameKindHello, FrameKindPing, FrameKindGoAway,
		FrameKindMakeStream, FrameKindResetStream,
	}

	inc := incompatible[slices.Index(knownFrames, f.FrameKind)]
	switch inc {
	case FrameKindHello:
		return &HelloFrame{}
	case FrameKindPing:
		return &PingFrame{}
	case FrameKindGoAway:
		return &GoAwayFrame{}
	case FrameKindMakeStream:
		return &MakeStreamFrame{}
	case FrameKindResetStream:
		return &ResetStreamFrame{}
	case FrameKindData:
		return &DataFrame{}
	default:
		panic("Unknown frame type " + inc.String())
	}
}

func doRoundTrip[T Framer](t *testing.T, m CompressionMethod, fr T) (recovered T, err error) {
	tType := reflect.TypeOf(fr).Elem()

	data := fr.IntoFrame().Bytes(m)
	rawFrame := frameFromBytes(t, data)
	err = rawFrame.Decompress(m)
	require.NoError(t, err)

	recovered = reflect.New(tType).Interface().(T)
	err = recovered.FromFrame(rawFrame)
	return
}

func testFrameRoundTrip(t *testing.T, associated bool, fr Framer) {
	tType := reflect.TypeOf(fr).Elem()

	for _, m := range allCompressionMethods {
		t.Run(fmt.Sprintf("can be transferred using %s as compression", m), func(t *testing.T) {
			recovered, err := doRoundTrip(t, m, fr)
			require.NoError(t, err)
			assert.Equal(t, fr, recovered)
		})

		t.Run("correctly rejects incompatible frames", func(t *testing.T) {
			incompatibleFrame := incompatibleFrameFor(t, fr.IntoFrame()).IntoFrame()

			recovered := reflect.New(tType).Interface().(Framer)
			err := recovered.FromFrame(incompatibleFrame)
			require.Error(t, err)
			require.ErrorContains(t, err, "frame type mismatch")
		})
	}

	if associated {
		t.Run("requires an associated stream", func(t *testing.T) {
			brokenFr := fr.IntoFrame()
			brokenFr.StreamID = 0
			data := brokenFr.Bytes(CompressionMethodNone)
			rawFrame := frameFromBytes(t, data)
			err := rawFrame.Decompress(CompressionMethodNone)
			require.NoError(t, err)

			recovered := reflect.New(tType).Interface().(Framer)
			err = recovered.FromFrame(rawFrame)
			require.Error(t, err)
			require.ErrorContains(t, err, "must be associated")
		})
	} else {
		t.Run("rejects an associated stream", func(t *testing.T) {
			brokenFr := fr.IntoFrame()
			brokenFr.StreamID = randomStreamID(t)
			data := brokenFr.Bytes(CompressionMethodNone)
			rawFrame := frameFromBytes(t, data)
			err := rawFrame.Decompress(CompressionMethodNone)
			require.NoError(t, err)

			recovered := reflect.New(tType).Interface().(Framer)
			err = recovered.FromFrame(rawFrame)
			require.Error(t, err)
			require.ErrorContains(t, err, "must not be associated")
		})
	}
}

func randomBytes(t *testing.T, n int) []byte {
	t.Helper()

	buf := make([]byte, n)
	_, err := rand.Read(buf)
	require.NoError(t, err)
	return buf
}

func randomStreamID(t *testing.T) uint32 {
	t.Helper()
	return binary.LittleEndian.Uint32(randomBytes(t, 4))
}

func TestFrame(t *testing.T) {
	t.Run("DataFrame", func(t *testing.T) {
		testFrameRoundTrip(t, true, &DataFrame{
			StreamID:  randomStreamID(t),
			EndData:   true,
			EndStream: true,
			Payload:   []byte{0x01, 0x02, 0x03},
		})
	})

	t.Run("MakeStreamFrame", func(t *testing.T) {
		testFrameRoundTrip(t, true, &MakeStreamFrame{
			StreamID: randomStreamID(t),
		})
	})

	t.Run("ResetStreamFrame", func(t *testing.T) {
		testFrameRoundTrip(t, true, &ResetStreamFrame{
			StreamID:  randomStreamID(t),
			ErrorCode: ErrorCodeCancel,
		})

		t.Run("rejects frames with invalid size", func(t *testing.T) {
			fr := &Frame{
				StreamID:  randomStreamID(t),
				FrameKind: FrameKindResetStream,
				Flags:     0,
				Length:    1,
				Payload:   []byte{0x01},
			}
			rst := &ResetStreamFrame{}
			err := rst.FromFrame(fr)
			require.Error(t, err)
			require.ErrorContains(t, err, "invalid length for frame RESET_STREAM")
		})
	})

	t.Run("PingFrame", func(t *testing.T) {
		testFrameRoundTrip(t, false, &PingFrame{
			Ack:     true,
			Payload: randomBytes(t, 8),
		})

		t.Run("rejects frames with invalid size", func(t *testing.T) {
			fr := &Frame{
				StreamID:  0x00,
				FrameKind: FrameKindPing,
				Flags:     0,
				Length:    1,
				Payload:   []byte{0x01},
			}
			png := &PingFrame{}
			err := png.FromFrame(fr)
			require.Error(t, err)
			require.ErrorContains(t, err, "invalid length for frame PING")
		})
	})

	t.Run("GoAwayFrame", func(t *testing.T) {
		testFrameRoundTrip(t, false, &GoAwayFrame{
			LastStreamID:   randomStreamID(t),
			ErrorCode:      ErrorCodeCancel,
			AdditionalData: randomBytes(t, 8),
		})

		t.Run("rejects frames with invalid size", func(t *testing.T) {
			fr := &Frame{
				StreamID:  0x00,
				FrameKind: FrameKindGoAway,
				Flags:     0,
				Length:    1,
				Payload:   []byte{0x01},
			}
			goAway := &GoAwayFrame{}
			err := goAway.FromFrame(fr)
			require.Error(t, err)
			require.ErrorContains(t, err, "invalid length for frame GOAWAY")
		})
	})

	t.Run("HelloFrame", func(t *testing.T) {
		testFrameRoundTrip(t, false, &HelloFrame{
			CompressionGZip:      true,
			Ack:                  true,
			MaxConcurrentStreams: randomStreamID(t),
		})

		t.Run("rejects frames with invalid size", func(t *testing.T) {
			fr := &Frame{
				StreamID:  0x00,
				FrameKind: FrameKindHello,
				Flags:     0,
				Length:    1,
				Payload:   []byte{0x01},
			}
			set := &HelloFrame{}
			err := set.FromFrame(fr)
			require.Error(t, err)
			require.ErrorContains(t, err, "invalid length 1 for frame HELLO, expected either 0 or 4 bytes")
		})

		t.Run("rejects frames without ack flag and max concurrent streams", func(t *testing.T) {
			_, err := doRoundTrip(t, CompressionMethodNone, &HelloFrame{
				CompressionGZip:      false,
				Ack:                  false,
				MaxConcurrentStreams: 1,
			})
			require.Error(t, err)
			require.ErrorContains(t, err, "received non-ack HELLO")
		})
	})
}
