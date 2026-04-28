package codec

// ======================================================================================
// eXclusive OR Transform
// This codec is a transform and not a comression technique. This technique stores the
// first byte from the data set, then the result of every subsequent byte XOR'd with the
// previous. This can reduce entropy and improve compressability for certain data sets.
//
// examples:
//  | Data                                | Transformed
//  | 42 42 42 42 42 42 43 43 43 43 43 43 | 42 0 0 0 0 0 1 0 0 0 0 0
//  | 0 1 2 3 4 5 6 7 8 9 10 11 12 13 14  | 0 1 3 1 7 1 2 1 15 7 3 1 7 1 3
//  | 2 3 2 3 2 3 2 3 2 3 2 3 2 3 2 3 2 3 | 2 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1
// ======================================================================================

type XORCodec struct{}

func (XORCodec) EncodeBlock(src []byte) ([]byte, error) {
	// encode src using an XOR transform
	if len(src) < 2 {
		return src, nil
	}
	var prev = src[0]
	for i := 1; i < len(src); i++ {
		cur := src[i]
		src[i], prev = cur^prev, cur
	}
	return src, nil
}

func (XORCodec) DecodeBlock(src []byte) ([]byte, error) {
	// decode src using an XOR transform
	if len(src) < 2 {
		return src, nil
	}
	var prev = src[0]
	for i := 1; i < len(src); i++ {
		prev = src[i] ^ prev
		src[i] = prev
	}
	return src, nil
}

func (XORCodec) IsLossless() bool {
	return true
}
