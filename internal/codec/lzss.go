package codec

import (
	"squish/internal/sqerr"
	"sync"
)

// ======================================================================================
// Lempel–Ziv–Storer–Szymanski (LZSS)
//
// This codec works by taking data that has been seen before and replacing it with a
// lookback and match-length reference that is two bytes long.
//
// The two byte representation for lookback and match length is split such that 12 bits
// are used for lookback and 4 bites are used for match length. This allows for longer
// lookbacks and more reasonably represents the typical length of matches.
//
// ex: lookback = 6 and matchlength = 4
//              = 00000000 0110     = 0100 ---> 00000000 01100100 = (0, 100)
//
// Every set of 8 literals and lookback-match-length pairs are preceded by a single bytes
// whos bits represent whether the next chunk is a one byte literal, or a two byte match.
//
// ex: [0 1 2 3 0 1 0 1 2 3 4] -> [2 0 1 2 3 0 1 0 100 4]
//                                 |             |___|___________________________
//                                 |-> Flag Byte | Lookback & Match Length tuple |
//                                 |-> 0000 0010
// ======================================================================================

const (
	maxLookBack        = 1<<12 - 1              // 4095 - how far back to look for matches
	minMatchLen  int32 = 3                      // min match length
	maxMatchLen        = 1<<4 - 1 + minMatchLen // 18 - how far forward you can match (including min match)
	maxMatchIter       = 32                     // number of hash matches to look back through before halting
	hashSize           = 1 << 16                // possible hash matches
)

type LZSSCodec struct{}

type lzssWorkspace struct {
	head       [hashSize]int32        // most recent match of hashed 3-byte sequence
	prev       [maxLookBack + 1]int32 // previous matches
	groupBytes []byte                 // current values corresponding to the current flag byte
}

var lzssPool = sync.Pool{
	// creates a new workspace
	New: func() any {
		return &lzssWorkspace{}
	},
}

func balanceBytes(lookBack int32, matchLen int32) []byte {
	// combine lookback and match length into two byte representation with 12 bits for lookback and 4 bits for match length
	a := byte((lookBack >> 4) & 0xFF)
	b := byte(((lookBack << 4) & 0xF0) | ((matchLen - minMatchLen) & 0x0F))
	return []byte{a, b}
}

func splitBytes(a byte, b byte) (int32, int32) {
	// split lookback and match length into individual values from two byte representation
	lookback := (int32(a) << 4) | int32((b>>4)&0x0F)
	matchLen := int32(b&0x0F) + minMatchLen
	return lookback, matchLen
}

func hashBytes(bytes []byte) int {
	// determine a quasi-unique integer value from some bytes
	hash := 0
	for i := range len(bytes) {
		hash = (hash << 6) | int(bytes[i])
	}
	return hash & (hashSize - 1)
}

func (LZSSCodec) EncodeBlock(src []byte) ([]byte, error) {
	// encode src using the Lempel–Ziv–Storer–Szymanski algorithm
	var (
		out                = make([]byte, 0, len(src)*9/8)
		srcIdx       int32 = 0
		flagIdx      int
		flagByte     byte
		hash         int // hash of the current minMatchLen bytes
		curMatchIdx  int32
		curMatchLen  int32
		bestMatchIdx int32
		bestMatchLen int32
		iterations   int32 // number of iterations of checking matches
		lookBackIdx  int32 // index of the lookback window start
		srcLen       = int32(len(src))
	)
	// instantiate your workspace
	ws := lzssPool.Get().(*lzssWorkspace)
	defer lzssPool.Put(ws)
	// set all your matches to -1, meaning no-match
	for i := range len(ws.head) {
		ws.head[i] = -1
	}
	for i := range len(ws.prev) {
		ws.prev[i] = -1
	}
	for srcIdx < srcLen {
		// make a new flag byte and associated group of match
		flagIdx = 7
		flagByte = 0
		ws.groupBytes = ws.groupBytes[:0]
		for flagIdx >= 0 {
			// break out early if you match out of data before finishing a flag byte
			if srcIdx >= srcLen {
				break
			}
			bestMatchLen = 0
			if srcIdx+minMatchLen <= srcLen {
				// hash the next minMatchLen number of bytes
				// use that hash to get the most recent potential match of that same hash using the head array
				iterations = 0
				hash = hashBytes(src[srcIdx : srcIdx+minMatchLen])
				curMatchIdx = ws.head[hash]
				lookBackIdx = max(0, srcIdx-maxLookBack)
				// if no match, then loop through the prev array looking at previous matches of that hash
				// repeat until a match is found or the search expires (iterations -> maxMatchIter)
				for curMatchIdx != -1 && curMatchIdx >= lookBackIdx && iterations < maxMatchIter {
					// while there is a match and it is within the lookback window and you are under the max match checks
					curMatchLen = 0
					for curMatchLen < maxMatchLen && src[curMatchIdx+curMatchLen] == src[srcIdx+curMatchLen] {
						// while still matching and the match isn't too long
						curMatchLen++
						// break out if the match reached the end of the input or overlaps with current hashed bytes
						if srcIdx+curMatchLen >= srcLen || curMatchIdx+curMatchLen >= srcIdx {
							break
						}
					}
					// save it off if the match ends and it is the best encountered so far
					if curMatchLen >= minMatchLen && curMatchLen > bestMatchLen {
						bestMatchIdx = srcIdx - curMatchIdx
						bestMatchLen = curMatchLen
						if bestMatchLen == maxMatchLen {
							break
						}
					}
					curMatchIdx = ws.prev[curMatchIdx%(maxLookBack+1)]
					iterations++
				}
			}
			// add the best match or just the literals to the flag group
			start := srcIdx
			if bestMatchLen >= minMatchLen {
				flagByte |= (1 << flagIdx)
				ws.groupBytes = append(ws.groupBytes, balanceBytes(bestMatchIdx, bestMatchLen)...)
				srcIdx += bestMatchLen
			} else {
				ws.groupBytes = append(ws.groupBytes, src[srcIdx])
				srcIdx++
			}
			// update head and prev arrays with skipped byte hashes
			end := srcIdx
			for k := start; k < end; k++ {
				if minMatchLen+k <= srcLen {
					hash = hashBytes(src[k : k+minMatchLen])
					ws.prev[k%(maxLookBack+1)] = ws.head[hash]
					ws.head[hash] = k
				}
			}
			flagIdx--
		}
		out = append(out, flagByte)
		out = append(out, ws.groupBytes...)
	}
	return out, nil
}

func (LZSSCodec) DecodeBlock(src []byte) ([]byte, error) {
	// decode src using the Lempel–Ziv–Storer–Szymanski algorithm
	if len(src) == 0 {
		return []byte{}, nil
	}
	var (
		srcIdx   int32
		flagByte byte
		flagBit  byte
		flagIdx  int
		outLen   int32
		lookback int32 // current look back value
		matchLen int32 // current match length value
		srcLen   = int32(len(src))
	)
	// first pass for determining output length
	for srcIdx < srcLen {
		flagByte = src[srcIdx]
		srcIdx++
		for flagIdx = 7; flagIdx >= 0; flagIdx-- {
			// read flag bit to determine how to process next bytes
			flagBit = (flagByte >> flagIdx) & 0x01
			if flagBit == 0 {
				// copy a literal to the output
				outLen++
				srcIdx++
			} else {
				// if it is a lookback-match-length pair, copy literal from 'lookback' back in the output
				if srcIdx+1 >= srcLen {
					return []byte{}, sqerr.New(sqerr.Corrupt, "Invalid LZSS format")
				}
				matchLen = int32(src[srcIdx+1]&0x0F) + minMatchLen
				if matchLen < minMatchLen {
					return []byte{}, sqerr.New(sqerr.Corrupt, "Invalid match length in LZSS")
				}
				outLen += matchLen
				srcIdx += 2
			}
			if srcIdx > srcLen {
				break
			}
		}
	}
	// second pass to actually decode it
	srcIdx = 0
	out := make([]byte, 0, outLen)
	for srcIdx < srcLen {
		flagByte = src[srcIdx]
		srcIdx++
		for flagIdx = 7; flagIdx >= 0; flagIdx-- {
			if srcIdx >= srcLen {
				break
			}
			// read flag bit to determine how to process next bytes
			flagBit = (flagByte >> flagIdx) & 0x01
			if flagBit == 0 {
				// copy literals to output
				out = append(out, src[srcIdx])
				srcIdx++
			} else {
				// copy range for lookback-match-length pairs
				lookback, matchLen = splitBytes(src[srcIdx], src[srcIdx+1])
				for range matchLen {
					out = append(out, out[int32(len(out))-lookback])
				}
				srcIdx += 2
			}
		}
	}
	return out, nil
}

func (LZSSCodec) IsLossless() bool {
	return true
}
