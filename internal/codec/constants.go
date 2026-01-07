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

// codec string to codec ID map
var StringToCodecIDMap = map[string]uint8{
	"RAW":     RAW,
	"RLE":     RLE,
	"HUFFMAN": HUFFMAN,
}

// codec interface
type Codec interface {
	EncodeBlock(src []byte) (dst []byte, err error)
	DecodeBlock(src []byte) (dst []byte, err error)
	IsLossless() bool
}
