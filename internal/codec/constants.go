package codec

// codec IDs
const (
	RAW = iota
	RLE
	RLE2
	RLE3
	RLE4
	LRLE
	LRLE2
	LRLE3
	LRLE4
	ZRLE
	HUFFMAN
	LZSS
	AUTO
	MTF
	BWT
)

// codec key map
var CodecMap = map[uint8]Codec{
	RAW:     RAWCodec{},
	RLE:     RLECodec{byteLength: 1, lossless: true},
	RLE2:    RLECodec{byteLength: 2, lossless: true},
	RLE3:    RLECodec{byteLength: 3, lossless: true},
	RLE4:    RLECodec{byteLength: 4, lossless: true},
	LRLE:    RLECodec{byteLength: 1, lossless: false},
	LRLE2:   RLECodec{byteLength: 2, lossless: false},
	LRLE3:   RLECodec{byteLength: 3, lossless: false},
	LRLE4:   RLECodec{byteLength: 4, lossless: false},
	ZRLE:    ZRLECodec{},
	HUFFMAN: HUFFMANCodec{},
	LZSS:    LZSSCodec{},
	AUTO:    &AUTOCodec{},
	MTF:     MTFCodec{},
	BWT:     BWTCodec{},
}

// codec string to codec ID map
var StringToCodecIDMap = map[string]uint8{
	"RAW":     RAW,
	"RLE":     RLE,
	"RLE2":    RLE2,
	"RLE3":    RLE3,
	"RLE4":    RLE4,
	"LRLE":    LRLE,
	"LRLE2":   LRLE2,
	"LRLE3":   LRLE3,
	"LRLE4":   LRLE4,
	"ZRLE":    ZRLE,
	"HUFFMAN": HUFFMAN,
	"LZSS":    LZSS,
	"AUTO":    AUTO,
	"MTF":     MTF,
	"BWT":     BWT,
}

// codec aliases
var CodecAliases = map[string]string{
	"DEFLATE": "LZSS-HUFFMAN",
}

// codec interface
type Codec interface {
	EncodeBlock(src []byte) (dst []byte, err error)
	DecodeBlock(src []byte) (dst []byte, err error)
	IsLossless() bool
}
