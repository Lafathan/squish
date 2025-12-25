package codec

type RAWCodec struct{}

func (RAWCodec) EncodeBlock(src []byte) ([]byte, uint8, error) {
	return src, 0, nil
}

func (RAWCodec) DecodeBlock(src []byte, padBits uint8) ([]byte, error) {
	return src, nil
}

func (RAWCodec) IsLossless() bool {
	return true
}
