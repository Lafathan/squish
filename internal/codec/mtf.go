package codec

type MTFCodec struct{}

func makeAlphabet() [256]byte {
	// create a symbol alphabet
	var sym [256]byte
	for i := range len(sym) {
		sym[i] = byte(i)
	}
	return sym
}

func (MTFCodec) EncodeBlock(src []byte) ([]byte, error) {
	// make the symbol alphabet
	symbols := makeAlphabet()
	// loop through the input
	for i, v := range src {
		// if the data value matches the symbol
		for j, s := range symbols {
			// replace it with the index in the alphabet
			if s == v {
				src[i] = byte(j)
				copy(symbols[1:], symbols[:j])
				symbols[0] = s
				break
			}
		}
	}
	return src, nil
}

func (MTFCodec) DecodeBlock(src []byte) ([]byte, error) {
	// make the symbol alphabet
	symbols := makeAlphabet()
	// loop through the input
	for i, v := range src {
		// if the data value matches the symbol
		for j, s := range symbols {
			// replace it with the value at that index
			if byte(j) == v {
				src[i] = s
				copy(symbols[1:], symbols[:j])
				symbols[0] = s
				break
			}
		}
	}
	return src, nil
}

func (MTFCodec) IsLossless() bool {
	return true
}
