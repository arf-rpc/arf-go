package wire

import "fmt"

type ErrorCode uint32

const (
	ErrorCodeNoError            ErrorCode = 0x00
	ErrorCodeProtocolError      ErrorCode = 0x01
	ErrorCodeInternalError      ErrorCode = 0x02
	ErrorCodeStreamClosed       ErrorCode = 0x03
	ErrorCodeFrameSizeError     ErrorCode = 0x04
	ErrorCodeRefusedStream      ErrorCode = 0x05
	ErrorCodeCancel             ErrorCode = 0x06
	ErrorCodeCompressionError   ErrorCode = 0x07
	ErrorCodeEnhanceYourCalm    ErrorCode = 0x08
	ErrorCodeInadequateSecurity ErrorCode = 0x09
)

var errorToString = map[ErrorCode]string{
	ErrorCodeNoError:            "No error",
	ErrorCodeProtocolError:      "Protocol error",
	ErrorCodeInternalError:      "Internal error",
	ErrorCodeStreamClosed:       "Stream closed",
	ErrorCodeFrameSizeError:     "Frame size error",
	ErrorCodeRefusedStream:      "Refused stream",
	ErrorCodeCancel:             "Cancel",
	ErrorCodeCompressionError:   "Compression error",
	ErrorCodeEnhanceYourCalm:    "Enhance your calm",
	ErrorCodeInadequateSecurity: "Inadequate security",
}

func (e ErrorCode) String() string {
	if s, ok := errorToString[e]; ok {
		return s
	}
	return fmt.Sprintf("unknown error 0x%02x", uint32(e))
}
