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
	RAW: RAWCodec{},
}

// codec interface
type Codec interface {
	EncodeBlock(src *[]byte) (dst []byte, padBits uint8, err error)
	DecodeBlock(src *[]byte, padBits uint8) (dst []byte, err error)
}
