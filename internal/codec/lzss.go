package codec

type LZSSCodec struct{}

func (LZSSCodec) EncodeBlock(src []byte) ([]byte, error) {
	return src, nil
}

func (LZSSCodec) DecodeBlock(src []byte) ([]byte, error) {
	return src, nil
}

func (LZSSCodec) IsLossless() bool {
	return true
}
