package wire

import (
	"net"
	"sync"
)

type server interface {
	connectionClosed(id int)
	StreamHandler
}

type StreamHandler interface {
	ServiceStream(stream Stream)
	CancelStream(stream Stream)
}

type Server struct {
	listener      net.Listener
	streamHandler StreamHandler

	connectionsMu sync.Mutex
	connections   map[int]*Conn
	connID        int
}

func NewServer(l net.Listener, handler StreamHandler) *Server {
	return &Server{
		listener:      l,
		streamHandler: handler,
		connections:   make(map[int]*Conn),
	}
}

func (s *Server) connectionClosed(id int) {
	s.connectionsMu.Lock()
	defer s.connectionsMu.Unlock()
	delete(s.connections, id)
}

func (s *Server) Serve() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return err
		}
		s.connectionsMu.Lock()
		c := NewConn(s, s.connID, conn)
		s.connID++
		s.connections[s.connID] = c
		s.connectionsMu.Unlock()
	}
}

func (s *Server) ServiceStream(stream Stream) {
	go s.streamHandler.ServiceStream(stream)
}

func (s *Server) CancelStream(stream Stream) {
	go s.streamHandler.CancelStream(stream)
}

func (s *Server) Shutdown() error {
	s.connectionsMu.Lock()
	defer s.connectionsMu.Unlock()
	err := s.listener.Close()
	for _, conn := range s.connections {
		conn.goAway(ErrorCodeNoError, nil, true)
	}
	return err
}
