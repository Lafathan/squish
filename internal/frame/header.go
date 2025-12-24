package frame

import (
	"errors"
	"fmt"
	"io"
)

type Header struct {
	Key          string // "SQSH" Magic string marking the start of a header
	Flags        uint8  // flags to determine processing
	Codec        uint8  // default codec used
	ChecksumMode uint8  // per block checksum mode
}

func (h *Header) Valid() error {
	// make sure the header starts with the valid start key
	if h.Key != MagicKey {
		return errors.New("invalid header start key")
	}
	if h.ChecksumMode > UncompressedChecksum+CompressedChecksum {
		return errors.New("invalid checksum method found")
	}
	return nil
}

func (h Header) String() string {
	s := fmt.Sprintf("Key:          %s\n", h.Key)
	s += fmt.Sprintf("Flags:        %04b\n", h.Flags)
	s += fmt.Sprintf("Codec:        %d\n", h.Codec)
	s += fmt.Sprintf("ChecksumMode: %04b\n", h.ChecksumMode)
	return s
}

func ReadHeader(r io.Reader) (Header, error) {
	var h Header
	// read in the header of the frame
	bytes := make([]byte, 7)
	_, err := io.ReadFull(r, bytes)
	if err != nil {
		return h, fmt.Errorf("error in reading header: %v", err)
	}

	// assign values to the header of the FrameReader
	h.Key = string(bytes[:4])
	h.Flags = bytes[4]
	h.Codec = bytes[5]
	h.ChecksumMode = bytes[6]

	return h, nil
}

func WriteHeader(w io.Writer, h Header) error {
	// build byte array for header
	bytes := []byte(h.Key)
	bytes = append(bytes, []byte{h.Flags, h.Codec, h.ChecksumMode}...)
	// write the header so the FrameWriter is ready to start writing blocks
	_, err := w.Write(bytes)
	if err != nil {
		return fmt.Errorf("error in writing header - %s: %v", h, err)
	}
	return nil
}
