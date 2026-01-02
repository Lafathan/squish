package codec

type RAWCodec struct{}

func (RAWCodec) EncodeBlock(src []byte) ([]byte, int, error) {
	return src, 0, nil
}

func (RAWCodec) DecodeBlock(src []byte, padBits int) ([]byte, error) {
	return src, nil
}

func (RAWCodec) IsLossless() bool {
	return true
}
