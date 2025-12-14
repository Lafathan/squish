package frame

import (
	"bufio"
	"io"
)

type FrameReader struct {
	Reader bufio.Reader // io.reader for reading a stream
	Header Header       // header of the stream
}

func NewFrameReader(r io.Reader, h Header) *FrameReader {
	// create a FrameReader from an io reader stream
	bufReader := bufio.NewReader(r)
	return &FrameReader{Reader: *bufReader, Header: h}
}
