package arf

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/arf-rpc/arf/rpc"
	"github.com/arf-rpc/arf/status"
	"github.com/arf-rpc/arf/wire"
	"github.com/go-stdlog/stdlog"
	"net"
	"os"
)

var StreamCanceledErr = errors.New("stream canceled")

type handler func(ctx context.Context, req Context) error
type Interceptor func(ctx context.Context, req Context, next Interceptor) error
type IDGenerator func() (string, error)

type ServerOptions struct {
	MaxConcurrentStreams uint32 /* TODO */
	Logger               stdlog.Logger
	IDGenerator          IDGenerator
}

type Server interface {
	RegisterService(service Service) error
	MustRegisterService(service Service)
	Serve() error
	Shutdown() error
	RegisterInterceptor(interceptor ...Interceptor)
}

func pseudoUUIDGen() (func() (string, error), error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return func() (string, error) {
		b := make([]byte, 16)
		_, err := rand.Read(b)
		if err != nil {
			return "", err
		}
		return hostname + "-" + hex.EncodeToString(b), nil
	}, nil
}

func NewServerListen(addr string, opts ServerOptions) (Server, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return NewServer(l, opts)
}

func NewServer(l net.Listener, opts ServerOptions) (Server, error) {
	var err error
	if opts.IDGenerator == nil {
		opts.IDGenerator, err = pseudoUUIDGen()
		if err != nil {
			return nil, err
		}
	}

	server := &srv{
		listener:      l,
		services:      make(map[string]Service),
		streams:       make(map[string]wire.Stream),
		streamContext: make(map[string]*streamContext),
		interceptors:  nil,
		idGenerator:   opts.IDGenerator,
		logger:        stdlog.Discard,
	}

	server.logger = opts.Logger

	srv := wire.NewServer(l, server)
	server.wireServer = srv

	return server, nil
}

type streamContext struct {
	cancel context.CancelCauseFunc
	ctx    context.Context
}

type srv struct {
	listener      net.Listener
	wireServer    *wire.Server
	services      map[string]Service
	streams       map[string]wire.Stream
	streamContext map[string]*streamContext
	interceptors  []Interceptor
	idGenerator   func() (string, error)
	logger        stdlog.Logger
}

func (s *srv) RegisterService(service Service) error {
	id := service.ArfServiceID()
	_, ok := s.services[id]
	if ok {
		return fmt.Errorf("service %s already registered", id)
	}
	s.services[id] = service

	return nil
}

func (s *srv) MustRegisterService(service Service) {
	err := s.RegisterService(service)
	if err != nil {
		panic(err)
	}
}

func (s *srv) Serve() error {
	return s.wireServer.Serve()
}

func (s *srv) RegisterInterceptor(interceptor ...Interceptor) {
	s.interceptors = append(s.interceptors, interceptor...)
}

func chainInterceptors(finalHandler handler, interceptors ...Interceptor) handler {
	finalInterceptor := func(ctx context.Context, req Context, _ Interceptor) error {
		return finalHandler(ctx, req)
	}

	for i := len(interceptors) - 1; i >= 0; i-- {
		currentInterceptor := interceptors[i]
		previousNext := finalInterceptor

		finalInterceptor = func(ctx context.Context, req Context, _ Interceptor) error {
			return currentInterceptor(ctx, req, previousNext)
		}
	}

	return func(ctx context.Context, req Context) error {
		return finalInterceptor(ctx, req, nil)
	}
}

func (s *srv) ServiceStream(str wire.Stream) {
	reqID, err := s.idGenerator()
	if err != nil {
		s.logger.Error(err, "failed to generate request ID")
		_ = str.Reset(wire.ErrorCodeInternalError)
		return
	}
	str.SetExternalID(reqID)
	log := s.logger.WithFields("request_id", reqID)

	req, err := rpc.MessageTFromReader[*rpc.Request](str)
	if err != nil {
		log.Error(err, "Failed deserializing request payload", "error", err)
		s.rejectInvalidStreamMsg(str, status.InternalError, "Failed deserializing request payload")
		return
	}

	svc, ok := s.services[req.Service]
	if !ok || !svc.RespondsTo(req.Method) {
		log.Info("Rejecting request as service does not respond to the requested method", "service", req.Streaming, "method", req.Method)
		s.rejectInvalidStream(str, status.Unimplemented)
		return
	}

	cctx, cancel := context.WithCancelCause(context.Background())
	s.streams[reqID] = str
	s.streamContext[reqID] = &streamContext{
		cancel: cancel,
		ctx:    cctx,
	}

	reqCtx := &ctx{
		str:           str,
		hasRecvStream: req.Streaming,
		req:           req,
		context:       cctx,
	}

	chain := chainInterceptors(func(ctx context.Context, req Context) error {
		return s.guardInvoke(ctx, reqCtx, svc)
	}, s.interceptors...)

	if err = chain(cctx, reqCtx); err != nil {
		var badStatusErr *status.BadStatus
		var statusErr *status.Status
		switch {
		case errors.As(err, &badStatusErr):
			s.emitError(str, badStatusErr, reqCtx)
		case errors.As(err, &statusErr):
			s.emitError(str, &status.BadStatus{
				Code:    *statusErr,
				Message: statusErr.Error(),
			}, reqCtx)
		default:
			log.Error(err, "Request handler or interceptor chain returned an error")
			s.emitError(str, &status.BadStatus{
				Code:    status.InternalError,
				Message: err.Error(),
			}, reqCtx)
		}

		return
	}
}

func (s *srv) guardInvoke(ctx context.Context, req *ctx, svc Service) (err error) {
	defer func() {
		if err == nil && !req.hasSentResponse {
			err = req.SendResponse(status.OK, nil, false, nil)
		}
	}()
	//defer func() {
	//	if rawErr := recover(); rawErr != nil {
	//		if rErr, ok := rawErr.(error); ok {
	//			err = fmt.Errorf("recovered from panic: %v", rErr)
	//		} else {
	//			err = fmt.Errorf("recovered from panic: %v", err)
	//		}
	//	}
	//}()

	return svc.InvokeMethod(req.Request().Method, ctx, req)
}

func (s *srv) CancelStream(stream wire.Stream) {
	ctx := s.streamContext[stream.ExternalID()]
	if ctx == nil {
		return
	}
	ctx.cancel(StreamCanceledErr)
}

func (s *srv) rejectInvalidStreamMsg(str wire.Stream, code status.Status, msg string) {
	enc, err := (&rpc.Response{
		Status:    uint16(code),
		Streaming: false,
		Metadata: rpc.MetadataFromMap(map[string][]byte{
			"arf-status-description": []byte(msg),
		}),
		Params: nil,
	}).Wrap()
	if err != nil {
		_ = str.Reset(wire.ErrorCodeInternalError)
		// TODO: Log
	}

	err = str.Write(enc, true)
	if err != nil {
		_ = str.Reset(wire.ErrorCodeInternalError)
		// TODO: Log
	}
}

func (s *srv) rejectInvalidStream(str wire.Stream, code status.Status) {
	s.rejectInvalidStreamMsg(str, code, code.Error())
}

func (s *srv) emitError(str wire.Stream, status *status.BadStatus, resp Context) {
	ctx := resp.(*ctx)
	var (
		enc []byte
		err error
	)
	meta := rpc.MetadataFromMap(map[string][]byte{
		"arf-status-description": []byte(status.Message),
	})
	if ctx.sendStreamStarted {
		enc, err = (&rpc.StreamError{
			Status:   uint16(status.Code),
			Metadata: meta,
		}).Wrap()
	} else {
		enc, err = (&rpc.Response{
			Status:    uint16(status.Code),
			Streaming: false,
			Metadata:  meta,
			Params:    nil,
		}).Wrap()
	}

	if err != nil {
		// TODO: Log
		_ = str.Reset(wire.ErrorCodeInternalError)
		return
	}
	err = str.Write(enc, true)
	if err != nil {
		// TODO: Log
		_ = str.Reset(wire.ErrorCodeInternalError)
		return
	}
}

func (s *srv) Shutdown() error {
	return s.wireServer.Shutdown()
}
