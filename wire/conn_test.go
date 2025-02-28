package wire

import (
	"bytes"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net"
	"testing"
	"time"
)

func makeConnection() (Client, conn, chan error) {
	local, remote := net.Pipe()
	cli := NewClient(local)

	conn := NewConn(nil, 1, remote)
	ch := make(chan error, 1)
	go func() {
		close(ch)
	}()

	return cli, conn, ch
}

func waitFor(t *testing.T, name string, duration time.Duration, assertion func() bool) {
	t.Helper()

	timeout := time.After(duration)
	done := make(chan struct{})

	loop := true
	defer func() { loop = false }()
	go func() {
		defer close(done)
		for loop {
			time.Sleep(10 * time.Millisecond)
			if assertion() {
				return
			}
		}
	}()

	select {
	case <-timeout:
		t.Fatalf("timed out waiting for %s", name)
	case <-done:
	}
}

func waitForConnectionReset(t *testing.T, cli Client) {
	t.Helper()

	waitFor(t, "connection to be reset", 3*time.Second, func() bool {
		return cli.(*client).err != nil
	})
	err := cli.(*client).err
	assert.IsType(t, &ConnectionResetError{}, err)
	var reset *ConnectionResetError
	assert.True(t, errors.As(err, &reset))
}

func waitConnectionShutdown(t *testing.T, conn conn, errch chan error) {
	t.Helper()

	timeout := time.After(3 * time.Second)
	done := make(chan struct{})

	loop := true
	defer func() { loop = false }()
	go func() {
		defer close(done)
		for loop {
			time.Sleep(10 * time.Millisecond)
			if !conn.(*Conn).running {
				return
			}
		}
	}()

	select {
	case <-timeout:
		t.Fatal("timed out waiting for connection shutdown")
		return
	case <-done:
	}

	// In case the following starts causing flaky tests, it would be a good idea
	// to poll it for a while before failing.
	select {
	case err := <-errch:
		assert.NoError(t, err, "expected conn.Service routine not to have returned an error")
	default:
		t.Fatalf("expected errch to have a nil error, but it was empty")
	}
}

func TestConn(t *testing.T) {
	t.Run("trying to exchange data without a HELLO frame causes a GOAWAY", func(t *testing.T) {
		cli, conn, errch := makeConnection()
		err := cli.Write((&DataFrame{
			StreamID:  0,
			EndData:   false,
			EndStream: false,
			Payload:   nil,
		}).IntoFrame())
		require.NoError(t, err)

		waitForConnectionReset(t, cli)
		waitConnectionShutdown(t, conn, errch)
	})

	t.Run("sending a GOAWAY terminates the connection", func(t *testing.T) {
		cli, conn, errch := makeConnection()

		err := cli.Configure(CompressionMethodNone)
		require.NoError(t, err)

		err = cli.Write((&GoAwayFrame{
			LastStreamID:   0,
			ErrorCode:      0,
			AdditionalData: nil,
		}).IntoFrame())
		require.NoError(t, err)

		err = cli.Close()
		require.NoError(t, err)
		waitConnectionShutdown(t, conn, errch)
	})

	t.Run("sending an ill-encoded frame causes a GOAWAY", func(t *testing.T) {
		cli, conn, errch := makeConnection()

		f := &GoAwayFrame{
			LastStreamID:   0,
			ErrorCode:      0,
			AdditionalData: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F},
		}
		illFr := f.IntoFrame()
		illFr.FrameKind = FrameKindHello
		_, err := io.Copy(cli.(*client).io, bytes.NewReader(illFr.Bytes(CompressionMethodNone)))
		require.NoError(t, err)

		waitForConnectionReset(t, cli)
		waitConnectionShutdown(t, conn, errch)
	})

	t.Run("sending DATA to a stream without HEADERS causes it to be reset", func(t *testing.T) {
		cli, conn, errch := makeConnection()

		err := cli.Configure(CompressionMethodNone)
		require.NoError(t, err)

		err = cli.Write((&DataFrame{
			StreamID:  1,
			EndData:   true,
			EndStream: false,
			Payload:   []byte{0x01, 0x02, 0x03},
		}).IntoFrame())
		require.NoError(t, err)

		err = cli.Terminate(ErrorCodeNoError)
		require.NoError(t, err)
		waitConnectionShutdown(t, conn, errch)
	})

	t.Run("sending PING without an ack flag causes it to be echoed back", func(t *testing.T) {
		cli, conn, errch := makeConnection()

		err := cli.Configure(CompressionMethodNone)
		require.NoError(t, err)

		payload := randomBytes(t, 8)
		err = cli.Write((&PingFrame{
			Ack:     false,
			Payload: payload,
		}).IntoFrame())
		require.NoError(t, err)

		err = cli.Terminate(ErrorCodeNoError)
		require.NoError(t, err)
		waitConnectionShutdown(t, conn, errch)
	})

	t.Run("sending PING without an ack flag does nothing", func(t *testing.T) {
		cli, conn, errch := makeConnection()

		err := cli.Configure(CompressionMethodNone)
		require.NoError(t, err)

		payload := randomBytes(t, 8)
		err = cli.Write((&PingFrame{
			Ack:     true,
			Payload: payload,
		}).IntoFrame())
		require.NoError(t, err)

		err = cli.Terminate(ErrorCodeNoError)
		require.NoError(t, err)
		waitConnectionShutdown(t, conn, errch)
	})

	t.Run("sending MAKE_STREAM initializes a stream", func(t *testing.T) {
		cli, conn, errch := makeConnection()

		err := cli.Configure(CompressionMethodNone)
		require.NoError(t, err)

		err = cli.Write((&MakeStreamFrame{
			StreamID: 1,
		}).IntoFrame())

		waitFor(t, "stream to be registered", 3*time.Second, func() bool {
			_, ok := conn.(*Conn).streams[1]
			return ok
		})

		err = cli.Terminate(ErrorCodeNoError)
		require.NoError(t, err)

		waitConnectionShutdown(t, conn, errch)
	})

	t.Run("sending DATA to an initialized stream enqueues it", func(t *testing.T) {
		cli, conn, errch := makeConnection()

		err := cli.Configure(CompressionMethodNone)
		require.NoError(t, err)

		str, err := cli.NewStream()
		require.NoError(t, err)

		waitFor(t, "stream to be registered", 3*time.Second, func() bool {
			_, ok := conn.(*Conn).streams[1]
			return ok
		})

		err = str.Write([]byte{0x01, 0x02, 0x03}, false)
		require.NoError(t, err)

		buf := make([]byte, 3)
		n, err := conn.(*Conn).streams[1].Read(buf)
		require.NoError(t, err)
		assert.Equal(t, 3, n)
		assert.Equal(t, []byte{0x01, 0x02, 0x03}, buf)

		err = cli.Terminate(ErrorCodeNoError)
		require.NoError(t, err)

		waitConnectionShutdown(t, conn, errch)
	})

	t.Run("sending RESET_STREAM to an initialized stream resets it", func(t *testing.T) {
		cli, conn, errch := makeConnection()

		err := cli.Configure(CompressionMethodNone)
		require.NoError(t, err)

		str, err := cli.NewStream()
		require.NoError(t, err)

		waitFor(t, "stream to be registered", 3*time.Second, func() bool {
			_, ok := conn.(*Conn).streams[1]
			return ok
		})

		err = str.Reset(ErrorCodeNoError)
		require.NoError(t, err)

		waitFor(t, "stream to be reset", 3*time.Second, func() bool {
			return conn.(*Conn).streams[1].(*stream).state.code == streamStateClosed
		})

		err = cli.Terminate(ErrorCodeNoError)
		require.NoError(t, err)

		waitConnectionShutdown(t, conn, errch)
	})

	t.Run("sending unknown frames without a stream causes a GOAWAY", func(t *testing.T) {
		cli, conn, errch := makeConnection()

		err := cli.Configure(CompressionMethodNone)
		require.NoError(t, err)

		data := &Frame{
			StreamID:  0,
			FrameKind: 0xFF,
			Flags:     0,
			Length:    0,
			Payload:   nil,
		}
		buf := bytes.NewReader(data.Bytes(CompressionMethodNone))
		_, err = io.Copy(cli.(*client).io, buf)
		require.NoError(t, err)

		waitFor(t, "connection to be dropped", 3*time.Second, func() bool {
			return !conn.(*Conn).running
		})
		waitFor(t, "client to be stopped", 3*time.Second, func() bool {
			return !cli.(*client).running
		})

		waitConnectionShutdown(t, conn, errch)
	})

	t.Run("sending unknown frames to a stream causes a reset", func(t *testing.T) {

	})
}
