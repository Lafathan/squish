package frame

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

type frameWriter struct {
	writer io.Writer // io.writer for writing a stream
	header Header    // header of the stream
}

func NewFrameWriter(w io.Writer, h Header) *frameWriter {
	return &frameWriter{writer: w, header: h}
}

func (fw *frameWriter) Ready() error {
	// write the header bytes to the stream
	return writeHeader(fw.writer, fw.header)
}

func (fw *frameWriter) Close() error {
	// write an EOS block to the stream
	return fw.WriteBlock(Block{BlockType: EOS, CSize: 0}, nil)
}

func (fw *frameWriter) WriteBlock(b Block, payload io.Reader) error {
	if payload == nil {
		if b.CSize > 0 {
			return errors.New("nil payload but compressed size is non-zero")
		}
		payload = bytes.NewReader(nil)
	}
	// build block header
	err := writeBlock(fw, b)
	if err != nil {
		return fmt.Errorf("frame error when writing header: %w", err)
	}
	// check for zero length
	if b.CSize == 0 {
		return nil
	}
	// copy the payload to the writer
	n, err := io.CopyN(fw.writer, payload, int64(b.CSize))
	if err != nil {
		return fmt.Errorf("error when copying payload to frame writer: %w", err)
	}
	// check to see if the payload is the correct size
	if n != int64(b.CSize) {
		return errors.New("payload size does not match compressed size value")
	}
	return err
}
