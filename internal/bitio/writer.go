package bitio

import (
	"io"
)

type BitWriter struct {
	Writer io.Writer // io.reader for reading a stream
	Buffer uint64    // buffer holding current streamed bits
	Nbits  uint8     // number of bits currently not written to file
}

func NewBitWriter(w io.Writer) *BitWriter {
	// creates a bit reader from and io reader stream
	return &BitWriter{Writer: w}
}

func (bw *BitWriter) WriteBits(bits uint64, nbits uint8) error {
	// add the bits to the current buffer
	bw.Buffer = (bw.Buffer << nbits) + (bits & ((uint64(1) << nbits) - 1))
	// add to the count of unwritten bits
	bw.Nbits += nbits

	// loop through the buffer, building a slice of bytes as you go
	bytesToWrite := bw.Nbits / 8
	bytesBuffer := make([]byte, bytesToWrite)
	for i := range bytesToWrite {
		b := byte((bw.Buffer >> (bw.Nbits - 8)) & ((1 << 8) - 1))
		bw.Nbits -= 8
		bytesBuffer[bytesToWrite-1-i] = b
	}

	// write the bytes
	_, err := bw.Writer.Write(bytesBuffer)
	if err != nil {
		return err
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
	return nil
}
