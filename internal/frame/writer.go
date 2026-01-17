package frame

import (
	"bytes"
	"fmt"
	"io"
	"squish/internal/sqerr"
)

type frameWriter struct {
	writer io.Writer // io.writer for writing a stream
	header Header    // header of the stream
}

func NewFrameWriter(w io.Writer, h Header) *frameWriter {
	return &frameWriter{writer: w, header: h}
}

func (fw *frameWriter) Ready() error {
	return writeHeader(fw.writer, fw.header) // write the header bytes to the stream
}

func (fw *frameWriter) Close() error {
	return fw.WriteBlock(Block{BlockType: EOS, CSize: 0}, nil) // write EOS block to stream
}

func (fw *frameWriter) WriteBlock(b Block, payload io.Reader) error {
	if payload == nil {
		if b.CSize > 0 {
			return sqerr.New(sqerr.Corrupt, "nil payload but compressed size is non-zero")
		}
		payload = bytes.NewReader(nil)
	}
	err := writeBlock(fw, b) // build block header
	if err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}
	if b.CSize == 0 { // check for zero length
		return nil
	}
	n, err := io.CopyN(fw.writer, payload, int64(b.CSize)) // copy the payload to the writer
	if err != nil {
		return fmt.Errorf("failed when copying payload to frame writer: %w", err)
	}
	if n != int64(b.CSize) { // check to see if the payload is the correct size
		return sqerr.New(sqerr.Corrupt, fmt.Sprintf("mismatched payload size: got %d - expected %d", n, b.CSize))
	}
	return err
}
