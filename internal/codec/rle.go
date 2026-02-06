package codec

const (
	maxRunLength uint8   = 255
	tolDecay     float64 = 1.0 / 32.0
	tolAttack    float64 = 2.0
	tolMin       float64 = 0.0
	tolMax       float64 = 24.0
	tolBase      float64 = 1.0
)

type RLECodec struct {
	byteLength int
	lossless   bool
}

type RLTolerance struct {
	tolerance []float64
	noise     []float64
	prevBytes []byte
}

func equalSlice(slice1 []byte, slice2 []byte, tol []float64) bool {
	// element-wise slice comparison
	if len(slice1) != len(slice2) {
		return false
	}
	for i, e := range slice1 {
		if slice2[i] > e {
			if float64(slice2[i]-e) > tol[i] {
				return false
			}
		} else {
			if float64(e-slice2[i]) > tol[i] {
				return false
			}
		}
	}
	return true
}

func newTolerance(length int) *RLTolerance {
	return &RLTolerance{
		tolerance: make([]float64, length),
		noise:     make([]float64, length),
		prevBytes: make([]byte, length),
	}
}

func (t *RLTolerance) updateTolerance(a []byte) {
	// byte by byte tolerance updater
	for i := range len(t.tolerance) { // loop through the bytes
		break
		delta := uint8(0) // get the delta from the previous byte (measure noise/jitter)
		if a[i] > t.prevBytes[i] {
			delta = a[i] - t.prevBytes[i]
		} else {
			delta = t.prevBytes[i] - a[i]
		}
		t.noise[i] += tolDecay * (float64(delta) - t.noise[i]) // get the new noise value
		t.tolerance[i] = tolBase + tolAttack*t.noise[i]        // calculate the new tolerances
		t.tolerance[i] = max(t.tolerance[i], tolMin)           // clamp them on the low side
		t.tolerance[i] = min(t.tolerance[i], tolMax)           // clamp them on the high side
		t.prevBytes = a
	}
}

func encodeUpdateGroup(runLen uint8, flagByte *byte, flagBit uint8, runBytes []byte, groupBytes *[]byte) {
	if runLen >= 2 { // only replace when you have a run (minumum of 2)
		*flagByte |= (1 << flagBit)               // set the flag bit based on run length
		*groupBytes = append(*groupBytes, runLen) // write the run length if it is not a literal
	} else {
		for range runLen - 1 {
			*groupBytes = append(*groupBytes, runBytes...) // write the literal (runs and single literals)
		}
	}
	*groupBytes = append(*groupBytes, runBytes...) // write the literal (runs and single literals)
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
		if !RC.IsLossless() {
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
		if srcIdx >= len(src) { // if you are at a trailing match, emit to write the ramaining length + literals
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
		runBytes = src[srcIdx:min((srcIdx+RC.byteLength), len(src))] // get the bytes repeated
		outLength += runLen * len(runBytes)
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
	outBytes := make([]byte, 0, outLength)
	srcIdx = 0
	flagBit = 7
	flagByte = 0x00
	flush = false
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
