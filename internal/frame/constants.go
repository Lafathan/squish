package frame

const MagicKey = "SQSH"
const MaxBlockSize = uint64(1<<32 - 1)
const ChecksumSize = 4

// Header Flag constants
const (
	HasChecksum uint8 = 1 << iota
)

const (
	EOSCodec = iota
	DefaultCodec
	BlockCodec
)

const (
	NoCheckSum = iota
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
