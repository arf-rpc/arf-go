package wire

import (
	"fmt"
	"io"
	"sync"
)

type Stream interface {
	io.Reader

	handleResetStream(rs *ResetStreamFrame)
	handleData(data *DataFrame)

	Write(data []byte, endStream bool) error
	Reset(code ErrorCode) error
	CloseLocal() error
	ID() uint32
	SetExternalID(string)
	ExternalID() string
}

func NewStream(id uint32, c conn) Stream {
	return &stream{
		c:      c,
		id:     id,
		reader: NewBlockReader(),
	}
}

type stream struct {
	id         uint32
	c          conn
	state      streamState
	reader     *BlockReader
	writeMu    sync.Mutex
	externalID string
}

func (s *stream) ID() uint32                      { return s.id }
func (s *stream) SetExternalID(externalID string) { s.externalID = externalID }
func (s *stream) ExternalID() string              { return s.externalID }

func (s *stream) handleResetStream(rs *ResetStreamFrame) {
	if s.state.RecvResetStream() != nil {
		// TODO: This is returning ErrorCodeStreamClose although the returned
		//       value may be something other than ErrorStreamClosed, as
		//       RecvResetStream will return any previous error captured by the
		//       state. Same applies to handleData.
		err := s.reset(ErrorCodeStreamClosed)
		fmt.Printf("Error sending reset during recv reset: %s\n", err)
		return
	}
	s.state.Close()
	s.state.SetError(&StreamResetError{
		Reason: rs.ErrorCode,
	})
	s.reader.internalClose()
}

func (s *stream) handleData(data *DataFrame) {
	if err := s.state.RecvData(); err != nil {
		err = s.reset(ErrorCodeStreamClosed)
		fmt.Printf("Error sending reset during recv reset: %s\n", err)
		return
	}
	s.reader.Enqueue(data.Payload)
	if data.EndStream {
		s.state.CloseRemote()
	}
}

func (s *stream) write(msg Framer) error {
	fr := msg.IntoFrame()
	fr.StreamID = s.id
	return s.c.Write(fr)
}

func (s *stream) Write(data []byte, endStream bool) error {
	if err := s.state.SendData(); err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	for _, fr := range DataFramesFromBuffer(s.id, endStream, data) {
		if err := s.write(fr); err != nil {
			return err
		}
	}

	if endStream {
		s.state.CloseLocal()
	}

	return nil
}

func (s *stream) reset(code ErrorCode) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.write(&ResetStreamFrame{
		StreamID:  s.id,
		ErrorCode: code,
	})
}

func (s *stream) Reset(code ErrorCode) error {
	if err := s.state.SendResetStream(); err != nil {
		return err
	}

	s.state.Close()
	s.reader.internalClose()
	return s.reset(code)
}

func (s *stream) Read(into []byte) (int, error) {
	if s.state.err != nil {
		return 0, s.state.err
	}
	ok, n, err := s.reader.TryRead(into)
	if ok {
		return n, err
	}

	if err := s.state.RecvData(); err != nil {
		return 0, err
	}

	return s.reader.Read(into)
}

func (s *stream) CloseLocal() error {
	if err := s.state.SendData(); err != nil {
		return err
	}
	s.state.CloseLocal()
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	return s.write(&DataFrame{
		StreamID:  s.id,
		EndData:   true,
		EndStream: true,
		Payload:   nil,
	})
}
