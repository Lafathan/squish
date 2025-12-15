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
	err := fr.Header.ReadHeader(fr.Reader)
	if err != nil {
		return err
	}

	// validity check
	headerError := fr.Header.valid()

	return headerError
}

func (fr *FrameReader) Next() (Block, io.Reader, error) {
	// read in the block header
	var b Block
	err := b.ReadBlock(fr)
	if err != nil {
		return b, nil, err
	}

	// validity check
	blockError := b.valid()
	if blockError != nil {
		return b, nil, blockError
	}

	// generate an io.reader for the payload
	payloadReader := io.LimitReader(fr.Reader, int64(b.CSize))

	return b, payloadReader, nil
}

func (fr *FrameReader) ReadBytes(n int) ([]byte, error) {
	// read n bytes from a FrameReader stream
	bytes := make([]byte, n)
	_, err := io.ReadFull(fr.Reader, bytes)
	return bytes, err
}
