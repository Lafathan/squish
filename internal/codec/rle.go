package codec

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
	var (
		residual float32
		tol      float32
	)
	for i := range len(t.tolerance) {
		// calculate the new tolerance based on a residuals
		residual = float32(absByteDiff(t.anchor[i], data[i]))
		t.sigma[i] = (1-tolAlpha)*t.sigma[i] + tolAlpha*residual
		tol = tolMin + tolK*t.sigma[i]
		t.tolerance[i] = clampFloat(tol, tolMin, tolMax)
		// track and update candidate for new anchor values
		if residual <= t.tolerance[i] {
			// track repeats of valid candidates
			if absByteDiff(t.candidate[i], data[i]) <= tolBand {
				t.count[i]++
			} else {
				t.candidate[i] = data[i]
				t.count[i] = 1
			}
			// choose a new anchor if candidate repeats enough
			if t.count[i] >= int(tolHang) {
				t.anchor[i] = t.candidate[i]
				t.count[i] = 0
			}
		} else {
			// pick a new anchor if residual is way outside of the window
			t.anchor[i] = data[i]
			t.sigma[i] = 0
			t.candidate[i] = data[i]
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
		flagBit    uint8        = 7                                    // current bit representing a pair or not
		flagByte   byte         = 0x00                                 // byte holding flag bits
		runLen     uint32       = 1                                    // current length of the run
		runBytes   []byte       = nil                                  // current bytes being repeated
		groupBytes []byte       = make([]byte, 0, 8*(RC.byteLength+1)) // current set of encoded bytes
		srcIdx     int          = 0                                    // index as you traverse the source
		srcBytes   []byte       = nil                                  // current bytes from the source
		tol        *RLTolerance = newTolerance(RC.byteLength)          // noise and tolerance calculations
		out        []byte       = make([]byte, 0, len(src)*9/8)        // encoded bytes
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

// func decodeGetFlagAndRunLength(flagByte *byte, flagBit uint8, runLen *int, srcIdx *int, src []byte) error {
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
