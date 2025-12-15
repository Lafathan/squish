package frame

import (
	"bufio"
	"errors"
	"io"
)

type FrameWriter struct {
	Writer *bufio.Writer // io.writer for writing a stream
	Header Header        // header of the stream
}

func NewFrameWriter(r io.Writer, flags, codec, checksumMode uint8) *FrameWriter {
	// create a FrameWriter from an io writer stream
	bufWriter := bufio.NewWriter(r)
	// fill in the header pieces
	h := Header{
		Key:          Key,
		Flags:        flags,
		Codec:        codec,
		ChecksumMode: checksumMode,
	}
	return &FrameWriter{Writer: bufWriter, Header: h}
}

func (fw *FrameWriter) Ready() error {
	// write the bytes to the stream
	err := fw.Header.WriteHeader(fw.Writer)
	if err != nil {
		return err
	}
	// flush it
	err = fw.Writer.Flush()
	return err
}

func (fw *FrameWriter) WriteBlock(b Block, p []byte) error {
	// build block header
	err := b.WriteBlock(fw)
	if err != nil {
		return err
	}
	// check to see if the payload is the correct size
	if len(p) != int(b.CSize) {
		return errors.New("payload size does not match compressed size value")
	}
	// append the payload to the block
	_, err = fw.Writer.Write(p)
	if err != nil {
		return err
	}
	// flush it
	err = fw.Writer.Flush()
	return err
}
