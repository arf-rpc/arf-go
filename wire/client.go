package wire

import (
	"bytes"
	"io"
	"sync"
)

type Client interface {
	Configure(compression CompressionMethod) error
	Close() error
	Write(*Frame) error
	NewStream() (Stream, error)
	Terminate(reason ErrorCode) error
}

type client struct {
	writeMu       *FairMutex
	io            io.ReadWriteCloser
	toWrite       chan *outboundFrame
	signalHelloOK func()
	helloOK       chan struct{}
	drop          chan struct{}
	reader        *FrameReader

	compression CompressionMethod
	err         error

	streamsMu    sync.Mutex
	streams      map[uint32]Stream
	lastStreamID uint32

	runningMu sync.Mutex
	running   bool

	setup                bool
	maxConcurrentStreams uint32
}

func NewClient(conn io.ReadWriteCloser) Client {
	helloOk := make(chan struct{})
	c := &client{
		writeMu: NewFairMutex(),
		io:      conn,
		toWrite: make(chan *outboundFrame, 128),
		signalHelloOK: sync.OnceFunc(func() {
			close(helloOk)
		}),
		helloOK: helloOk,
		drop:    make(chan struct{}),
		reader:  NewFrameReader(conn),
		streams: make(map[uint32]Stream),
	}
	go func() {
		err := c.service()
		if c.err == nil {
			c.err = err
		}
	}()

	return c
}

func (c *client) Configure(compression CompressionMethod) error {
	err := c.Write((&HelloFrame{
		CompressionGZip:      compression == CompressionMethodGzip,
		Ack:                  false,
		MaxConcurrentStreams: 0,
	}).IntoFrame())
	if err != nil {
		return err
	}
	c.waitForHello()
	return nil
}

func (c *client) Close() error {
	_ = c.io.Close()
	c.signalHelloOK()
	return nil
}

func (c *client) Write(frame *Frame) error {
	if c.err != nil {
		return c.err
	}
	out := outboundFramePool.Get().(*outboundFrame)
	defer outboundFramePool.Put(out)

	out.frame = frame
	out.result = make(chan error, 1)
	c.toWrite <- out
	return <-out.result
}

func (c *client) NewStream() (Stream, error) {
	if c.err != nil {
		return nil, c.err
	}

	c.streamsMu.Lock()
	id := c.lastStreamID + 1
	c.lastStreamID++
	c.streamsMu.Unlock()

	if err := c.Write((&MakeStreamFrame{StreamID: id}).IntoFrame()); err != nil {
		return nil, err
	}

	str := NewStream(id, c)

	c.streamsMu.Lock()
	c.streams[id] = str
	c.streamsMu.Unlock()

	return str, nil
}

func (c *client) Terminate(reason ErrorCode) error {
	if c.err != nil {
		return c.err
	}

	c.streamsMu.Lock()
	id := c.lastStreamID
	c.streamsMu.Unlock()
	return c.Write((&GoAwayFrame{
		LastStreamID:   id,
		ErrorCode:      reason,
		AdditionalData: nil,
	}).IntoFrame())
}

func (c *client) terminate() {
	c.runningMu.Lock()
	defer c.runningMu.Unlock()
	if !c.running {
		return
	}

	c.running = false
	close(c.toWrite)
	close(c.drop)
}

func (c *client) service() error {
	c.runningMu.Lock()
	c.running = true
	c.runningMu.Unlock()
	err := make(chan error, 2)
	go c.serviceWrites()
	go c.serviceReads(err)
	select {
	case err := <-err:
		c.terminate()
		if !c.running {
			return nil
		}
		return err
	case <-c.drop:
		c.terminate()
		return nil
	}
}

func (c *client) serviceWrites() {
loop:
	for out := range c.toWrite {
		c.writeMu.Lock()
		buf := bytes.NewReader(out.frame.Bytes(c.compression))
		_, err := io.Copy(c.io, buf)
		if err != nil {
			if c.err != nil {
				out.result <- c.err
				c.writeMu.Unlock()
				continue loop
			}
			c.runningMu.Lock()
			if !c.running {
				c.runningMu.Unlock()
				c.writeMu.Unlock()
				return
			}
			c.runningMu.Unlock()

			out.result <- err
			c.writeMu.Unlock()
			continue loop
		}
		c.writeMu.Unlock()
		out.result <- nil
	}
}

func (c *client) serviceReads(errCh chan error) {
	for {
		fr, err := c.reader.Read()
		if err != nil {
			c.runningMu.Lock()
			if !c.running {
				c.runningMu.Unlock()
				return
			}
			c.runningMu.Unlock()
			errCh <- err
			return
		}
		if err = c.dispatch(fr); err != nil {
			c.runningMu.Lock()
			if !c.running {
				c.runningMu.Unlock()
				return
			}
			c.runningMu.Unlock()
			errCh <- err
			return
		}
	}
}

func (c *client) dispatch(fr *Frame) error {
	var err error
	if fr.FrameKind != FrameKindHello && fr.FrameKind != FrameKindPing && fr.FrameKind != FrameKindResetStream && fr.FrameKind != FrameKindGoAway && !c.setup {
		c.reset(ErrorCodeProtocolError, "Expected a HELLO frame, received "+fr.FrameKind.String()+" instead")
		return nil
	}

	switch fr.FrameKind {
	case FrameKindData:
		data := &DataFrame{}
		if err = data.FromFrame(fr); err != nil {
			break
		}
		err = c.handleData(data)
	case FrameKindResetStream:
		rst := &ResetStreamFrame{}
		if err = rst.FromFrame(fr); err != nil {
			break
		}
		err = c.handleResetStream(rst)
	case FrameKindPing:
		ping := &PingFrame{}
		if err = ping.FromFrame(fr); err != nil {
			break
		}
		err = c.handlePing(ping)
	case FrameKindGoAway:
		c.runningMu.Lock()
		goAway := &GoAwayFrame{}
		if err = goAway.FromFrame(fr); err != nil {
			c.runningMu.Unlock()
			break
		}
		c.runningMu.Unlock()
		if err = c.handleGoAway(goAway); err != nil {
			c.runningMu.Unlock()
			break
		}
		return nil
	case FrameKindHello:
		hello := &HelloFrame{}
		if err = hello.FromFrame(fr); err != nil {
			break
		}
		err = c.handleHello(hello)

	default:
		c.terminate()
		return nil
	}

	if err != nil {
		return err
	}

	return nil
}

func (c *client) reset(reason ErrorCode, details string) {
	c.err = &ConnectionResetError{Reason: reason, Details: details}
	if c.err != nil {
		return
	}

	fr := &GoAwayFrame{
		LastStreamID:   c.lastStreamID,
		ErrorCode:      reason,
		AdditionalData: nil,
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	r := bytes.NewReader(fr.IntoFrame().Bytes(c.compression))
	_, err := io.Copy(c.io, r)
	if err != nil {
		// TODO: Log
	}
}

func (c *client) handlePing(fr *PingFrame) error {
	if fr.Ack {
		return nil
	}

	pong := &PingFrame{
		Ack:     true,
		Payload: fr.Payload,
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	r := bytes.NewReader(pong.IntoFrame().Bytes(c.compression))
	_, err := io.Copy(c.io, r)
	return err
}

func (c *client) handleGoAway(fr *GoAwayFrame) error {
	if !c.running {
		return nil
	}
	c.running = false

	c.err = &ConnectionResetError{
		Reason:  fr.ErrorCode,
		Details: "Server closed connection with status " + fr.ErrorCode.String(),
	}
	c.terminate()
	return nil
}

func (c *client) handleHello(fr *HelloFrame) error {
	if !fr.Ack {
		c.reset(ErrorCodeProtocolError, "Server emitted a non-ack HELLO frame")
	}

	if fr.CompressionGZip {
		c.compression = CompressionMethodGzip
	}
	c.maxConcurrentStreams = fr.MaxConcurrentStreams
	c.setup = true

	c.signalHelloOK()

	return nil
}

func (c *client) resetStream(id uint32, reason ErrorCode) error {
	return c.Write((&ResetStreamFrame{
		StreamID:  id,
		ErrorCode: reason,
	}).IntoFrame())
}

func (c *client) fetchStream(id uint32) (Stream, bool) {
	c.streamsMu.Lock()
	defer c.streamsMu.Unlock()

	s, ok := c.streams[id]
	return s, ok
}

func (c *client) handleData(fr *DataFrame) error {
	str, ok := c.fetchStream(fr.StreamID)
	if !ok {
		return c.resetStream(fr.StreamID, ErrorCodeProtocolError)
	}
	str.handleData(fr)
	return nil
}

func (c *client) handleResetStream(fr *ResetStreamFrame) error {
	str, ok := c.fetchStream(fr.StreamID)
	if !ok {
		return c.resetStream(fr.StreamID, ErrorCodeProtocolError)
	}
	str.handleResetStream(fr)
	return nil
}

func (c *client) waitForHello() {
	<-c.helloOK
}

func (c *client) handleStream(*Stream) { /* noop */ }

func (c *client) cancelStream(*Stream) { /* noop */ }
