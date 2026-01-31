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
	var (
		runLen    uint8        = 1                           // current length of the run
		runBytes  []byte       = nil                         // current bytes being repeated
		srcIdx    int          = 0                           // index as you traverse the source
		srcBytes  []byte       = nil                         // current bytes from the source
		srcLen    int          = len(src)                    // length of input
		tolerance *RLTolerance = newTolerance(RC.byteLength) // noise and tolerance calculations
	)
	if srcLen == 0 {
		return []byte{}, nil
	}
	if srcLen < RC.byteLength {
		encBytes := []byte{1}               // return a run length of 1 of the original source
		encBytes = append(encBytes, src...) // if the source is shorter than the byte length
		return encBytes, nil
	}
	runLen = 1
	encBytes := make([]byte, 0, (srcLen/RC.byteLength+1)*(RC.byteLength+1))
	runBytes = src[:RC.byteLength]
	for srcIdx = RC.byteLength; srcIdx+RC.byteLength <= srcLen; srcIdx += RC.byteLength {
		srcBytes = src[srcIdx : srcIdx+RC.byteLength] // get next set of bytes from the source
		if !RC.lossless {
			tolerance.updateTolerance(srcBytes)
		}
		if equalSlice(runBytes, srcBytes, tolerance.tolerance) && runLen < maxRunLength {
			runLen++ // count them if they match the previous bytes
			continue
		}
		encBytes = append(encBytes, runLen)      // add the run length
		encBytes = append(encBytes, runBytes...) // add the run bytes
		runBytes = srcBytes                      // set the run bytes to the new bytes
		runLen = 1                               // reset the run length
	}
	encBytes = append(encBytes, runLen) // flush the final run
	encBytes = append(encBytes, runBytes...)
	if rem := srcLen % RC.byteLength; rem != 0 { // if there are leftover bytes (sub byteLength)
		encBytes = append(encBytes, 1)                   // add the run length
		encBytes = append(encBytes, src[srcLen-rem:]...) // add the run bytes
	}
	return encBytes, nil
}

func (RC RLECodec) DecodeBlock(src []byte) ([]byte, error) {
	var (
		outLen int = 0        // length of the output
		outIdx int = 0        // how many bytes have been decoded
		srcIdx int = 0        // index as you traverse the source
		srcLen int = len(src) // how long is the input
		runLen int = 0        // how long is the current run
	)
	if srcLen == 0 {
		return []byte{}, nil
	}
	if srcLen <= RC.byteLength {
		return src[1:], nil
	}
	for srcIdx < srcLen { // while you are not at the end of the source
		runLen = int(src[srcIdx]) // count how many bytes will be added
		srcIdx++                  // jump to the actual bytes to repeat
		if rem := srcLen - srcIdx; RC.byteLength > rem {
			outLen += rem // if a short chunk is remaining, add it to the output length
			break
		}
		outLen += runLen * RC.byteLength // increase the output
		srcIdx += RC.byteLength          // jump to the next run length value
	}
	decBytes := make([]byte, outLen) // make the array you need for output
	srcIdx = 0                       // keep track of where you are in the input
	for srcIdx < srcLen {
		runLen = int(src[srcIdx]) // count how many bytes will be added
		srcIdx++                  // jump to the actual bytes to repeat
		if rem := srcLen - srcIdx; RC.byteLength > rem {
			for i := range len(src) - srcIdx {
				decBytes[outIdx] = src[srcIdx+i] // if a short chunk is remaining, add the bytes to the output
				outIdx++
			}
			break
		}
		for range runLen { // for every repetition
			for i := range srcIdx + RC.byteLength - srcIdx {
				decBytes[outIdx] = src[srcIdx+i] // loop through and add the bytes repeated
				outIdx++
			}
		}
		srcIdx += RC.byteLength // jump to the next run-bytes pair
	}
	return decBytes, nil
}

func (RC RLECodec) IsLossless() bool {
	return RC.lossless
}
