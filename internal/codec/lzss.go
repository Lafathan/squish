package codec

const (
	maxLookBack = 1<<12 - 1              // 4095 - how far back to look for matches
	minMatchLen = 3                      // min match length
	maxMatchLen = 1<<4 - 1 + minMatchLen // 15 - how far forward you can match (after min match)
)

type LZSSCodec struct{}

func balanceBytes(lookBack int, runLength int) []byte {
	a := byte((lookBack >> 4) & 0xFF)                        // keep the 8 MSb of the lookback in one byte
	b := byte(((lookBack << 4) & 0xF0) | (runLength & 0x0F)) // and 4 LSb of lookback + 4 bit length in other byte
	return []byte{a, b}
}

func splitBytes(byte1 byte, byte2 byte) (int, int) {
	lookback := (int(byte1) << 4) | int((byte2>>4)&0x0F)
	runLen := int(byte2&0x0F) + minMatchLen
	return lookback, runLen
}

func findMatches(backward []byte, forward []byte) (byte, []byte) {
	var (
		matched     bool = true // whether or not the we are currently matched
		matchCount  int  = 0    // length of current match
		maxMatch    int  = 0    // length of longest match so far
		maxMatchIdx int  = 0    // index of longest match so far
	)
	for i := range len(backward) - len(forward) { // stop when i == start of forward
		matchCount = 0                                        // reset match count for each element in lookback window
		matched = backward[matchCount] == forward[matchCount] // grab the first potential match
		for matched {                                         // while we are matched
			matchCount++ // keep counting up how long the match is
			if i+matchCount >= len(backward) || matchCount >= len(forward) {
				break // break out if we are about to exceed the length of anything being compared
			}
			matched = backward[i+matchCount] == forward[matchCount] // are we still matched?
		}
		if matchCount > maxMatch {
			maxMatch = matchCount // keep the longest match
			maxMatchIdx = i       // store the index of the longest match so far
		}
		if matchCount == maxMatchLen {
			break // we found the longest possible match, break
		}
		i++ // go to the next element in the lookback window
	}
	if maxMatch > minMatchLen { // make sure the longest match is even worth encoding
		return byte(1), balanceBytes(len(backward)-maxMatchIdx, maxMatch-minMatchLen)
	}
	return byte(0), []byte{forward[0]} // otherwise return the literal
}

func (LZSSCodec) EncodeBlock(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return []byte{}, nil
	}
	// encodes src using a flagBit - lookback - run technique
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
		srcIdx    int    = 0                             // where you are in the input
		srcLen    int    = len(src)                      // length of the input
		flagByte  byte   = 0                             // value of the flag byte
		flagIdx   int    = 7                             // index of flag byte to be edited
		output    []byte = make([]byte, 0, len(src)*9/8) // output byte slice
		curStream []byte = make([]byte, 0, 16)           // current stream of output bytes corresponding to the flag byte
		flagBit   byte   = 0                             // flag bit value
		bytes     []byte = make([]byte, 0, 2)            // literal byte or lookback-run byte pair
		lookBack  int    = 0                             // lookback value (changes at start of block
		lookAhead int    = 0                             // lookAhead value (changes at end of block)
	)
	for srcIdx < srcLen {
		lookBack = max(srcIdx-maxLookBack, 0)                                        // get valid lookback value
		lookAhead = min(srcIdx+maxMatchLen, srcLen)                                  // get valid lookahead value
		flagBit, bytes = findMatches(src[lookBack:lookAhead], src[srcIdx:lookAhead]) // get flag bit and literal / lookback-run bytes
		flagByte |= (flagBit << flagIdx) & (1 << flagIdx)                            // add flagbit to the flagByte
		curStream = append(curStream, bytes[0])                                      // add first byte to stream (literal or lookback)
		if flagBit > 0 {                                                             // if it was a lookback-run byte pair
			curStream = append(curStream, bytes[1])    // add the run length bytes too
			srcIdx += int(bytes[1]&0x0F) + minMatchLen // jump ahead to the end of the current run
		} else {
			srcIdx++ // move to the next byte
		}
		if flagIdx == 0 { // if you are at the end of the current flag byte
			output = append(output, flagByte)     // append the flag byte to the output
			output = append(output, curStream...) // append the current stream of encoded data to the output
			curStream = curStream[:0]             // remake the current stream
			flagIdx = 7                           // reset the flag index
			flagByte = 0
		} else {
			flagIdx-- // decrement the flag index
		}
	}
	output = append(output, flagByte)     // append the flag byte to the output
	output = append(output, curStream...) // append the current stream of encoded data to the output
	return output, nil
}

func (LZSSCodec) DecodeBlock(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return []byte{}, nil
	}
	var (
		flagByte byte            // current flag byte
		flagBit  byte            // current flag bit
		flagIdx  int             // current flag bit index
		srcLen   int  = len(src) // length of the input
		srcIdx   int             // where you are in the input
		outLen   int             // length of decoded data
		lookback int             // how far to looking back
		runLen   int             // how long a run is

	)
	for srcIdx < srcLen { // scan through the input to count how long the output will be
		flagByte = src[srcIdx]                     // get the current flag byte
		srcIdx++                                   // move past the flag byte
		for flagIdx = 7; flagIdx >= 0; flagIdx-- { // loop through the flag bits
			flagBit = (flagByte >> flagIdx) & 0x01 // grab the bit
			if flagBit == 0 {                      // if it is a literal
				outLen++ // increase the output length by one byte
				srcIdx++ // move forward as you scan through the source
			} else {
				outLen += int(src[srcIdx+1]&0x0F) + minMatchLen // increase the output by the length of the run
				srcIdx += 2                                     // move forward as you scan through the source
			}
			if srcIdx > srcLen {
				break
			}
		}
	}
	srcIdx = 0
	output := make([]byte, 0, outLen) // make the output byte slice
	for srcIdx < srcLen {             // scan through the input to count how long the output will be
		flagByte = src[srcIdx]                     // get the current flag byte
		srcIdx++                                   // move forward in the input
		for flagIdx = 7; flagIdx >= 0; flagIdx-- { // loop through the flag bits
			flagBit = (flagByte >> flagIdx) & 0x01 // grab the bit
			if flagBit == 0 {                      // if it is a literal
				output = append(output, src[srcIdx]) // add the literal to the output
				srcIdx++                             // move forward as you scan through the source
			} else {
				lookback, runLen = splitBytes(src[srcIdx], src[srcIdx+1]) // get the reference details
				for range runLen {
					output = append(output, output[len(output)-lookback]) // copy the match up to the front
				}
				srcIdx += 2 // move forward as you scan through the source
			}
			if srcIdx > srcLen-1 {
				break
			}
		}
	}
	return output, nil
}

func (LZSSCodec) IsLossless() bool {
	return true
}
