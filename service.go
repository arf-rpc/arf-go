package arf

import (
	"context"
)

type Service interface {
	ServiceResponder
	ArfServiceID() string
	RespondsTo(name string) bool
}

type ServiceResponder interface {
	InvokeMethod(name string, ctx context.Context, request Context) error
}

type ServiceExecutor func(context.Context, Context) error

type ServiceAdapter struct {
	Methods   map[string]ServiceExecutor
	ServiceID string
}

func (s ServiceAdapter) ArfServiceID() string { return s.ServiceID }

func (s ServiceAdapter) InvokeMethod(name string, ctx context.Context, request Context) error {
	return s.Methods[name](ctx, request)
}

func (s ServiceAdapter) RespondsTo(name string) bool {
	_, ok := s.Methods[name]
	return ok
}
