package frame

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

type FrameWriter struct {
	Writer io.Writer // io.writer for writing a stream
	Header Header    // header of the stream
}

func NewFrameWriter(w io.Writer, h Header) *FrameWriter {
	return &FrameWriter{Writer: w, Header: h}
}

func (fw *FrameWriter) Ready() error {
	// write the header bytes to the stream
	return WriteHeader(fw.Writer, fw.Header)
}

func (fw *FrameWriter) Close() error {
	// write an EOS block to the stream
	return fw.WriteBlock(Block{BlockType: EOS, CSize: 0}, nil)
}

func (fw *FrameWriter) WriteBlock(b Block, payload io.Reader) error {
	if payload == nil {
		if b.CSize > 0 {
			return errors.New("nil payload but compressed size is non-zero")
		}
		payload = bytes.NewReader(nil)
	}
	// build block header
	err := WriteBlock(fw, b)
	if err != nil {
		return fmt.Errorf("frame error when writing header: %w", err)
	}
	// check for zero length
	if b.CSize == 0 {
		return nil
	}
	// copy the payload to the writer
	n, err := io.CopyN(fw.Writer, payload, int64(b.CSize))
	if err != nil {
		return fmt.Errorf("error when copying payload to frame writer: %w", err)
	}
	// check to see if the payload is the correct size
	if n != int64(b.CSize) {
		return errors.New("payload size does not match compressed size value")
	}
	return err
}
