package frame

import (
	"errors"
)

type Header struct {
	Key          string // "SQSH" Magic string marking the start of a header
	Flags        uint8  // flags to determine processing
	Codec        uint8  // default codec used
	ChecksumMode uint8  // stream checksum mode
}

func (h Header) valid() error {
	// make sure the header starts with the valid start key
	if h.Key != Key {
		return errors.New("invalid header start key")
	}
	return nil
}
