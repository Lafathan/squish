package frame

import (
	"bufio"
	"io"
)

type FrameReader struct {
	Reader *bufio.Reader // io.reader for reading a stream
	Header Header        // header of the stream
}

func NewFrameReader(r io.Reader) *FrameReader {
	// create a FrameReader from an io reader stream
	bufReader := bufio.NewReader(r)
	// create an empty header for now
	var h Header
	return &FrameReader{Reader: bufReader, Header: h}
}

func (fr *FrameReader) Ready() error {
	// read in the header of the frame
	header, err := ReadHeader(fr.Reader)
	if err != nil {
		return err
	}
	fr.Header = header

	// validity check
	headerError := fr.Header.Valid()

	return headerError
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
