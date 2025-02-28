package arf

import (
	"context"
	"github.com/arf-rpc/arf/rpc"
	"github.com/arf-rpc/arf/status"
	"github.com/arf-rpc/arf/wire"
)

type Context interface {
	Recv() (any, error)
	Send(v any) error
	EndSend() error
	Response() *rpc.Response
	Request() *rpc.Request
	SendResponse(code status.Status, params []any, streaming bool, metadata rpc.Metadata) error
}

type ctx struct {
	str               wire.Stream
	err               error
	hasRecvStream     bool
	recvStreamError   error
	recvStreamStarted bool
	hasSendStream     bool
	sendStreamStarted bool
	sendStreamError   error
	resp              *rpc.Response
	req               *rpc.Request
	context           context.Context
	hasSentResponse   bool
}

func (c *ctx) Response() *rpc.Response { return c.resp }

func (c *ctx) Recv() (any, error) {
	if !c.hasRecvStream {
		return nil, &rpc.NoStreamError{Recv: true}
	}
	if c.err != nil {
		return nil, c.err
	}
	if c.recvStreamError != nil {
		return nil, c.recvStreamError
	}

	if !c.recvStreamStarted {
		msg, err := rpc.MessageFromReader(c.str)
		if err != nil {
			c.err = err
			return nil, err
		}
		if msg.Kind() == rpc.MessageKindStartStream {
			c.recvStreamStarted = true
		} else {
			c.err = &rpc.StreamFailure{
				Msg: "received unexpected message kind",
			}
			return nil, c.err
		}
	}

	for {
		msg, err := rpc.MessageFromReader(c.str)
		if err != nil {
			c.err = err
			return nil, err
		}

		switch msg.Kind() {
		case rpc.MessageKindStreamItem:
			return msg.(*rpc.StreamItem).Value, nil
		case rpc.MessageKindEndStream:
			c.recvStreamError = &rpc.StreamEndError{}
			return nil, c.recvStreamError
		case rpc.MessageKindStreamError:
			c.recvStreamError = msg.(*rpc.StreamError)
			return nil, c.recvStreamError
		case rpc.MessageKindStreamMetadata:
			meta := msg.(*rpc.StreamMetadata)
			c.resp.Metadata = meta.Metadata
		default:
			c.err = &rpc.StreamFailure{Msg: "received unexpected message kind"}
			return nil, c.err
		}
	}
}

func (c *ctx) Send(v any) error {
	if !c.hasSendStream {
		return &rpc.NoStreamError{Recv: false}
	}
	if c.err != nil {
		return c.err
	}
	if c.sendStreamError != nil {
		return c.sendStreamError
	}

	if !c.sendStreamStarted {
		enc, err := (&rpc.StartStream{}).Wrap()
		if err != nil {
			c.sendStreamError = err
			return err
		}

		err = c.str.Write(enc, false)
		if err != nil {
			c.err = err
			return err
		}

		c.sendStreamStarted = true
	}

	enc, err := (&rpc.StreamItem{Value: v}).Wrap()
	if err != nil {
		c.sendStreamError = err
		return err
	}

	err = c.str.Write(enc, false)
	if err != nil {
		c.err = err
	}
	return err
}

func (c *ctx) EndSend() error {
	if !c.hasSendStream {
		return &rpc.NoStreamError{Recv: false}
	}
	if c.err != nil {
		return c.err
	}
	if c.sendStreamError != nil {
		return c.sendStreamError
	}

	msg := &rpc.EndStream{}
	data, err := msg.Wrap()
	if err != nil {
		c.err = err
		return err
	}
	err = c.str.Write(data, true)
	if err != nil {
		c.err = err
	}
	return err
}

func (c *ctx) ReadResponse() (*rpc.Response, error) {
	if c.err != nil {
		return nil, c.err
	}

	resp, err := rpc.MessageTFromReader[*rpc.Response](c.str)
	if err != nil {
		c.err = err
		return nil, err
	}

	c.resp = resp
	c.hasRecvStream = resp.Streaming

	return resp, nil
}

func (c *ctx) SendResponse(code status.Status, params []any, streaming bool, metadata rpc.Metadata) error {
	resp := &rpc.Response{
		Status:    uint16(code),
		Streaming: streaming,
		Metadata:  metadata,
		Params:    params,
	}

	enc, err := resp.Wrap()
	if err != nil {
		return err
	}

	c.hasSentResponse = true
	c.hasSendStream = streaming
	c.sendStreamStarted = false

	return c.str.Write(enc, !streaming)
}

func (c *ctx) Request() *rpc.Request { return c.req }
