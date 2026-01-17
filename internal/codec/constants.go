package codec

// codec IDs
const (
	RAW = iota
	RLE
	RLE2
	RLE3
	RLE4
	HUFFMAN
	LZ77
	DCT
)

// codec key map
var CodecMap = map[uint8]Codec{
	RAW:     RAWCodec{},
	RLE:     RLENCodec{byteLength: 1},
	RLE2:    RLENCodec{byteLength: 2},
	RLE3:    RLENCodec{byteLength: 3},
	RLE4:    RLENCodec{byteLength: 4},
	HUFFMAN: HUFFMANCodec{},
}

// codec string to codec ID map
var StringToCodecIDMap = map[string]uint8{
	"RAW":     RAW,
	"RLE":     RLE,
	"RLE2":    RLE2,
	"RLE3":    RLE3,
	"RLE4":    RLE4,
	"HUFFMAN": HUFFMAN,
}

// codec interface
type Codec interface {
	EncodeBlock(src []byte) (dst []byte, err error)
	DecodeBlock(src []byte) (dst []byte, err error)
	IsLossless() bool
}
