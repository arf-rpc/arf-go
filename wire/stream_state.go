package wire

type streamStateCode int

const (
	streamStateOpen streamStateCode = iota
	streamStateHalfClosedLocal
	streamStateHalfClosedRemote
	streamStateClosed
)

func (s streamStateCode) String() string {
	switch s {
	case streamStateOpen:
		return "open"
	case streamStateHalfClosedLocal:
		return "half-closed (local)"
	case streamStateHalfClosedRemote:
		return "half-closed (remote)"
	case streamStateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

type streamState struct {
	code streamStateCode
	err  error
}

func (s *streamState) Error() error {
	return s.err
}

func (s *streamState) Close() {
	s.code = streamStateClosed
}

func (s *streamState) CloseLocal() {
	switch s.code {
	case streamStateOpen:
		s.code = streamStateHalfClosedLocal
	case streamStateHalfClosedLocal:
		s.code = streamStateHalfClosedLocal
	case streamStateHalfClosedRemote:
		s.code = streamStateClosed
	case streamStateClosed:
		s.code = streamStateClosed
	}
}

func (s *streamState) CloseRemote() {
	switch s.code {
	case streamStateOpen:
		s.code = streamStateHalfClosedRemote
	case streamStateHalfClosedLocal:
		s.code = streamStateClosed
	case streamStateHalfClosedRemote:
		s.code = streamStateHalfClosedRemote
	case streamStateClosed:
		s.code = streamStateClosed
	}
}

func (s *streamState) SetError(err error) {
	s.err = err
}

func (s *streamState) RecvResetStream() error {
	switch {
	case s.err != nil:
		return s.err
	case s.code == streamStateClosed:
		return ClosedStreamErr
	default:
		return nil
	}
}

func (s *streamState) RecvData() error {
	switch {
	case s.err != nil:
		return s.err
	case s.code == streamStateClosed:
		return ClosedStreamErr
	case s.code == streamStateHalfClosedRemote:
		return ClosedStreamErr
	default:
		return nil
	}
}

func (s *streamState) SendResetStream() error {
	switch {
	case s.err != nil:
		return s.err
	case s.code == streamStateClosed:
		return ClosedStreamErr
	case s.code == streamStateHalfClosedLocal:
		return ClosedStreamErr
	default:
		return nil
	}
}

func (s *streamState) SendData() error {
	switch {
	case s.err != nil:
		return s.err
	case s.code == streamStateClosed:
		return ClosedStreamErr
	case s.code == streamStateHalfClosedLocal:
		return ClosedStreamErr
	default:
		return nil
	}
}
