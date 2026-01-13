package frame

import (
	"fmt"
	"io"
)

type Header struct {
	Key          string  // "SQSH" Magic string marking the start of a header
	Flags        uint8   // flags to determine processing
	Codec        []uint8 // default codec used
	ChecksumMode uint8   // per block checksum mode
}

func (h *Header) valid() error {
	if h.Key != MagicKey { // make sure the header starts with the valid start key
		return fmt.Errorf("invalid header start key")
	}
	if h.ChecksumMode > UncompressedChecksum+CompressedChecksum {
		return fmt.Errorf("invalid checksum method found")
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

func (header1 Header) equal(header2 Header) bool {
	a := header1.Key == header2.Key
	b := header1.Flags == header2.Flags
	c := header1.ChecksumMode == header2.ChecksumMode
	d := true
	for i := range header1.Codec {
		d = header1.Codec[i] == header2.Codec[i]
		if !d {
			return false
		}
	}
	return a && b && c && d
}

func readHeader(r io.Reader) (Header, error) {
	var h Header
	bytes := make([]byte, 7) // read in the header of the frame
	_, err := io.ReadFull(r, bytes)
	if err != nil {
		return h, fmt.Errorf("error in reading header: %w", err)
	}
	h.Key = string(bytes[:4]) // assign values to the header of the FrameReader
	h.Flags = bytes[4]
	h.ChecksumMode = bytes[5]
	codecs := bytes[6]
	h.Codec = make([]byte, codecs)
	_, err = io.ReadFull(r, h.Codec)
	if err != nil {
		return h, fmt.Errorf("error in reading codecs from header: %w", err)
	}
	return h, nil
}

func writeHeader(w io.Writer, h Header) error {
	bytes := []byte(h.Key) // build byte array for header
	bytes = append(bytes, h.Flags)
	bytes = append(bytes, h.ChecksumMode)
	bytes = append(bytes, byte(len(h.Codec)))
	bytes = append(bytes, h.Codec...)
	_, err := w.Write(bytes) // write the header so FrameWriter is ready to write blocks
	if err != nil {
		return fmt.Errorf("error in writing header - %s: %w", h, err)
	}
	return nil
}
