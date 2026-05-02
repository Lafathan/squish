package codec

// ======================================================================================
// Run Length Encoding
//
// This codec works by taking streams of the same, or similar values in the data set and
// represents them with a n-byte run-length tuples. The run length is represented with a
// variable length integer.
//
// ex: [1 1 1 1 2 3 4 4 4 4 5 5] -> [1 4 2 3 4 4 5 2]
//
// The user has the option of selecting up to 4-byte strides to look for runs of. For
// example, an image with 3 color channels would likely compress better using RLE3.
//
// ex: data set         [0 0 0 0 0 0 1 2 1 2 1 2 3 4 5 3 4 5 3 4 5]
//     byte-stride 1 -> [0 6         1 2 1 2 1 2 3 4 5 3 4 5 3 4 5]
//     byte-stride 2 -> [0 0 3       1 2 3       3 4 5 3 4 5 3 4 5]
//     byte-stride 3 -> [0 0 0 2     1 2 1 2 1 2 3 4 5 3          ]
//
// Every 8 byte-length chunks are preceded by a single byte whos bits represent whether
// the next chunk is an n-byte run-length tuple, or just a set of raw bytes.
//
// For example, if the data is 4 unique bytes followed by 4 runs, the leading
// byte for that set would be 00001111 = 15.
//
// ex: [6 2 3 0 3 3 4 4 4 4 4 4 8 8 9 9 9 9] -> [15 6 2 3 0 3 2 4 6 8 2 9 4]
//
// The option is available to make the compression lossy which adds additional processing
// that intelligently determines when to consider a value the same as the previous value
// within some dynamic tolerance.
//
// For lossy encoding, streams of "close enough" btyes are streated as a stream of an
// anchor value. The tolerance for "close enough" is dynamic and changes relative to
// the variance in the dataset. Higher variance results in higher tolerance (more loss).
//
// ex: [0 0 0 0 0 0 0 0 1 2 3 4 5 6 7 8 9 9 9 9 7 7 7 7] -> [224 0 11 4 4 8 9]
//     [0 1 0 1 0 1 0 1 1 2 3 4 5 6 7 8 9 9 9 9 7 7 7 7] -> [192 0 13 6 11]
//     [0 1 2 0 1 2 0 1 1 2 3 4 5 6 7 8 9 9 9 9 7 7 7 7] -> [192 0 13 7 10]
//
// The variable length integer is defined in the binary package as the following:
// - unsigned integers are serialized 7 bits at a time, starting with the least
//   significant bits
// - the most significant bit (msb) in each output byte indicates if there is a
//   continuation byte (msb = 1)
// ======================================================================================

import (
	"encoding/binary"
	"squish/internal/sqerr"
)

const (
	tolAlpha float32 = 0.15 // tolerance sigma decay
	tolMin   float32 = 2.0  // residual that will always result in conforming to anchor
	tolMax   float32 = 6.0  // residual that will always result in a new anchor
	tolK     float32 = 1.5  // variance to tolerance factor
	tolBand  uint8   = 1    // wiggle allowance when considering new anchor candidate
	tolHang  uint8   = 3    // required repetitions for candidate to become new anchor
)

type RLECodec struct {
	byteLength int  // byte width for matching consecutive chunks
	lossless   bool // whether or not to use a tolerance for snapping
}

type RLTolerance struct {
	anchor    []byte    // anchor value to snap values to
	sigma     []float32 // current weighted error value
	tolerance []float32 // tolerance used for snapping
	candidate []byte    // potential new anchor
	count     []int     // current run length of candidate
}

func newTolerance(n int) *RLTolerance {
	return &RLTolerance{
		anchor:    make([]byte, n),
		sigma:     make([]float32, n),
		tolerance: make([]float32, n),
		candidate: make([]byte, n),
		count:     make([]int, n),
	}
}

func equalSliceWithinTolerance(slice1 []byte, slice2 []byte, tol []float32) bool {
	// element-wise slice comparison
	if len(slice1) != len(slice2) {
		return false
	}
	for i := range len(slice1) {
		if float32(absByteDiff(slice1[i], slice2[i])) > tol[i] {
			return false
		}
	}
	return true
}

func (t *RLTolerance) updateTolerance(data []byte) {
	// update tolerances based on new data point
	for i := range len(t.tolerance) {
		// grab some useful values
		s, d := t.sigma[i], data[i]
		// calculate a residual
		r := float32(absByteDiff(t.anchor[i], d))
		// calculate the new tolerance
		s += (r - s) * tolAlpha // s * (1 - tolAlpha) + r * tolAlpha
		t.sigma[i] = s
		t.tolerance[i] = clampFloat(tolMin+tolK*s, tolMin, tolMax)
		// track and update candidate for new anchor values
		if r <= t.tolerance[i] {
			// track repeats of valid candidates
			if absByteDiff(t.candidate[i], d) <= tolBand {
				t.count[i]++
			} else {
				t.candidate[i] = d
				t.count[i] = 1
			}
			// choose a new anchor if candidate repeats enough
			if t.count[i] >= int(tolHang) {
				t.anchor[i] = t.candidate[i]
				t.count[i] = 0
			}
		} else {
			// pick a new anchor if residual is way outside of the window
			t.anchor[i] = d
			t.sigma[i] = 0
			t.candidate[i] = d
			t.count[i] = 0
			t.tolerance[i] = tolMin
		}
	}
}

func encodeUpdateGroup(runLen uint32, flagByte *byte, flagBit uint8, runBytes []byte, groupBytes *[]byte) {
	// update group associated with current flag byte
	if runLen >= 2 {
		// update the current flag bit to represent a run, then append the length and the literal bytes
		*flagByte |= (1 << flagBit)
		*groupBytes = binary.AppendUvarint(*groupBytes, uint64(runLen))
		*groupBytes = append(*groupBytes, runBytes...)
	} else {
		// no need to update the flag bit (defaults to 0), then append the literal bytes
		*groupBytes = append(*groupBytes, runBytes...)
	}
}

func (RC RLECodec) EncodeBlock(src []byte) ([]byte, error) {
	// encode src using run-length encoding
	if len(src) == 0 {
		return src, nil
	}
	var (
		flagBit    uint8        = 7                                // current bit representing a pair or not
		flagByte   byte         = 0x00                             // byte holding flag bits
		runLen     uint32       = 1                                // current length of the run
		runBytes   []byte       = nil                              // current bytes being repeated
		groupBytes []byte       = make([]byte, 0, 8*RC.byteLength) // current set of encoded bytes
		srcIdx     int          = 0                                // index as you traverse the source
		srcBytes   []byte       = nil                              // current bytes from the source
		tol        *RLTolerance = newTolerance(RC.byteLength)      // noise and tolerance calculations
		out        []byte       = make([]byte, 0, len(src)*9/8)    // encoded bytes
	)
	for srcIdx < len(src) {
		srcBytes = src[srcIdx:min(srcIdx+RC.byteLength, len(src))]
		if !RC.IsLossless() && len(srcBytes) == RC.byteLength {
			tol.updateTolerance(srcBytes)
		}
		srcIdx += len(srcBytes)
		if runBytes == nil {
			// initialize if necessary
			runBytes = srcBytes
			runLen = 1
		} else if equalSliceWithinTolerance(runBytes, srcBytes, tol.tolerance) {
			// increment the run
			runLen++
		} else {
			// update group
			encodeUpdateGroup(runLen, &flagByte, flagBit, runBytes, &groupBytes)
			// update output and start a new flag byte if you run out of flag bits
			if flagBit == 0 {
				out = append(out, flagByte)
				out = append(out, groupBytes...)
				groupBytes = groupBytes[:0]
				flagBit = 7
				flagByte = 0x00
			} else {
				flagBit--
			}
			runBytes = srcBytes
			runLen = 1
		}
		// trailing match, write remaining length and literals
		if srcIdx >= len(src) {
			encodeUpdateGroup(runLen, &flagByte, flagBit, runBytes, &groupBytes)
			out = append(out, flagByte)
			out = append(out, groupBytes...)
		}
	}
	return out, nil
}

func decodeGetFlagAndRunLength(flagByte *byte, flagBit uint8, runLen *uint64, srcIdx *int, src []byte) error {
	// grab the current flag bit, read a new flag if necessary, and update to the next run length
	if flagBit == 7 {
		// read a new flag bit if necessary
		*flagByte = src[*srcIdx]
		*srcIdx++
	}
	if *srcIdx > len(src) {
		return sqerr.New(sqerr.Corrupt, "RLE encounterd early EOS")
	}
	// read the next run length if flag bit informs of run length
	if *flagByte&(1<<flagBit) > 0 {
		var bytes int
		*runLen, bytes = binary.Uvarint(src[*srcIdx:])
		if bytes < 1 {
			return sqerr.New(sqerr.Corrupt, "RLE encountered insufficient length run")
		}
		*srcIdx += bytes
	} else {
		*runLen = 1 // otherwise it is just a single literal
	}
	return nil
}

func (RC RLECodec) DecodeBlock(src []byte) ([]byte, error) {
	// decode src using run-length decoding
	if len(src) == 0 {
		return src, nil
	}
	var (
		srcIdx           = 0     // where you are in the source
		flagBit   uint8  = 7     // current bit index in the flag byte
		flagByte  byte           // current flag byte
		runLen    uint64 = 1     // current run length
		runBytes  []byte         // current bytes to be repeated
		outLength uint64 = 0     // first pass variable for allocating for decoding
		flush            = false // whether or not you are at the end
	)
	// first pass for allocating output length
	for srcIdx < len(src) {
		//decodeGetFlagAndRunLength(&flagByte, flagBit, &runLen, &srcIdx, src)
		decodeGetFlagAndRunLength(&flagByte, flagBit, &runLen, &srcIdx, src)
		outLength += runLen * uint64(RC.byteLength)
		srcIdx += RC.byteLength
		if srcIdx >= len(src) {
			break
		}
		if flagBit == 0 {
			flagBit = 7
		} else {
			flagBit--
		}
	}
	// make output and reset variable
	out := make([]byte, 0, outLength)
	srcIdx = 0
	flagBit = 7
	runLen = 1
	// second pass to actually decode
	for srcIdx < len(src) {
		decodeGetFlagAndRunLength(&flagByte, flagBit, &runLen, &srcIdx, src)
		runBytes = src[srcIdx:min((srcIdx+RC.byteLength), len(src))]
		for range runLen {
			out = append(out, runBytes...)
		}
		srcIdx += len(runBytes)
		if len(runBytes) < RC.byteLength || srcIdx >= len(src) {
			// flush if literal was not full length (RC.byteLength)
			flush = true
		}
		if flagBit == 0 || flush {
			if flush {
				break
			}
			flagBit = 7
		} else {
			flagBit--
		}
	}
	return out, nil
}

func (RC RLECodec) IsLossless() bool {
	return RC.lossless
}
