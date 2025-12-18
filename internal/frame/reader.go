package frame

import (
	"io"
)

type FrameReader struct {
	Reader io.Reader // io.reader for reading a stream
	Header Header    // header of the stream
}

func NewFrameReader(r io.Reader) *FrameReader {
	// create an empty header for now
	return &FrameReader{Reader: r}
}

func (fr *FrameReader) Ready() error {
	// read in the header of the frame
	header, err := ReadHeader(fr.Reader)
	if err != nil {
		return err
	}
	fr.Header = header
	return fr.Header.Valid()
}

func (fr *FrameReader) Next() (Block, io.Reader, error) {
	// read in the block header
	block, err := ReadBlock(fr)
	if err != nil {
		return block, nil, err
	}
	// validity check
	blockError := block.Valid()
	if blockError != nil {
		return block, nil, blockError
	}
	// generate an io.reader for the payload
	payloadReader := io.LimitReader(fr.Reader, int64(block.CSize))

	return block, payloadReader, nil
}

func (fr *FrameReader) ReadBytes(n int) ([]byte, error) {
	// read n bytes from a FrameReader stream
	bytes := make([]byte, n)
	_, err := io.ReadFull(fr.Reader, bytes)
	return bytes, err
}

func (fr *FrameReader) ReadByte() (byte, error) {
	// read single byte
	bytes, err := fr.ReadBytes(1)
	return bytes[0], err
}
