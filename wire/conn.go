package wire

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
)

type outboundFrame struct {
	frame  *Frame
	result chan error
}

var outboundFramePool = sync.Pool{
	New: func() interface{} { return &outboundFrame{} },
}

type conn interface {
	Write(fr *Frame) error
}

// Conn represents a single active connection to a server
type Conn struct {
	io          io.ReadWriteCloser
	id          int
	compression CompressionMethod
	err         error
	reader      *FrameReader

	streamsMu    sync.RWMutex
	streams      map[uint32]Stream
	lastStreamID uint32

	toWrite chan *outboundFrame
	drop    chan struct{}

	maxConcurrentStreams uint32
	runningMu            sync.Mutex
	running              bool
	configured           bool
	terminateAfter       *Frame
	parent               server
}

func NewConn(s server, id int, io io.ReadWriteCloser) *Conn {
	c := &Conn{
		io:           io,
		id:           id,
		compression:  CompressionMethodNone,
		streams:      make(map[uint32]Stream),
		lastStreamID: 0,
		toWrite:      make(chan *outboundFrame, 128),
		drop:         make(chan struct{}),
		running:      true,
		reader:       NewFrameReader(io),
		parent:       s,
	}

	go c.serviceWrites()
	go c.serviceReads()

	return c
}

func (c *Conn) terminate() {
	if !c.running {
		return
	}
	c.running = false

	close(c.drop)
	_ = c.io.Close()
	close(c.toWrite)
	if c.parent != nil {
		c.parent.connectionClosed(c.id)
	}
}

func (c *Conn) serviceWrites() {
	for out := range c.toWrite {
		if c.err != nil {
			out.result <- c.err
			continue
		}

		data := out.frame.Bytes(c.compression)
		_, err := io.Copy(c.io, bytes.NewReader(data))
		if err != nil {
			out.result <- err
			continue
		}
		out.result <- nil

		if out.frame == c.terminateAfter {
			c.terminate()
		}
	}
}

func (c *Conn) serviceReads() {
	for c.running {
		fr, err := c.reader.Read()
		if err != nil {
			c.err = err
			c.terminate()
			break
		}
		c.dispatchFrame(fr)
	}
}

func (c *Conn) dispatchFrame(fr *Frame) {
	switch fr.FrameKind {
	case FrameKindHello:
		hello := &HelloFrame{}
		if err := hello.FromFrame(fr); err != nil {
			c.goAway(ErrorCodeProtocolError, nil, true)
			return
		}
		c.handleHello(hello)

	case FrameKindPing:
		ping := &PingFrame{}
		if err := ping.FromFrame(fr); err != nil {
			c.goAway(ErrorCodeProtocolError, nil, true)
			return
		}
		c.handlePing(ping)

	case FrameKindGoAway:
		goAway := &GoAwayFrame{}
		if err := goAway.FromFrame(fr); err != nil {
			c.goAway(ErrorCodeProtocolError, nil, true)
			return
		}
		c.handleGoAway(goAway)

	case FrameKindMakeStream:
		makeStream := &MakeStreamFrame{}
		if err := makeStream.FromFrame(fr); err != nil {
			c.goAway(ErrorCodeProtocolError, nil, true)
			return
		}
		c.handleMakeStream(makeStream)

	case FrameKindResetStream:
		rs := &ResetStreamFrame{}
		if err := rs.FromFrame(fr); err != nil {
			c.goAway(ErrorCodeProtocolError, nil, true)
			return
		}
		c.handleResetFrame(rs)

	case FrameKindData:
		data := &DataFrame{}
		if err := data.FromFrame(fr); err != nil {
			c.goAway(ErrorCodeProtocolError, nil, true)
			return
		}
		c.handleData(data)

	default:
		c.goAway(ErrorCodeProtocolError, nil, true)
		return
	}
}

func (c *Conn) handleHello(conf *HelloFrame) {
	if c.configured {
		c.goAway(ErrorCodeProtocolError, nil, true)
		return
	}

	gzip := false
	if conf.CompressionGZip {
		gzip = true
		c.compression = CompressionMethodGzip
	}

	c.configured = true
	err := c.Write((&HelloFrame{
		CompressionGZip:      gzip,
		Ack:                  true,
		MaxConcurrentStreams: 0, // TODO
	}).IntoFrame())
	if err != nil {
		// TODO: Log
		c.terminate()
	}
}

func (c *Conn) handlePing(ping *PingFrame) {
	if !c.configured {
		c.goAway(ErrorCodeProtocolError, nil, true)
		return
	}

	if ping.Ack {
		return // TODO
	}

	err := c.Write((&PingFrame{
		Ack:     true,
		Payload: ping.Payload,
	}).IntoFrame())
	if err != nil {
		c.err = err
		// TODO: log
		c.terminate()
	}
}

func (c *Conn) handleGoAway(msg *GoAwayFrame) {
	if !c.configured {
		c.goAway(ErrorCodeProtocolError, nil, true)
		return
	}

	// Client intends to disconnect.
	c.cancelStreams(msg.ErrorCode)
	c.terminate()
	return
}

func (c *Conn) cancelStreams(errCode ErrorCode) {
	c.streamsMu.Lock()
	defer c.streamsMu.Unlock()
	for _, s := range c.streams {
		if err := s.Reset(errCode); err != nil {
			if errors.Is(err, ClosedStreamErr) {
				continue
			} else {
				// TODO: Log
			}
		}
	}
}

func (c *Conn) fetchStream(id uint32) (Stream, bool) {
	c.streamsMu.RLock()
	defer c.streamsMu.RUnlock()
	v, ok := c.streams[id]
	return v, ok
}

func (c *Conn) resetStream(id uint32, code ErrorCode) {
	r := &ResetStreamFrame{
		StreamID:  id,
		ErrorCode: code,
	}
	if err := c.Write(r.IntoFrame()); err != nil {
		// TODO: Log
	}
}

func (c *Conn) handleMakeStream(req *MakeStreamFrame) {
	if !c.configured {
		c.goAway(ErrorCodeProtocolError, nil, true)
		return
	}

	id := req.StreamID
	_, ok := c.fetchStream(id)
	if ok {
		// TODO: Log
		c.goAway(ErrorCodeProtocolError, nil, true)
		return
	}

	c.streamsMu.Lock()
	defer c.streamsMu.Unlock()
	c.streams[id] = NewStream(id, c)
	if c.parent != nil {
		c.parent.ServiceStream(c.streams[id])
	}
}

func (c *Conn) handleResetFrame(rs *ResetStreamFrame) {
	if !c.configured {
		c.goAway(ErrorCodeProtocolError, nil, true)
		return
	}

	s, ok := c.fetchStream(rs.StreamID)
	if !ok {
		// TODO: Log
		c.resetStream(rs.StreamID, ErrorCodeProtocolError)
		return
	}
	s.handleResetStream(rs)
}

func (c *Conn) handleData(data *DataFrame) {
	if !c.configured {
		c.goAway(ErrorCodeProtocolError, nil, true)
		return
	}

	s, ok := c.fetchStream(data.StreamID)
	if !ok {
		// TODO: Log
		c.resetStream(data.StreamID, ErrorCodeProtocolError)
		return
	}

	s.handleData(data)
}

func (c *Conn) Write(fr *Frame) error {
	out := outboundFramePool.Get().(*outboundFrame)
	defer outboundFramePool.Put(out)
	err := make(chan error, 1)
	out.frame = fr
	out.result = err
	c.toWrite <- out
	return <-err
}

func (c *Conn) goAway(code ErrorCode, extraData []byte, terminate bool) {
	fr := &GoAwayFrame{
		LastStreamID:   c.lastStreamID,
		ErrorCode:      code,
		AdditionalData: extraData,
	}

	frame := fr.IntoFrame()
	if terminate {
		c.terminateAfter = frame
	}

	if err := c.Write(frame); err != nil {
		fmt.Printf("conn.goAway: Write error: %s\n", err)
		// TODO: Log
	}
}
