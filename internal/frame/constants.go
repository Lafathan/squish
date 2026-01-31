package frame

const MagicKey = "SQZ"
const MaxBlockSize = 1<<24 - 1

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
