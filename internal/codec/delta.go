package codec

// ======================================================================================
// Delta Transform
//
// This codec is a transform and not a compression technique. This transform stores the
// first byte from the data set, then the result of every subsequent byte differenced
// with the previous. This can improve compressability for certain data sets.
//
// ex: [42 42 42 42 42 43 43 43 43 42 42 42 42] -> [42 0 0 0 0 0 1 0 0 0 129 0 0 0]
// ======================================================================================

type DELTACodec struct{}

func (DELTACodec) EncodeBlock(src []byte) ([]byte, error) {
	// encode source using a delta encoding
	if len(src) < 2 {
		return src, nil
	}
	var prev = int8(src[0])
	for i := 1; i < len(src); i++ {
		s := int8(src[i])
		prev, src[i] = s, byte(s-prev)
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
		prev += int8(src[i])
		src[i] = byte(prev)
	}
	return src, nil
}

func (DELTACodec) IsLossless() bool {
	return true
}
