package codec

// ======================================================================================
// Move To Front
//
// This codec is a transform and not a compression technique. This transform works by
// replacing a value with the index of that value in an alphabet. After every replacement
// the alphabet is updated by moving the matched element to the front. Frequently
// repeated characters become small values.
//
// ex:  Input                                              Alphabet
//     [ m  m  i  i  s  s  i  i  s  s  i  i  p  p  i  i] - ijklmnopqrs
//     [ 4                                             ] - mijklnopqrs
//     [ 4  0                                          ] - mijklnopqrs
//     [ 4  0  1                                       ] - imjklnopqrs
//     [ 4  0  1  0                                    ] - imjklnopqrs
//     [ 4  0  1  0 10                                 ] - simjklnopqr
//     [ 4  0  1  0 10  0                              ] - simjklnopqr
//     [ 4  0  1  0 10  0  1                           ] - ismjklnopqr
//     [ 4  0  1  0 10  0  1  0                        ] - ismjklnopqr
//     [ 4  0  1  0 10  0  1  0  1                     ] - simjklnopqr
//     [ 4  0  1  0 10  0  1  0  1  0                  ] - simjklnopqr
//     [ 4  0  1  0 10  0  1  0  1  0  1               ] - ismjklnopqr
//     [ 4  0  1  0 10  0  1  0  1  0  1  0            ] - ismjklnopqr
//     [ 4  0  1  0 10  0  1  0  1  0  1  0  8         ] - pismjklnoqr
//     [ 4  0  1  0 10  0  1  0  1  0  1  0  8  0      ] - pismjklnoqr
//     [ 4  0  1  0 10  0  1  0  1  0  1  0  8  0  1   ] - ipsmjklnoqr
//     [ 4  0  1  0 10  0  1  0  1  0  1  0  8  0  1  0] - ipsmjklnoqr
// ======================================================================================

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
