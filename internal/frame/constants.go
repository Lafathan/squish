package frame

const MagicKey = "SQSH"
const MaxBlockSize = uint64(1<<32 - 1)
const ChecksumSize = 4

// Block types
const (
	EOSCodec = iota
	DefaultCodec
	BlockCodec
)

// Header Flag constants
const (
	NoChecksum = iota
	UncompressedChecksum
	CompressedChecksum
)

// Header Codec constants
const (
	RAW = iota
	RLE
	HUFFMAN
	LZ77
)
