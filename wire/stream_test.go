package wire

import (
	"bytes"
	"crypto/rand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"reflect"
	"testing"
)

func mustRead(path string) []byte {
	f, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return f
}

var random = mustRead("../fixtures/random.bin")

type dummyConn struct {
	messages []*Frame
}

func (d *dummyConn) cancelStream(Stream) {
}

func (d *dummyConn) handleStream(Stream) {}

func (d *dummyConn) Next() *Frame {
	if len(d.messages) == 0 {
		return nil
	}
	msg := d.messages[0]
	d.messages = d.messages[1:]
	return msg
}

func NextAs[T Framer](t *testing.T, d *dummyConn) T {
	t.Helper()

	var v T
	reflect.ValueOf(&v).Elem().Set(reflect.New(reflect.TypeOf(v).Elem()))
	n := d.Next()
	require.NotNilf(t, n, "expected received message not to be nil")
	err := v.FromFrame(n)
	require.NoError(t, err)
	return v
}

func (d *dummyConn) Write(fr *Frame) error {
	d.messages = append(d.messages, fr)
	return nil
}

func makeStream() (*dummyConn, Stream) {
	dummy := &dummyConn{}
	return dummy, NewStream(1, dummy)
}

func TestStream(t *testing.T) {
	t.Run("open state", func(t *testing.T) {
		t.Run("receiving a reset closes the connection", func(t *testing.T) {
			d, s := makeStream()
			s.handleResetStream(&ResetStreamFrame{
				StreamID:  1,
				ErrorCode: ErrorCodeCancel,
			})

			require.Nil(t, d.Next())
			assert.Equal(t, streamStateClosed, s.(*stream).state.code)
		})

		t.Run("receiving data enqueues it for reading", func(t *testing.T) {
			d, s := makeStream()
			data := make([]byte, 32)
			_, err := rand.Read(data)
			require.NoError(t, err)

			s.handleData(&DataFrame{
				StreamID:  1,
				EndData:   true,
				EndStream: false,
				Payload:   data,
			})

			require.Nil(t, d.Next())
			assert.Equal(t, streamStateOpen, s.(*stream).state.code)

			read := make([]byte, 32)
			_, err = io.ReadFull(s, read)
			require.NoError(t, err)

			assert.Equal(t, data, read)
		})

		t.Run("receiving data with END_DATA allows extra data to be sent", func(t *testing.T) {
			d, s := makeStream()
			data1 := make([]byte, 32)
			_, err := rand.Read(data1)
			require.NoError(t, err)

			data2 := make([]byte, 32)
			_, err = rand.Read(data2)
			require.NoError(t, err)

			s.handleData(&DataFrame{
				StreamID:  1,
				EndData:   false,
				EndStream: false,
				Payload:   data1,
			})

			require.Nil(t, d.Next())
			assert.Equal(t, streamStateOpen, s.(*stream).state.code)

			s.handleData(&DataFrame{
				StreamID:  1,
				EndData:   true,
				EndStream: false,
				Payload:   data2,
			})

			read := make([]byte, 64)
			_, err = io.ReadFull(s, read)
			require.NoError(t, err)

			assert.Equal(t, bytes.Join([][]byte{data1, data2}, nil), read)

			require.Nil(t, d.Next())
			assert.Equal(t, streamStateOpen, s.(*stream).state.code)
		})

		t.Run("receiving data with END_STREAM causes a half-close", func(t *testing.T) {
			d, s := makeStream()
			data := make([]byte, 32)
			_, err := rand.Read(data)
			require.NoError(t, err)

			s.handleData(&DataFrame{
				StreamID:  1,
				EndData:   true,
				EndStream: true,
				Payload:   data,
			})

			require.Nil(t, d.Next())
			assert.Equal(t, streamStateHalfClosedRemote, s.(*stream).state.code)
		})

		t.Run("calling Write issues a DataFrame", func(t *testing.T) {
			d, s := makeStream()
			err := s.Write([]byte("hello"), false)
			require.NoError(t, err)

			data := NextAs[*DataFrame](t, d)

			assert.Equal(t, []byte("hello"), data.Payload)
			assert.Equal(t, uint32(1), data.StreamID)
			assert.True(t, data.EndData)
			assert.False(t, data.EndStream)
		})

		t.Run("calling Write with END_STREAM causes a local half-close", func(t *testing.T) {
			_, s := makeStream()
			err := s.Write([]byte("hello"), true)
			require.NoError(t, err)

			assert.Equal(t, streamStateHalfClosedLocal, s.(*stream).state.code)
		})

		t.Run("calling CloseLocal issues an empty DATA frame", func(t *testing.T) {
			d, s := makeStream()
			err := s.CloseLocal()
			require.NoError(t, err)

			data := NextAs[*DataFrame](t, d)

			assert.Empty(t, data.Payload)
			assert.Equal(t, uint32(1), data.StreamID)
			assert.True(t, data.EndStream)

			assert.Equal(t, streamStateHalfClosedLocal, s.(*stream).state.code)
		})

		t.Run("having both sides closed closes the connection", func(t *testing.T) {
			d, s := makeStream()
			s.handleData(&DataFrame{
				StreamID:  1,
				EndStream: true,
				Payload:   nil,
			})

			assert.Equal(t, streamStateHalfClosedRemote, s.(*stream).state.code)

			err := s.CloseLocal()
			require.NoError(t, err)

			data := NextAs[*DataFrame](t, d)

			assert.Empty(t, data.Payload)
			assert.Equal(t, uint32(1), data.StreamID)
			assert.True(t, data.EndStream)

			assert.Equal(t, streamStateClosed, s.(*stream).state.code)

		})

		t.Run("large data is correctly segmented", func(t *testing.T) {
			d, s := makeStream()

			err := s.Write(random, false)
			require.NoError(t, err)

			assert.Equal(t, streamStateOpen, s.(*stream).state.code)

			d1 := NextAs[*DataFrame](t, d)

			d2 := NextAs[*DataFrame](t, d)

			assert.Equal(t, uint32(1), d1.StreamID)
			assert.False(t, d1.EndStream)
			assert.False(t, d1.EndData)
			assert.Equal(t, maxPayload, len(d1.Payload))

			assert.Equal(t, uint32(1), d2.StreamID)
			assert.False(t, d2.EndStream)
			assert.True(t, d2.EndData)
			assert.Equal(t, len(random)-maxPayload, len(d2.Payload))
		})
	})

	t.Run("closed state", func(t *testing.T) {
		t.Run("calling Write returns an ClosedStreamErr", func(t *testing.T) {
			_, s := makeStream()
			err := s.Reset(ErrorCodeCancel)
			require.NoError(t, err)

			err = s.Write([]byte("hello"), false)
			assert.ErrorIs(t, ClosedStreamErr, err)
		})

		t.Run("calling WriteHeaders returns an ClosedStreamErr", func(t *testing.T) {
			_, s := makeStream()
			err := s.Reset(ErrorCodeCancel)
			require.NoError(t, err)

			err = s.Write([]byte("hello"), false)
			assert.ErrorIs(t, ClosedStreamErr, err)
		})

		t.Run("calling CloseLocal returns an ClosedStreamErr", func(t *testing.T) {
			_, s := makeStream()
			err := s.Reset(ErrorCodeCancel)
			require.NoError(t, err)

			err = s.CloseLocal()
			assert.ErrorIs(t, ClosedStreamErr, err)
		})

		t.Run("receiving Reset returns ErrorCodeStreamClosed", func(t *testing.T) {
			d, s := makeStream()
			err := s.Reset(ErrorCodeCancel)
			require.NoError(t, err)

			d.Next() // consume ResetStreamFrame from previous reset call

			s.handleResetStream(&ResetStreamFrame{
				StreamID:  1,
				ErrorCode: ErrorCodeCancel,
			})

			r := NextAs[*ResetStreamFrame](t, d)
			assert.Equal(t, ErrorCodeStreamClosed, r.ErrorCode)
		})

		t.Run("receiving Data returns ErrorCodeStreamClosed", func(t *testing.T) {
			d, s := makeStream()
			err := s.Reset(ErrorCodeCancel)
			require.NoError(t, err)
			d.Next() // consume ResetStreamFrame from previous reset call

			s.handleData(&DataFrame{
				StreamID:  1,
				EndData:   true,
				EndStream: false,
				Payload:   []byte{1, 2, 3},
			})

			r := NextAs[*ResetStreamFrame](t, d)
			assert.Equal(t, ErrorCodeStreamClosed, r.ErrorCode)
		})
	})

	t.Run("remote reset", func(t *testing.T) {
		t.Run("calling Write returns a StreamResetError", func(t *testing.T) {
			_, s := makeStream()

			s.handleResetStream(&ResetStreamFrame{
				StreamID:  1,
				ErrorCode: ErrorCodeCancel,
			})

			err := s.Write([]byte("hello"), false)
			assert.Equal(t, &StreamResetError{ErrorCodeCancel}, err)
		})

		t.Run("calling Read returns a StreamResetError", func(t *testing.T) {
			_, s := makeStream()
			s.handleResetStream(&ResetStreamFrame{
				StreamID:  1,
				ErrorCode: ErrorCodeCancel,
			})

			_, err := s.Read([]byte{})
			assert.Equal(t, &StreamResetError{ErrorCodeCancel}, err)

		})

		t.Run("calling CloseLocal returns a StreamResetError", func(t *testing.T) {
			_, s := makeStream()

			s.handleResetStream(&ResetStreamFrame{
				StreamID:  1,
				ErrorCode: ErrorCodeCancel,
			})

			err := s.CloseLocal()
			assert.Equal(t, &StreamResetError{ErrorCodeCancel}, err)
		})
	})

	t.Run("half-closed remote state", func(t *testing.T) {
		t.Run("it accepts incoming ResetStreamFrame", func(t *testing.T) {
			_, s := makeStream()
			s.handleData(&DataFrame{
				StreamID:  1,
				EndStream: true,
			})

			s.handleResetStream(&ResetStreamFrame{
				StreamID:  1,
				ErrorCode: ErrorCodeCancel,
			})

			assert.Equal(t, streamStateClosed, s.(*stream).state.code)
		})

		t.Run("it ignores incoming DataFrame", func(t *testing.T) {
			_, s := makeStream()
			s.handleData(&DataFrame{
				StreamID:  1,
				EndStream: true,
			})

			s.handleData(&DataFrame{
				StreamID:  1,
				EndData:   true,
				EndStream: false,
				Payload:   []byte{0x00, 0x01, 0x02},
			})

			select {
			case <-s.(*stream).reader.blocks:
				assert.Fail(t, "Expected data to have been dropped")
			default:
			}

		})
	})

	t.Run("half-closed local state", func(t *testing.T) {
		t.Run("receiving DATA with END_STREAM transitions to closed", func(t *testing.T) {
			_, s := makeStream()
			err := s.CloseLocal()
			require.NoError(t, err)

			assert.Equal(t, streamStateHalfClosedLocal, s.(*stream).state.code)

			s.handleData(&DataFrame{
				StreamID:  1,
				EndData:   true,
				EndStream: true,
				Payload:   []byte{0x00, 0x01, 0x02},
			})
			assert.Equal(t, streamStateClosed, s.(*stream).state.code)
		})
	})
}
