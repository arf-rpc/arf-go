package proto

const boolFlagMask = 0x01 << 4

func decodeBoolean(v byte) bool {
	return v&boolFlagMask == boolFlagMask
}

func encodeBoolean(b bool) []byte {
	v := byte(TypeBoolean)
	if b {
		v |= boolFlagMask
	}
	return []byte{v}
}
