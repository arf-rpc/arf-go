package arf

import (
	"context"
	"crypto/tls"
	"github.com/arf-rpc/arf-go/rpc"
	"github.com/arf-rpc/arf-go/wire"
	"net"
)

type ClientOption func(*client)

func WithTLSConfig(cfg *tls.Config) ClientOption {
	return func(c *client) {
		c.tlsConfig = cfg
	}
}

func Dial(addr string, opts ...ClientOption) (Client, error) {
	c := &client{}
	for _, fn := range opts {
		fn(c)
	}

	var conn net.Conn
	var err error
	if c.tlsConfig != nil {
		conn, err = tls.Dial("tcp", addr, c.tlsConfig)
	} else {
		conn, err = net.Dial("tcp", addr)
	}

	if err != nil {
		return nil, err
	}

	c.c = wire.NewClient(conn)

	if err = c.c.Configure(wire.CompressionMethodNone); err != nil {
		return nil, err
	}

	return c, nil
}

type Client interface {
	Close() error
	Call(ctx context.Context, serviceIdentifier, serviceMethod string, opts ...CallOption) (Context, error)
}

type client struct {
	c         wire.Client
	tlsConfig *tls.Config
}

type callOptions struct {
	outputMetadataTarget *rpc.Metadata
}

func (c *client) Close() error {
	return c.c.Close()
}

type CallOption func(*rpc.Request, *callOptions)

func WithStream() CallOption {
	return func(r *rpc.Request, o *callOptions) {
		r.Streaming = true
	}
}

func WithMetadata(m rpc.Metadata) CallOption {
	return func(r *rpc.Request, o *callOptions) {
		r.Metadata = m
	}
}

func WithParams(params ...any) CallOption {
	return func(r *rpc.Request, o *callOptions) {
		r.Params = params
	}
}

func WithOutputMetadata(m *rpc.Metadata) CallOption {
	return func(r *rpc.Request, o *callOptions) {
		if m == nil {
			return
		}
		o.outputMetadataTarget = m
	}
}

func (c *client) cancelErr(str wire.Stream, err error) error {
	_ = str.Reset(wire.ErrorCodeCancel)
	return err
}

func (c *client) Call(cctx context.Context, serviceIdentifier, serviceMethod string, opts ...CallOption) (Context, error) {
	str, err := c.c.NewStream()
	if err != nil {
		return nil, err
	}
	req := &rpc.Request{
		Service: serviceIdentifier,
		Method:  serviceMethod,
	}
	extraOpts := callOptions{}
	for _, v := range opts {
		v(req, &extraOpts)
	}

	encoded, err := req.Wrap()
	if err != nil {
		return nil, c.cancelErr(str, err)
	}

	if err = str.Write(encoded, !req.Streaming); err != nil {
		_ = str.CloseLocal()
		return nil, err
	}

	if cctx.Err() != nil {
		return nil, c.cancelErr(str, err)
	}

	if req.Streaming {
		startStream := &rpc.StartStream{}
		data, err := startStream.Wrap()
		if err != nil {
			return nil, c.cancelErr(str, err)
		}
		err = str.Write(data, false)
		if err != nil {
			return nil, c.cancelErr(str, err)
		}
	}

	resp, err := rpc.MessageTFromReader[*rpc.Response](str)
	if err != nil {
		return nil, c.cancelErr(str, err)
	}

	if extraOpts.outputMetadataTarget != nil {
		*extraOpts.outputMetadataTarget = resp.Metadata
	}

	return &ctx{
		context:           cctx,
		str:               str,
		hasRecvStream:     resp.Streaming,
		recvStreamError:   nil,
		recvStreamStarted: false,
		hasSendStream:     req.Streaming,
		sendStreamError:   nil,
		resp:              resp,
		req:               req,
	}, nil
}
