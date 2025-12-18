package frame

const MagicKey = "SQSH"
const MaxBlockSize = uint64(1<<32 - 1)
const ChecksumSize = 4

// Header Flag constants
const (
	HasChecksum uint8 = 1 << iota
)

const (
	EOSCodec     = 0
	DefaultCodec = 1
	BlockCodec   = 2
)

const (
	NoCheckSum           = 0
	UncompressedChecksum = 1
	CompressedChecksum   = 2
)

// Header Codec constants
const (
	RAW     = 0
	RLE     = 1
	HUFFMAN = 2
	LZ77    = 3
)
