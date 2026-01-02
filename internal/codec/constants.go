package codec

// codec IDs
const (
	RAW = iota
	RLE
	HUFFMAN
	LZ77
	DCT
)

// codec key map
var CodecMap = map[uint8]Codec{
	RAW:     RAWCodec{},
	RLE:     RLECodec{},
	HUFFMAN: HUFFMANCodec{},
}

// codec interface
type Codec interface {
	EncodeBlock(src []byte) (dst []byte, padBits int, err error)
	DecodeBlock(src []byte, padBits int) (dst []byte, err error)
	IsLossless() bool
}
