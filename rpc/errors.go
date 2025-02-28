package rpc

import "fmt"

type MessageKindMismatchError struct {
	Expected MessageKind
	Received MessageKind
}

func (e *MessageKindMismatchError) Error() string {
	return fmt.Sprintf("expected kind %v but got %v", e.Expected, e.Received)
}

type NoStreamError struct {
	Recv bool
}

func (e *NoStreamError) Error() string {
	if e.Recv {
		return fmt.Sprintf("no incoming stream from server")
	} else {
		return fmt.Sprintf("no outgoing stream to server")
	}
}

type StreamEndError struct{}

func (e *StreamEndError) Error() string {
	return "stream has ended"
}

type StreamFailure struct {
	Msg string
}

func (e *StreamFailure) Error() string {
	return fmt.Sprintf("stream failed: %s", e.Msg)
}
