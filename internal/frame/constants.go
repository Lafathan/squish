package frame

const MagicKey = "SQSH"
const MaxBlockSize = uint64(1<<16 - 1)

// Block types
const (
	EOS = iota
	DefaultCodec
	BlockCodec
)

// Header Flag constants
const (
	NoChecksum = iota
	UncompressedChecksum
	CompressedChecksum
)
