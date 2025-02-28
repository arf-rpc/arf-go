package wire

import "fmt"

type FrameTypeMismatchError struct {
	Expected, Received FrameKind
}

func (f *FrameTypeMismatchError) Error() string {
	return fmt.Sprintf("frame type mismatch: expected %s, got %s", f.Expected, f.Received)
}

type UnexpectedUnassociatedFrameError struct {
	Kind FrameKind
}

func (u *UnexpectedUnassociatedFrameError) Error() string {
	return fmt.Sprintf("frame %s must be associated to a stream", u.Kind)
}

type UnexpectedAssociatedFrameError struct {
	Kind FrameKind
}

func (u *UnexpectedAssociatedFrameError) Error() string {
	return fmt.Sprintf("frame %s must not be associated to a stream", u.Kind)
}

type InvalidFrameLengthError struct {
	message string
}

func (i *InvalidFrameLengthError) Error() string {
	return i.message
}

type InvalidFrameError struct {
	message string
}

func (i *InvalidFrameError) Error() string {
	return i.message
}

var ClosedStreamErr = fmt.Errorf("stream is closed")

type StreamResetError struct {
	Reason ErrorCode
}

func (c *StreamResetError) Error() string {
	desc, ok := errorToString[c.Reason]
	if !ok {
		return fmt.Sprintf("stream reset: unknown error code %02x", c.Reason)
	}

	return fmt.Sprintf("stream reset: %s", desc)
}

type StreamCanceledError struct {
	Reason ErrorCode
}

func (c *StreamCanceledError) Error() string {
	desc, ok := errorToString[c.Reason]
	if !ok {
		return fmt.Sprintf("stream canceled: unknown error code %02x", c.Reason)
	}

	return fmt.Sprintf("stream canceled: %s", desc)
}

type ConnectionResetError struct {
	Reason  ErrorCode
	Details string
}

func (c *ConnectionResetError) Error() string {
	if len(c.Details) > 0 {
		return fmt.Sprintf("connection reset: %s: %s", c.Reason, c.Details)
	}
	return fmt.Sprintf("connection reset: %s", c.Reason)
}
