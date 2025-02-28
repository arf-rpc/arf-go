package wire

import (
	"bytes"
	"compress/flate"
	"fmt"
	"io"
)

//go:generate stringer -type=CompressionMethod -output=compression_string.go

type CompressionMethod int

const (
	CompressionMethodNone CompressionMethod = iota
	CompressionMethodGzip
)

var allCompressionMethods = []CompressionMethod{
	CompressionMethodNone,
	CompressionMethodGzip,
}

func (c CompressionMethod) Compress(buf []byte) []byte {
	var b bytes.Buffer
	var w io.WriteCloser

	switch c {
	case CompressionMethodNone:
		return buf
	case CompressionMethodGzip:
		w, _ = flate.NewWriter(&b, flate.BestCompression)
	default:
		panic(fmt.Sprintf("Unsupported compression method %s", c))
	}

	_, _ = w.Write(buf)
	_ = w.Close()
	return b.Bytes()
}

func (c CompressionMethod) Decompress(data []byte) ([]byte, error) {
	var r io.Reader

	switch c {
	case CompressionMethodNone:
		return data, nil
	case CompressionMethodGzip:
		r = flate.NewReader(bytes.NewReader(data))
		defer func() {
			_ = r.(io.Closer).Close()
		}()
	default:
		panic(fmt.Sprintf("Unsupported compression method %s", c))
	}

	return io.ReadAll(r)
}
