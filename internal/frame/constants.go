package frame

const MagicKey = "SQSH"
const MaxBlockSize = uint32(1<<32 - 1)

// Header Flag constants
const (
	HasChecksum uint8 = 1 << iota
)

const (
	DefaultCodec = 0
	BlockCodec   = 1
)

const (
	UncompressedCheckSum = 1
	CompressedCheckSum   = 2
)

// Header Codec constants
const (
	RAW     = 0
	RLE     = 1
	HUFFMAN = 2
	LZ77    = 3
)
