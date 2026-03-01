package frame

import (
	"bytes"
	"fmt"
	"io"
	"squish/internal/sqerr"
)

type Header struct {
	Key          string  // Magic string marking the start of a header
	Flags        uint8   // flags to determine processing
	Codec        []uint8 // default codec used
	ChecksumMode uint8   // per block checksum mode
}

func (h *Header) valid() error {
	// checks validity of a frame header
	// determined by valid header key and valid checksum mode byte
	if h.Key != MagicKey {
		return sqerr.New(sqerr.Corrupt, "invalid header start key found")
	}
	if h.ChecksumMode > UncompressedChecksum+CompressedChecksum {
		return sqerr.New(sqerr.Corrupt, "invalid checksum method found")
	}
	return nil
}

func (header1 Header) equal(header2 Header) bool {
	// checks if two headers are the same
	a := header1.Key == header2.Key
	b := header1.Flags == header2.Flags
	c := header1.ChecksumMode == header2.ChecksumMode
	d := bytes.Equal(header1.Codec, header2.Codec)
	return a && b && c && d
}

func (h Header) String() string {
	s := fmt.Sprintf("Key:          %s\n", h.Key)
	s += fmt.Sprintf("Flags:        %04b\n", h.Flags)
	s += fmt.Sprintf("Codec:        %d\n", h.Codec)
	s += fmt.Sprintf("ChecksumMode: %04b\n", h.ChecksumMode)
	return s
}

func readHeader(r io.Reader) (Header, error) {
	// read in the details of the header
	var (
		h     Header
		bytes = make([]byte, len(MagicKey)+3)
	)
	_, err := io.ReadFull(r, bytes)
	if err != nil {
		return h, fmt.Errorf("failed to read header: %w", err)
	}
	// build the header from the bytes
	h.Key = string(bytes[:len(MagicKey)])
	h.Flags = bytes[len(MagicKey)]
	h.ChecksumMode = bytes[len(MagicKey)+1]
	codecs := bytes[len(MagicKey)+2]
	// read in the codecs
	h.Codec = make([]byte, codecs)
	_, err = io.ReadFull(r, h.Codec)
	if err != nil {
		return h, fmt.Errorf("failed to read header codecs: %w", err)
	}
	return h, nil
}

func writeHeader(w io.Writer, h Header) error {
	// build byte array for header
	bytes := []byte(h.Key)
	bytes = append(bytes, h.Flags)
	bytes = append(bytes, h.ChecksumMode)
	bytes = append(bytes, byte(len(h.Codec)))
	bytes = append(bytes, h.Codec...)
	// write the header so FrameWriter is ready to write blocks
	_, err := w.Write(bytes)
	if err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}
	return nil
}
