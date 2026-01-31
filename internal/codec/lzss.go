package codec

const (
	maxLookBack  = 1<<12 - 1              // 4095 - how far back to look for matches
	minMatchLen  = 3                      // min match length
	maxMatchLen  = 1<<4 - 1 + minMatchLen // 15 - how far forward you can match (after min match)
	maxMatchIter = 32                     // number of hash matches to look back through before halting
	hashSize     = 1 << 16
)

type LZSSCodec struct{}

func balanceBytes(lookBack int, runLen int) []byte {
	a := byte((lookBack >> 4) & 0xFF)                                     // keep the 8 MSb of the lookback in one byte
	b := byte(((lookBack << 4) & 0xF0) | ((runLen - minMatchLen) & 0x0F)) // and 4 LSb of lookback + 4 bit length in other byte
	return []byte{a, b}
}

func splitBytes(a byte, b byte) (int, int) {
	lookback := (int(a) << 4) | int((b>>4)&0x0F) // lookback is first byte + 4 msb of second bit
	runLen := int(b&0x0F) + minMatchLen          // length is lsb bits of second byte + minimum match length
	return lookback, runLen
}

func hashBytes(bytes []byte) int {
	hash := 0
	for i := range len(bytes) {
		hash = (hash << 8) | int(bytes[i])
	}
	return hash & (hashSize - 1)
}

func (LZSSCodec) EncodeBlock(src []byte) ([]byte, error) {
	var (
		head         [hashSize]int                                        // most recent match of hashes 3-byte sequence
		prev         [maxLookBack + 1]int                                 // previous matches
		output       []byte               = make([]byte, 0, len(src)*9/8) // output byte slice
		srcIdx       int                  = 0                             // where you are in the input
		matchStream  []byte               = make([]byte, 0, 16)           // current matching values corresponding to flag bits
		flagIdx      int                                                  // where you are in processing flags
		flagByte     byte                                                 // the flag byte
		hash         int                                                  // hash of the next minMatchLen bytes
		curMatchIdx  int                                                  // where the current match is in the input
		curMatchLen  int                                                  // how long it is
		bestMatchLen int                                                  // best match length per 3 byte hash
		bestLookBack int                                                  // lookback for that best match
		iterations   int                                                  // number of iterations of checking matches
		lookBackIdx  int                                                  // index of the lookback window start
	)
	for i := range len(head) {
		head[i] = -1 // set the head hash-match index mapping array to -1
	}
	for i := range len(prev) {
		prev[i] = -1 // set the list of previous matches to -1
	}
	for srcIdx < len(src) {
		flagIdx = 7                   // start at the msb of the flag
		flagByte = 0                  // reset the flag
		matchStream = matchStream[:0] // wipe the match stream
		for flagIdx >= 0 {            // loop through the flag bits
			if srcIdx >= len(src) {
				break // dip if you run out of source before finishing the flag byte
			}
			bestMatchLen = 0                    // reset you best match
			bestLookBack = -1                   // and best look back
			if srcIdx+minMatchLen <= len(src) { // don't go out of bounds
				iterations = 0                                     // reset your iterations of match checks
				hash = hashBytes(src[srcIdx : srcIdx+minMatchLen]) // get the hash of the current three consecutive bytes
				curMatchIdx = head[hash]                           // get the index of the last match of that hash
				lookBackIdx = max(0, srcIdx-maxLookBack)           // determine the lookback value
				for curMatchIdx != -1 &&                           // while there is a match within the window
					curMatchIdx >= lookBackIdx && // and the match is within the window
					iterations < maxMatchIter { // and you haven't exceeded your max iterations
					curMatchLen = 0                  // reset the length of the match
					for curMatchLen < maxMatchLen && // while you haven't achieved the longest match possible
						srcIdx+curMatchLen < len(src) && // your front match isn't extending past the source data
						curMatchIdx < srcIdx && // you aren't getting ahead of yourself... literally
						src[curMatchIdx+curMatchLen] == src[srcIdx+curMatchLen] { // and the match continues
						curMatchLen++ // keep counting
					}
					if curMatchLen >= minMatchLen && curMatchLen > bestMatchLen { // save it off if is the best match yet
						bestMatchLen = curMatchLen
						bestLookBack = srcIdx - curMatchIdx
						if bestMatchLen == maxMatchLen {
							break
						}
					}
					curMatchIdx = prev[curMatchIdx%(maxLookBack+1)] // grab the next match if it is still iterating
					iterations++                                    // count it
				}
			}
			start := srcIdx                  // where does the match start
			if bestMatchLen >= minMatchLen { // for matches
				flagByte |= (1 << flagIdx)                                                     // add a 1 bit to the flag
				matchStream = append(matchStream, balanceBytes(bestLookBack, bestMatchLen)...) // add the look back + length bytes
				srcIdx += bestMatchLen                                                         // increment where you are in the source data
			} else { // for literals
				matchStream = append(matchStream, src[srcIdx]) // add the literal
				srcIdx++                                       // increment where you are in the source data
			}
			end := srcIdx                  // keep track of where the match ended
			for k := start; k < end; k++ { // loop through all the 3 byte chunks from the beginning of the match to the end
				if minMatchLen+k <= len(src) { // don't read past the end
					hash = hashBytes(src[k : k+minMatchLen]) // get the hash of the next bytes
					prev[k%(maxLookBack+1)] = head[hash]     // update the old matches
					head[hash] = k                           // update the newest most recent match
				}
			}
			flagIdx--
		}
		output = append(output, flagByte)
		output = append(output, matchStream...)
	}
	return output, nil
}

func (LZSSCodec) DecodeBlock(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return []byte{}, nil
	}
	var (
		srcIdx   int  // where you are in the input
		flagByte byte // current flag byte
		flagBit  byte // current flag bit
		flagIdx  int  // current flag bit index
		outLen   int  // length of decoded data
		lookback int  // how far to looking back
		runLen   int  // how long a run is

	)
	for srcIdx < len(src) { // scan through the input to count how long the output will be
		flagByte = src[srcIdx]                     // get the current flag byte
		srcIdx++                                   // move past the flag byte
		for flagIdx = 7; flagIdx >= 0; flagIdx-- { // loop through the flag bits
			flagBit = (flagByte >> flagIdx) & 0x01 // grab the bit
			if flagBit == 0 {                      // if it is a literal
				outLen++ // increase the output length by one byte
				srcIdx++ // move forward as you scan through the source
			} else if srcIdx+1 < len(src) {
				outLen += int(src[srcIdx+1]&0x0F) + minMatchLen // increase the output by the length of the run
				srcIdx += 2                                     // move forward as you scan through the source
			}
			if srcIdx > len(src) {
				break
			}
		}
	}
	srcIdx = 0
	output := make([]byte, 0, outLen) // make the output byte slice
	for srcIdx < len(src) {           // scan through the input to count how long the output will be
		flagByte = src[srcIdx]                     // get the current flag byte
		srcIdx++                                   // move forward in the input
		for flagIdx = 7; flagIdx >= 0; flagIdx-- { // loop through the flag bits
			if srcIdx >= len(src) {
				break
			}
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
		}
	}
	return output, nil
}

func (LZSSCodec) IsLossless() bool {
	return true
}
