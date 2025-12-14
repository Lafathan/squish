package frame

type Header struct {
	Key          string // "SQSH" Magic string marking the start of a header
	Flags        uint8  // flags to determine processing
	Codec        uint8  // default codec used
	ChecksumMode uint8  // stream checksum mode
}
