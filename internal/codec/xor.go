package codec

// ======================================================================================
// eXclusive OR Transform
//
// This codec is a transform and not a compression technique. This transform stores the
// first byte from the data set, then the result of every subsequent byte XOR'd with the
// previous. This can reduce entropy and improve compressability for certain data sets.
//
// ex: [42 42 42 42 42 42 43 43 43 43] -> [42 0 0 0 0 0 1 0 0 0]
//     [2 3 2 3 2 3 2 3 2 3 2 3 2 3 2] -> [ 2 1 1 1 1 1 1 1 1 1 1 1 1 1 1]
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
