package frame

const Key = "SQSH"
const MaxBlockSize = 1 << 32

// Header Flag constants
const (
	HasChecksum uint8 = 1 << iota
)

const (
	DefaultCodec uint8 = 1 << iota
	BlockCodec
)

const (
	UncompressedCheckSum uint8 = 1 << iota
	CompressedCheckSum
)

// Header Codec constants
const (
	RAW uint8 = 1 << iota
	RLE
	HUFFMAN
	LZ77
)
