package frame

type Block struct {
	BlockType      uint8  // 0x00 EOS, 0x01 Default codec, 0x02 Block codec
	Codec          uint8  // only present if BlockType == 0x02
	USize          uint64 // uncompressed size
	CSize          uint64 // compressed size
	ChecksumMethod uint8  // Checksum - uncompressed (0x01), compressed (0x02), both, or none
	Checksum       uint8  // checksum value 4 bytes for uncompressed, 4 bytes for compressed
}
