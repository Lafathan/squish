package frame

import (
	"errors"
	"io"
)

type Header struct {
	Key          string // "SQSH" Magic string marking the start of a header
	Flags        uint8  // flags to determine processing
	Codec        uint8  // default codec used
	ChecksumMode uint8  // stream checksum mode
}

func (h *Header) Valid() error {
	// make sure the header starts with the valid start key
	if h.Key != MagicKey {
		return errors.New("invalid header start key")
	}
	return nil
}

func ReadHeader(r io.Reader) (Header, error) {
	var h Header
	// read in the header of the frame
	bytes := make([]byte, 7)
	_, err := io.ReadFull(r, bytes)
	if err != nil {
		return h, err
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
		return err
	}
	return nil
}
