package codec

const (
	maxZeroRunLength uint8 = 255
)

type ZRLECodec struct {
	byteLength int
	lossless   bool
}

func (ZRLECodec) EncodeBlock(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return src, nil
	}
	var (
		runLen   uint8  = 0                         // length of current run
		srcIdx   int    = 0                         // index as you traverse the source
		outBytes []byte = make([]byte, 0, len(src)) // encoded bytes
	)
	for srcIdx < len(src) {
		if src[srcIdx] == 0x00 && runLen < maxZeroRunLength {
			if runLen == 0 {
				outBytes = append(outBytes, 0x00)
			}
			runLen++
		} else {
			if runLen > 0 {
				outBytes = append(outBytes, runLen)
				runLen = 0
			}
			outBytes = append(outBytes, src[srcIdx])
		}
		srcIdx++
	}
	return outBytes, nil
}

func (ZRLECodec) DecodeBlock(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return src, nil
	}
	var (
		srcIdx    = 0 // where you are in the source
		outLength = 0 // first pass variable for allocating for decoding
	)
	for srcIdx < len(src) {
		if src[srcIdx] == 0x00 {
			srcIdx++
			outLength += int(src[srcIdx])
		}
		outLength++
		srcIdx++
	}
	srcIdx = 0
	outBytes := make([]byte, 0, outLength)
	for srcIdx < len(src) {
		if src[srcIdx] == 0x00 {
			srcIdx++
			for range src[srcIdx] {
				outBytes = append(outBytes, 0x00)
			}
		} else {
			outBytes = append(outBytes, src[srcIdx])
		}
		srcIdx++
	}
	return outBytes, nil
}

func (ZRLECodec) IsLossless() bool {
	return true
}
