package frame

import (
	"bufio"
	"io"
)

type FrameWriter struct {
	Writer bufio.Writer // io.writer for writing a stream
	Header Header       // header of the stream
}

func NewFrameWriter(r io.Writer, h Header) *FrameWriter {
	// create a FrameWriter from an io writer stream
	bufWriter := bufio.NewWriter(r)
	return &FrameWriter{Writer: *bufWriter, Header: h}
}
