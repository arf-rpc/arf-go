package wire

func DataFramesFromBuffer(streamID uint32, endStream bool, buffer []byte) []Framer {
	bufLen := len(buffer)
	if bufLen <= maxPayload {
		return []Framer{
			&DataFrame{
				StreamID:  streamID,
				EndData:   true,
				EndStream: endStream,
				Payload:   buffer,
			},
		}
	}

	var frames []Framer
	frames = append(frames, &DataFrame{
		StreamID:  streamID,
		EndData:   false,
		EndStream: endStream,
		Payload:   buffer[0:maxPayload],
	})
	written := maxPayload

	for {
		toWrite := min(bufLen-written, maxPayload)
		endData := bufLen-written-toWrite == 0
		frames = append(frames, &DataFrame{
			StreamID:  streamID,
			EndData:   endData,
			EndStream: endStream,
			Payload:   buffer[written : written+toWrite],
		})
		written += toWrite
		if endData {
			break
		}
	}

	return frames
}
