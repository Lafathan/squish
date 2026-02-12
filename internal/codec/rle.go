package codec

const (
	maxRunLength uint8   = 127
	tolAlpha     float64 = 0.15 // tolerance sigma decay
	tolMin       float64 = 2.0  // residual that will always result in conforming to anchor
	tolMax       float64 = 6.0  // residual that will always result in a new anchor
	tolK         float64 = 1.5  // variance to tolerance factor
	tolBand      uint8   = 1    // wiggle allowance when considering new anchor candidate
	tolHang      uint8   = 3    // required repetitions for candidate to become new anchor
)

type RLECodec struct {
	byteLength int
	lossless   bool
}

type RLTolerance struct {
	anchor    []byte
	sigma     []float64
	tolerance []float64
	candidate []byte
	count     []int
}

func equalSlice(slice1 []byte, slice2 []byte, tol []float64) bool {
	// element-wise slice comparison
	if len(slice1) != len(slice2) {
		return false
	}
	for i := range len(slice1) {
		if slice1[i] > slice2[i] {
			if float64(slice1[i]-slice2[i]) > tol[i] {
				return false
			}
		} else {
			if float64(slice2[i]-slice1[i]) > tol[i] {
				return false
			}
		}
	}
	return true
}

func absByteDiff(a, b byte) byte {
	if a >= b {
		return a - b
	}
	return b - a
}

func clampFloat(f, lo, hi float64) float64 {
	f = max(f, lo)
	return min(f, hi)
}

func newTolerance(n int) *RLTolerance {
	return &RLTolerance{
		anchor:    make([]byte, n),
		sigma:     make([]float64, n),
		tolerance: make([]float64, n),
		candidate: make([]byte, n),
		count:     make([]int, n),
	}
}

func (t *RLTolerance) updateTolerance(data []byte) {
	var (
		res byte
		tol float64
	)
	for i := range len(t.tolerance) { // loop through the bytes
		res = absByteDiff(t.anchor[i], data[i])                      // get a residual of new data
		t.sigma[i] = (1-tolAlpha)*t.sigma[i] + tolAlpha*float64(res) // calculate sigma
		tol = tolMin + tolK*t.sigma[i]                               // calculate the new tolerance
		t.tolerance[i] = clampFloat(tol, tolMin, tolMax)             // clamp it
		if float64(res) <= t.tolerance[i] {
			if absByteDiff(t.candidate[i], data[i]) <= tolBand { // if candidate residual is in valid band
				t.count[i]++ // keep track of repeats of new candidate anchor
			} else {
				t.candidate[i] = data[i] // new candidate observed
				t.count[i] = 1           // reset the count
			}
			if t.count[i] >= int(tolHang) { // if there are enough of the candidate anchor
				t.anchor[i] = t.candidate[i] // replace the anchor
				t.count[i] = 0
			}
		} else {
			t.anchor[i] = data[i] // if the residual is way outside the window
			t.sigma[i] = 0        // pick the new anchor and reset everything
			t.candidate[i] = data[i]
			t.count[i] = 0
			t.tolerance[i] = tolMin
		}
	}
}

func encodeUpdateGroup(runLen uint8, flagByte *byte, flagBit uint8, runBytes []byte, groupBytes *[]byte) {
	if runLen >= 2 { // only replace when you have a run (minumum of 2)
		*flagByte |= (1 << flagBit)                    // set the flag bit based on run length
		*groupBytes = append(*groupBytes, runLen)      // write the run length if it is not a literal
		*groupBytes = append(*groupBytes, runBytes...) // write the literal (runs and single literals)
	} else {
		*groupBytes = append(*groupBytes, runBytes...) // write the literal (runs and single literals)
	}
}

func (RC RLECodec) EncodeBlock(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return src, nil
	}
	var (
		flagBit    uint8        = 7                                    // current bit representing a pair or not
		flagByte   byte         = 0x00                                 // byte holding flag bits
		runLen     uint8        = 1                                    // current length of the run
		runBytes   []byte       = nil                                  // current bytes being repeated
		groupBytes []byte       = make([]byte, 0, 8*(RC.byteLength+1)) // current set of encoded bytes
		srcIdx     int          = 0                                    // index as you traverse the source
		srcBytes   []byte       = nil                                  // current bytes from the source
		tolerance  *RLTolerance = newTolerance(RC.byteLength)          // noise and tolerance calculations
		outBytes   []byte       = make([]byte, 0, len(src)*9/8)        // encoded bytes
	)
	for srcIdx < len(src) {
		srcBytes = src[srcIdx:min(srcIdx+RC.byteLength, len(src))]
		if !RC.IsLossless() && len(srcBytes) == RC.byteLength {
			tolerance.updateTolerance(srcBytes)
		}
		srcIdx += len(srcBytes)
		if runBytes == nil { // if no run has been set yet
			runBytes = srcBytes // set the run to the current source bytes
			runLen = 1          // you have a run of length 1
		} else if equalSlice(runBytes, srcBytes, tolerance.tolerance) && runLen < maxRunLength {
			runLen++ // if the next source bytes are "equal" to the run bytes, count it
		} else { // if they are not equal
			encodeUpdateGroup(runLen, &flagByte, flagBit, runBytes, &groupBytes)
			if flagBit == 0 { // if you are at the end of your flag
				outBytes = append(outBytes, flagByte) // write the current group
				outBytes = append(outBytes, groupBytes...)
				groupBytes = groupBytes[:0] // reset for the next group
				flagBit = 7                 // reset the flag bit and byte
				flagByte = 0x00
			} else {
				flagBit--
			}
			runBytes = srcBytes // reset what you are matching against
			runLen = 1
		}
		if srcIdx >= len(src) { // trailing match, write ramaining length + literals
			encodeUpdateGroup(runLen, &flagByte, flagBit, runBytes, &groupBytes)
			outBytes = append(outBytes, flagByte)
			outBytes = append(outBytes, groupBytes...)
		}
	}
	return outBytes, nil
}

func decodeGetFlagAndRunLength(flagByte *byte, flagBit uint8, runLen *int, srcIdx *int, src []byte) {
	if flagBit == 7 { // if you just reset the flag bit
		*flagByte = src[*srcIdx] // get a new flag byte
		*srcIdx++                // move forward
	}
	if *flagByte&(1<<flagBit) > 0 { // if you come across a run
		*runLen = int(src[*srcIdx]) // grab the run length
		*srcIdx++
	} else {
		*runLen = 1 // otherwise it is just a single literal
	}
}

func (RC RLECodec) DecodeBlock(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return src, nil
	}
	var (
		srcIdx            = 0 // where you are in the source
		flagBit   uint8   = 7 // current bit index in the flag byte
		flagByte  byte        // current flag byte
		runLen    = 1         // current run length
		runBytes  []byte      // current bytes to be repeated
		outLength = 0         // first pass variable for allocating for decoding
		flush     = false     // whether or not you are at the end
	)
	for srcIdx < len(src) {
		decodeGetFlagAndRunLength(&flagByte, flagBit, &runLen, &srcIdx, src)
		outLength += runLen * RC.byteLength
		srcIdx += len(runBytes) // increment past the literal
		if srcIdx >= len(src) {
			break
		}
		if flagBit == 0 {
			flagBit = 7
		} else {
			flagBit--
		}
	}
	outBytes := make([]byte, 0, outLength)
	srcIdx = 0
	flagBit = 7
	runLen = 1
	for srcIdx < len(src) {
		decodeGetFlagAndRunLength(&flagByte, flagBit, &runLen, &srcIdx, src)
		runBytes = src[srcIdx:min((srcIdx+RC.byteLength), len(src))] // get the bytes repeated
		for range runLen {
			outBytes = append(outBytes, runBytes...)
		}
		srcIdx += len(runBytes) // increment past the literal
		if len(runBytes) < RC.byteLength || srcIdx >= len(src) {
			flush = true // flush if literal was not full length (RC.byteLength)
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
	return outBytes, nil
}

func (RC RLECodec) IsLossless() bool {
	return RC.lossless
}
