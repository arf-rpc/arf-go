package wire

type FrameKind uint8

func (k FrameKind) String() string {
	v, ok := frameNames[k]
	if !ok {
		return "UNKNOWN"
	}
	return v
}

const (
	FrameKindHello       FrameKind = 0x00
	FrameKindPing        FrameKind = 0x01
	FrameKindGoAway      FrameKind = 0x02
	FrameKindMakeStream  FrameKind = 0x03
	FrameKindResetStream FrameKind = 0x04
	FrameKindData        FrameKind = 0x05
)

var frameNames = map[FrameKind]string{
	FrameKindHello:       "HELLO",
	FrameKindPing:        "PING",
	FrameKindGoAway:      "GO_AWAY",
	FrameKindMakeStream:  "MAKE_STREAM",
	FrameKindResetStream: "RESET_STREAM",
	FrameKindData:        "DATA",
}

var frameFromByte = map[byte]FrameKind{
	byte(FrameKindHello):       FrameKindHello,
	byte(FrameKindPing):        FrameKindPing,
	byte(FrameKindGoAway):      FrameKindGoAway,
	byte(FrameKindMakeStream):  FrameKindMakeStream,
	byte(FrameKindResetStream): FrameKindResetStream,
	byte(FrameKindData):        FrameKindData,
}
