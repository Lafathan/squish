package codec

type MTFCodec struct{}

func makeAlphabet() ([256]byte, [256]byte) {
	// make an alphabet along with a positional reference array
	var sym, pos [256]byte
	for i := range len(sym) {
		j := byte(i)
		sym[i] = j
		pos[j] = j
	}
	return sym, pos
}

func (MTFCodec) EncodeBlock(src []byte) ([]byte, error) {
	// encode src using the move-to-front transform
	sym, pos := makeAlphabet()
	// loop through the source performing the mtf transform
	for sIdx := range len(src) {
		// get the value and replace it with the index of where it is in the alphabet
		sSym := src[sIdx]
		aIdx := pos[sSym]
		src[sIdx] = aIdx
		// nothing else to do if it is already in the front
		if aIdx == 0 {
			continue
		}
		// otherwise shift all other symbols to the right along with their alphabet indexes
		for j := aIdx; j > 0; j-- {
			x := sym[j-1]
			sym[j] = x
			pos[x] = j
		}
		// put the current symbol up front
		sym[0] = sSym
		pos[sSym] = 0
	}
	return src, nil
}

func (MTFCodec) DecodeBlock(src []byte) ([]byte, error) {
	// decode src using the inverse move-to-front transform
	sym, pos := makeAlphabet()
	// loop through the source performing the inverse mtf transform
	for sIdx := range len(src) {
		// get the index and replace it with the value at that index in the alphabet
		aIdx := src[sIdx]
		sSym := sym[aIdx]
		src[sIdx] = sSym
		// nothing else to do if it is already in the front
		if aIdx == 0 {
			continue
		}
		// otherwise shift all other symbols to the right along with their alphabet indexes
		for j := aIdx; j > 0; j-- {
			x := sym[j-1]
			sym[j] = x
			pos[x] = j
		}
		// put the current symbol up front
		sym[0] = sSym
		pos[sSym] = 0
	}
	return src, nil
}

func (MTFCodec) IsLossless() bool {
	return true
}
