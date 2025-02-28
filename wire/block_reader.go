package wire

import (
	"io"
	"sync"
)

type BlockReader struct {
	blocks chan []byte
	buf    []byte

	closedMu sync.Mutex
	closed   bool
}

func NewBlockReader() *BlockReader {
	return &BlockReader{
		blocks: make(chan []byte, 128),
	}
}

func (r *BlockReader) internalClose() {
	r.closedMu.Lock()
	defer r.closedMu.Unlock()
	if r.closed {
		return
	}
	r.closed = true
	close(r.blocks)
}

func (r *BlockReader) Enqueue(data []byte) {
	if len(data) == 0 {
		return
	}

	r.closedMu.Lock()
	defer r.closedMu.Unlock()
	if r.closed {
		return
	}
	r.blocks <- data
}

func (r *BlockReader) Close() error {
	r.internalClose()
	return nil
}

func (r *BlockReader) clearBuffer() {
	if len(r.buf) == 0 {
		r.buf = nil
	}
}

func (r *BlockReader) TryRead(into []byte) (bool, int, error) {
	defer r.clearBuffer()
	if r.buf == nil {
		r.closedMu.Lock()
		if r.closed {
			r.closedMu.Unlock()
			return true, 0, io.EOF
		}
		r.closedMu.Unlock()
		select {
		case r.buf = <-r.blocks:
			if r.buf == nil {
				return true, 0, io.EOF
			}
		default:
			return false, 0, nil
		}
	}
	intoLen := len(into)

	if len(r.buf) >= intoLen {
		copy(into, r.buf[:intoLen])
		r.buf = r.buf[intoLen:]
		return true, intoLen, nil
	}

	// buf length is smaller than `into`. Copy what we have, and return a
	// partial read.
	l := len(r.buf)
	copy(into, r.buf[:l])
	r.buf = r.buf[l:]
	return true, l, nil
}

func (r *BlockReader) Read(into []byte) (int, error) {
	defer r.clearBuffer()
	if r.buf == nil {
		r.closedMu.Lock()
		if r.closed {
			r.closedMu.Unlock()
			return 0, io.EOF
		}
		r.closedMu.Unlock()
		r.buf = <-r.blocks
		if r.buf == nil {
			return 0, io.EOF
		}
	}
	intoLen := len(into)

	if len(r.buf) >= intoLen {
		copy(into, r.buf[:intoLen])
		r.buf = r.buf[intoLen:]
		return intoLen, nil
	}

	// buf length is smaller than `into`. Copy what we have, and return a
	// partial read.
	l := len(r.buf)
	copy(into, r.buf[:l])
	r.buf = r.buf[l:]
	return l, nil
}
