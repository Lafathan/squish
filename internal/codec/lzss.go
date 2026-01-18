package codec

const (
	maxLookBack = 1<<12 - 1 // 4095 - how far back to look for matches
	maxMatchLen = 1<<4 - 1  // 15 - how far forward you can match (after min match)
	minMatchLen = 3         // min match length
)

type LZSSCodec struct{}

func findMatches(backward []byte, forward []byte) (byte, []byte) {
	return byte(0), []byte{}
}

func (LZSSCodec) EncodeBlock(src []byte) ([]byte, error) {
	// encodes src using a flag - lookback - run technique
	// ex: Mellow yellow fellow says hello! (length 32)
	// a byte where each bit representing 0 - byte literal or 1 - lookback-run pair
	// [00000000] M  e l l o w _ y
	// [10100000] 7  6 f 7 6 s a y s _
	// [01000000] h 12 4 !
	// result: (0x00)Mellow y(0xA0)76f76says (0x40)h(12)4 (length 24)
	//
	// lookback - run values are spread over two bytes
	// first 12 bits are lookback, last 4 bits are run length + 3 (since 3 is min length)
	var (
		srcIdx    int    = 0
		srcLen    int    = len(src)
		flagByte  byte   = 0
		flagIdx   int    = 7
		output    []byte = make([]byte, 0, len(src)*9/8)
		curStream []byte = make([]byte, 0, 17)
		flag      byte   = 0
		bytes     []byte = make([]byte, 0, 2)
		lookBack  int    = 0
		lookAhead int    = 0
	)
	for srcIdx < len(src) {
		lookBack = min(srcIdx-maxLookBack, 0)
		lookAhead = min(srcIdx+maxMatchLen+minMatchLen, srcLen)
		flag, bytes = findMatches(src[lookBack:srcIdx], src[srcIdx:lookAhead])
		flagByte |= (flag << flagIdx) & (1 << flagIdx)
		curStream = append(curStream, bytes[0])
		if flag > 0 {
			curStream = append(curStream, bytes[1])
		}
		if flagIdx == 0 {
			output = append(output, flagByte)
			output = append(output, curStream...)
			curStream = make([]byte, 0, 17)
			flagIdx = 7
		}
		srcIdx++
	}
	return src, nil
}

func (LZSSCodec) DecodeBlock(src []byte) ([]byte, error) {
	return src, nil
}

func (LZSSCodec) IsLossless() bool {
	return true
}
