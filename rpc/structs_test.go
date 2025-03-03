package rpc

import (
	"bytes"
	"github.com/arf-rpc/arf-go/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRequest(t *testing.T) {
	req := &Request{
		Service:   "org.example.test/FooService",
		Method:    "Add",
		Streaming: false,
		Metadata:  MetadataFromStringPairs("foo", "bar"),
		Params:    []any{uint32(1), uint32(2)},
	}
	encoded, err := req.Wrap()
	require.NoError(t, err)
	r := bytes.NewReader(encoded)
	read, err := MessageTFromReader[*Request](r)
	require.NoError(t, err)
	for i, v := range read.Params {
		read.Params[i] = uint32(v.(uint64))
	}
	assert.Equal(t, req, read)
}

func TestResponse(t *testing.T) {
	res := &Response{
		Status:    uint16(status.InternalError),
		Streaming: true,
		Metadata:  MetadataFromStringPairs("foo", "bar"),
		Params:    []any{"hello", "world"},
	}
	encoded, err := res.Wrap()
	require.NoError(t, err)
	r := bytes.NewReader(encoded)
	read, err := MessageTFromReader[*Response](r)
	require.NoError(t, err)
	assert.Equal(t, res, read)
}

func TestStartStream(t *testing.T) {
	s := &StartStream{}
	encoded, err := s.Wrap()
	require.NoError(t, err)
	r := bytes.NewReader(encoded)
	read, err := MessageTFromReader[*StartStream](r)
	require.NoError(t, err)
	require.Equal(t, s, read)
}

func TestStreamItem(t *testing.T) {
	s := &StreamItem{
		Value: "a value",
	}
	encoded, err := s.Wrap()
	require.NoError(t, err)
	r := bytes.NewReader(encoded)
	read, err := MessageTFromReader[*StreamItem](r)
	require.NoError(t, err)
	require.Equal(t, s, read)
}

func TestEndStream(t *testing.T) {
	s := &EndStream{}
	encoded, err := s.Wrap()
	require.NoError(t, err)
	r := bytes.NewReader(encoded)
	read, err := MessageTFromReader[*EndStream](r)
	require.NoError(t, err)
	require.Equal(t, s, read)
}

func TestStreamError(t *testing.T) {
	s := &StreamError{
		Status:   uint16(status.Aborted),
		Metadata: MetadataFromStringPairs("foo", "bar"),
	}
	encoded, err := s.Wrap()
	require.NoError(t, err)
	r := bytes.NewReader(encoded)
	read, err := MessageTFromReader[*StreamError](r)
	require.NoError(t, err)
	require.Equal(t, s, read)
}
