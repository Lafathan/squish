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
	// write the header bytes to the stream
	return writeHeader(fw.writer, fw.header)
}

func (fw *frameWriter) Close() error {
	// write EOS block to stream
	return fw.WriteBlock(Block{BlockType: EOS, CSize: 0}, nil)
}

func (fw *frameWriter) WriteBlock(b Block, payload io.Reader) error {
	// write a block to the fram
	if payload == nil {
		if b.CSize > 0 {
			return sqerr.New(sqerr.Corrupt, "nil payload but compressed size is non-zero")
		}
		payload = bytes.NewReader(nil)
	}
	// build block header and check for zero length
	if err := writeBlock(fw, b); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}
	if b.CSize == 0 {
		return nil
	}
	// copy the payload to the writer and verify written size
	n, err := io.CopyN(fw.writer, payload, int64(b.CSize))
	if err != nil {
		return fmt.Errorf("failed when copying payload to frame writer: %w", err)
	}
	if n != int64(b.CSize) {
		return sqerr.New(sqerr.Corrupt, fmt.Sprintf("mismatched payload size: got %d - expected %d", n, b.CSize))
	}
	return err
}
