package wire

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

func TestBlockReader(t *testing.T) {
	r := NewBlockReader()
	a := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	b := []byte{0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	c := []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18}
	r.Enqueue(a)
	r.Enqueue(b)
	r.Enqueue(c)

	firstRead := make([]byte, 16)
	n, err := io.ReadFull(r, firstRead)
	assert.Equal(t, 16, n)
	assert.Nil(t, err)
	assert.Equal(t, bytes.Join([][]byte{a, b}, nil), firstRead)

	secondRead := make([]byte, 16)
	n, err = r.Read(secondRead)
	assert.Equal(t, 8, n)
	assert.Nil(t, err)
	assert.Equal(t, c, secondRead[:n])
}
