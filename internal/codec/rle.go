package codec

const (
	maxRunLength uint8   = 255
	minRunLength uint8   = 2
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

func (RC RLECodec) EncodeBlock(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return src, nil
	}
	var (
		flagBit   uint8        = 7                                    // current bit representing a pair or not
		flagByte  byte         = 0x00                                 // byte holding flag bits
		runLen    uint8        = 1                                    // current length of the run
		runBytes  []byte       = nil                                  // current bytes being repeated
		outBytes  []byte       = make([]byte, 0, 8*(RC.byteLength+1)) // current set of bytes to be encoded
		srcIdx    int          = 0                                    // index as you traverse the source
		srcBytes  []byte       = nil                                  // current bytes from the source
		tolerance *RLTolerance = newTolerance(RC.byteLength)          // noise and tolerance calculations
	)
	if len(src) < RC.byteLength {
		flagByte = 0
		encBytes := []byte{flagByte}        // return a run length of 1 of the original source
		encBytes = append(encBytes, src...) // if the source is shorter than the byte length
		return encBytes, nil
	}
	encBytes := make([]byte, 0, len(src)*9/8)
	runBytes = src[:RC.byteLength]
	for srcIdx = RC.byteLength; srcIdx <= len(src); srcIdx += RC.byteLength {
		srcBytes = src[srcIdx:min(srcIdx+RC.byteLength, len(src))] // get next set of bytes from the source
		if len(srcBytes) < RC.byteLength {                         // if you have some trailing data
			if runLen > 1 {
				flagByte |= 0x01 << flagBit         // set the bit as representing a pair
				outBytes = append(outBytes, runLen) // save the run length
			} else {
				flagByte &= 0xFF - (0x01 << flagBit) // set the bit as representing a literal
			}
			outBytes = append(outBytes, runBytes...)                   // save the byte literals
			if flagBit == 0 || srcIdx+RC.byteLength >= len(srcBytes) { // if you are at the end of a byte block
				if len(outBytes) == 0 {
					break
				}
				encBytes = append(encBytes, flagByte)    // write the flagbit
				encBytes = append(encBytes, outBytes...) // write the encoded bytes
				outBytes = outBytes[:0]                  // reset the output bytes
				flagBit = 7                              // reset the flag bit
				flagByte = 0x00                          // reset the flag byte
			} else {
				flagBit--
			}
		}
		if !RC.lossless {
			tolerance.updateTolerance(srcBytes)
		}
		if equalSlice(runBytes, srcBytes, tolerance.tolerance) && runLen < maxRunLength {
			runLen++ // count them if they match the previous bytes
			continue
		}
		if runLen > 1 {
			flagByte |= 0x01 << flagBit         // set the bit as representing a pair
			outBytes = append(outBytes, runLen) // save the run length
		} else {
			flagByte &= 0xFF - (0x01 << flagBit) // set the bit as representing a literal
		}
		outBytes = append(outBytes, runBytes...) // save the byte literals
		if flagBit == 0 {                        // if you are at the end of a byte block
			encBytes = append(encBytes, flagByte)    // write the flagbit
			encBytes = append(encBytes, outBytes...) // write the encoded bytes
			outBytes = outBytes[:0]                  // reset the output bytes
			flagBit = 7                              // reset the flag bit
			flagByte = 0x00                          // reset the flag byte
		} else {
			flagBit--
		}
		runBytes = srcBytes // set the run bytes to the new bytes
		runLen = 1          // reset the run length
	}
	return encBytes, nil
}

func (RC RLECodec) DecodeBlock(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return src, nil
	}
	var (
		srcIdx     = 0     // where you are in the source
		flagBit    = 7     // current bit index in the flag byte
		flagByte   byte    // current flag byte
		groupBytes []byte  // current decoded bytes associated with the current flag
		runLen     = 1     // current run length
		runBytes   []byte  // current bytes to be repeated
		outLength  = 0     // first pass variable for allocating for decoding
		flush      = false // whether or not you are at the end
	)
	for srcIdx < len(src) {
		if flagBit == 7 { // if you just reset the flag bit
			flagByte = src[srcIdx] // get a new flag byte
			srcIdx++               // move forward
		}
		if flagByte&(1<<flagBit) > 0 { // if you come across a run
			runLen = int(src[srcIdx]) // grab the run length
			srcIdx++
		} else {
			runLen = 1 // otherwise it is just a single literal
		}
		runBytes = src[srcIdx:min((srcIdx+RC.byteLength), len(src))] // get the bytes repeated
		outLength += runLen * len(runBytes)
		srcIdx += len(runBytes) // increment past the literal
		if len(runBytes) < RC.byteLength || srcIdx >= len(src) {
			flush = true // flush if literal was not full length (RC.byteLength)
		}
		if flagBit == 0 || flush {
			outLength += 1
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
	flush = false
	for srcIdx < len(src) {
		if flagBit == 7 { // if you just reset the flag bit
			flagByte = src[srcIdx] // get a new flag byte
			srcIdx++               // move forward
		}
		if flagByte&(1<<flagBit) > 0 { // if you come across a run
			runLen = int(src[srcIdx]) // grab the run length
			srcIdx++
		} else {
			runLen = 1 // otherwise it is just a single literal
		}
		runBytes = src[srcIdx:min((srcIdx+RC.byteLength), len(src))] // get the bytes repeated
		for range runLen {
			groupBytes = append(groupBytes, runBytes...)
		}
		srcIdx += len(runBytes) // increment past the literal
		if len(runBytes) < RC.byteLength || srcIdx >= len(src) {
			flush = true // flush if literal was not full length (RC.byteLength)
		}
		if flagBit == 0 || flush {
			outBytes = append(outBytes, flagByte)
			outBytes = append(outBytes, groupBytes...)
			if flush {
				break
			}
			flagBit = 7
			groupBytes = groupBytes[:0]
		} else {
			flagBit--
		}
	}
	return outBytes, nil
}

func (RC RLECodec) IsLossless() bool {
	return RC.lossless
}
