package frame

import (
	"bufio"
	"encoding/binary"
	"io"
)

type FrameWriter struct {
	Writer bufio.Writer // io.writer for writing a stream
	Header Header       // header of the stream
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
	return &FrameWriter{Writer: *bufWriter, Header: h}
}

func (fw *FrameWriter) Ready() error {
	// build byte array for header
	bytes := []byte(fw.Header.Key)
	bytes = append(bytes, []byte{fw.Header.Flags, fw.Header.Codec, fw.Header.ChecksumMode}...)
	// write the header so the FrameWriter is ready to start writing blocks
	_, err := fw.Writer.Write(bytes)
	if err != nil {
		return err
	}
	return nil
}

func (fw *FrameWriter) WriteBlock(b Block, p []byte) error {
	// if EOS block is being written
	if b.BlockType == 0 {
		fw.Writer.WriteByte(b.BlockType)
		err := fw.Writer.Flush()
		if err != nil {
			return err
		}
		return nil
	}
	// build block header
	bytes := []byte{}
	bytes = append(bytes, b.BlockType, b.Codec)
	bytes = binary.BigEndian.AppendUint64(bytes, b.USize)
	bytes = binary.BigEndian.AppendUint64(bytes, b.CSize)
	bytes = append(bytes, b.ChecksumMethod)
	bytes = binary.BigEndian.AppendUint64(bytes, b.Checksum)
	// append the payload to the header
	bytes = append(bytes, p...)

	// write the bytes
	_, err := fw.Writer.Write(bytes)
	if err != nil {
		return err
	}
	return nil
}
