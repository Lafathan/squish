package codec

type XORCodec struct{}

func (XORCodec) EncodeBlock(src []byte) ([]byte, error) {
	// encode src using an XOR transform
	if len(src) < 2 {
		return src, nil
	}
	var prev = src[0]
	for i := 1; i < len(src); i++ {
		src[i], prev = src[i]^prev, src[i]
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
		src[i] = src[i] ^ prev
		prev = src[i]
	}
	return src, nil
}

func (XORCodec) IsLossless() bool {
	return true
}
