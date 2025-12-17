package frame

import (
	"errors"
	"io"
)

type FrameWriter struct {
	Writer io.Writer // io.writer for writing a stream
	Header Header    // header of the stream
}

func NewFrameWriter(w io.Writer, h Header) *FrameWriter {
	// create a FrameWriter from an io writer stream
	return &FrameWriter{Writer: w, Header: h}
}

func (fw *FrameWriter) Ready() error {
	// write the bytes to the stream
	return WriteHeader(fw.Writer, fw.Header)
}

func (fw *FrameWriter) WriteBlock(b Block, p []byte) error {
	// check to see if the payload is the correct size
	if len(p) != int(b.CSize) {
		return errors.New("payload size does not match compressed size value")
	}
	// build block header
	err := WriteBlock(fw, b)
	if err != nil {
		return err
	}
	// append the payload to the block
	_, err = fw.Writer.Write(p)
	return err
}
