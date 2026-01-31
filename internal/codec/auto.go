package codec

const (
	AutoDepth int = 2
	keepAlong int = 3
)

type AUTOCodec struct {
	CodecIDs []uint8
}

func (AC AUTOCodec) EncodeBlock(src []byte) ([]byte, error) {
	AC.CodecIDs = []uint8{LZSS, HUFFMAN}
	return src, nil
}

func (AUTOCodec) DecodeBlock(src []byte) ([]byte, error) {
	return src, nil
}

func (AUTOCodec) IsLossless() bool {
	return false
}
