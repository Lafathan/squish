package codec

// ======================================================================================
// eXclusive OR Transform
// This codec is a transform and not a comression technique. This technique stores the
// first byte from the data set, then the result of every subsequent byte XOR'd with the
// previous. This can reduce entropy and improve compressability for certain data sets.
//
// examples:
//  | Data                                | Transformed
//  | 00000000 01010101 00110000 11110010 | 00000000 01010101 01100101 11000010
//  | 0x42 0x42 0x42 0x42 0x42 0x42 0x43  | 0x42 0x00 0x00 0x00 0x00 0x00 0x01
//  | 0x00 0x01 0x02 0x03 0x04 0x05 0x06  | 0x00 0x01 0x03 0x01 0x07 0x01 0x03
//  | 0x02 0x03 0x02 0x03 0x02 0x03 0x02  | 0x02 0x01 0x01 0x01 0x01 0x01 0x01
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
