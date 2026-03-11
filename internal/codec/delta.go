package codec

type DELTACodec struct{}

func (DELTACodec) EncodeBlock(src []byte) ([]byte, error) {
	// encode source using a delta encoding
	if len(src) < 2 {
		return src, nil
	}
	var prev = int8(src[0])
	for i := 1; i < len(src); i++ {
		prev, src[i] = int8(src[i]), byte(int8(src[i])-prev)
	}
	return src, nil
}

func (DELTACodec) DecodeBlock(src []byte) ([]byte, error) {
	// decode source using a delta encoding
	if len(src) < 2 {
		return src, nil
	}
	var prev = int8(src[0])
	for i := 1; i < len(src); i++ {
		src[i] = byte(int8(src[i]) + prev)
		prev = int8(src[i])
	}
	return src, nil
}

func (DELTACodec) IsLossless() bool {
	return true
}
