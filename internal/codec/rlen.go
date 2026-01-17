package codec

const (
	maxRunLength uint8 = 255
)

type RLENCodec struct {
	byteLength int
}

func equalSlice(slice1 []byte, slice2 []byte) bool {
	// slice comparison
	if len(slice1) != len(slice2) {
		return false
	}
	for i, e := range slice1 {
		if slice2[i] != e {
			return false
		}
	}
	return true
}

func (RC RLENCodec) EncodeBlock(src []byte) ([]byte, error) {
	var (
		runLen   uint8  = 1        // current length of the run
		runBytes []byte = nil      // current bytes being repeated
		srcIdx   int    = 0        // index as you traverse the source
		srcBytes []byte = nil      // current bytes from the source
		srcLen   int    = len(src) // length of input
	)

	if srcLen == 0 {
		return []byte{}, nil
	}
	if srcLen < RC.byteLength {
		encodedMessage := []byte{1}
		encodedMessage = append(encodedMessage, src...)
		return encodedMessage, nil
	}
	runLen = 1
	encodedMessage := make([]byte, 0, (srcLen/RC.byteLength+1)*(RC.byteLength+1))
	runBytes = src[:RC.byteLength]
	for srcIdx = RC.byteLength; srcIdx+RC.byteLength <= srcLen; srcIdx += RC.byteLength {
		srcBytes = src[srcIdx : srcIdx+RC.byteLength] // get next set of bytes from the source
		if equalSlice(runBytes, srcBytes) && runLen < maxRunLength {
			runLen++ // count them if they match the previous bytes
			continue
		}
		encodedMessage = append(encodedMessage, runLen)      // add the run length
		encodedMessage = append(encodedMessage, runBytes...) // add the run bytes
		runBytes = srcBytes                                  // set the run bytes to the new bytes
		runLen = 1                                           // reset the run length
	}
	encodedMessage = append(encodedMessage, runLen) // flush the final run
	encodedMessage = append(encodedMessage, runBytes...)
	if rem := srcLen % RC.byteLength; rem != 0 { // if there are leftover bytes (sub byteLength)
		encodedMessage = append(encodedMessage, 1)                   // add the run length
		encodedMessage = append(encodedMessage, src[srcLen-rem:]...) // add the run bytes
	}
	return encodedMessage, nil
}

func (RC RLENCodec) DecodeBlock(src []byte) ([]byte, error) {
	var (
		outLen int = 0        // length of the output
		outIdx int = 0        // how many bytes have been decoded
		srcIdx int = 0        // index as you traverse the source
		srcLen int = len(src) // how long is the input
		runLen int = 0        // how long is the current run
	)
	if srcLen == 0 {
		return []byte{}, nil
	}
	if srcLen <= RC.byteLength {
		return src[1:], nil
	}
	for srcIdx < srcLen { // for each (count, byte) pair
		runLen = int(src[srcIdx])                        // count how many bytes will be added
		srcIdx++                                         // jump to the next pair
		if rem := srcLen - srcIdx; RC.byteLength > rem { // if a short chunk is remaining
			outLen += rem // add it to the output length
			break
		}
		outLen += runLen * RC.byteLength
		srcIdx += RC.byteLength
	}
	decodedMessage := make([]byte, outLen) // make the array you need for output
	srcIdx = 0                             // keep track of where you are in the input
	for srcIdx < srcLen {
		runLen = int(src[srcIdx])
		srcIdx++
		if rem := srcLen - srcIdx; RC.byteLength > rem {
			for _, elem := range src[srcIdx:] {
				decodedMessage[outIdx] = elem // add the remaining bytes to the output
				outIdx++
			}
			break
		}
		for range runLen { // for every repetition
			for _, elem := range src[srcIdx : srcIdx+RC.byteLength] {
				decodedMessage[outIdx] = elem // loop through and add the bytes repeated
				outIdx++
			}
		}
		srcIdx += RC.byteLength // jump to the next run-bytes pair
	}
	return decodedMessage, nil
}

func (RLENCodec) IsLossless() bool {
	return true
}
