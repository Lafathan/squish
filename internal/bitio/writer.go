package bitio

import (
	"bufio"
	"io"
)

type BitWriter struct {
	Writer bufio.Writer // io.reader for reading a stream
	Buffer uint64       // buffer holding current streamed bits
	Nbits  uint8        // number of bits currently not written to file
}

func NewBitWriter(w io.Writer) *BitWriter {
	// creates a bit reader from and io reader stream
	bufWriter := bufio.NewWriter(w)
	return &BitWriter{Writer: *bufWriter}
}

func (bw *BitWriter) WriteBits(bits uint64, nbits uint8) error {
	// add the bits to the current buffer
	bw.Buffer = (bw.Buffer << nbits) + (bits & ((1 << nbits) - 1))
	// add to the count of unwritten bits
	bw.Nbits += nbits

	// loop through the buffer, writing bytes as you go
	for bw.Nbits >= 8 {
		out := byte((bw.Buffer >> (bw.Nbits - 8)) & ((1 << 8) - 1))
		err := bw.Writer.WriteByte(out)
		if err != nil {
			return err
		}
		bw.Nbits -= 8
	}
	return nil
}

func (bw *BitWriter) Flush() error {
	// pad the bit stream to acheive valid byte length
	padding := (8 - bw.Nbits%8) % 8
	if padding != 0 {
		// pad it up if necessary
		err := bw.WriteBits(0, padding)
		if err != nil {
			return err
		}
	}
	// flush it
	err := bw.Writer.Flush()
	if err != nil {
		return err
	}
	return nil
}
