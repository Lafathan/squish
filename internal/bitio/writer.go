package bitio

import (
	"fmt"
	"io"
)

type BitWriter struct {
	Writer      io.Writer // io.reader for reading a stream
	Buffer      uint64    // buffer holding current streamed bits
	Nbits       int       // number of bits currently not written to file
	WriteBuffer [8]byte   // bytes to be written when clearing the buffer
}

func NewBitWriter(w io.Writer) *BitWriter {
	return &BitWriter{Writer: w}
}

func (bw *BitWriter) ClearBuffer() error {
	// how many bites need to be written
	bytesToWrite := bw.Nbits / 8
	// slice to store bytes to be written
	for i := range bytesToWrite {
		// same math explained in BitReader
		b := byte((bw.Buffer >> (bw.Nbits - 8)) & ((1 << 8) - 1))
		// reduce by 8 so you don't re-read bytes
		bw.Nbits -= 8
		// mask it down to prevent overflow
		bw.Buffer &= (1 << bw.Nbits) - 1
		// write the byte to the writing buffer
		bw.WriteBuffer[i] = b
	}
	_, err := bw.Writer.Write(bw.WriteBuffer[:bytesToWrite]) // write the bytes
	if err != nil {
		return fmt.Errorf("bitwriter error when writing %d bytes: %w", bytesToWrite, err)
	}
	return err
}

func (bw *BitWriter) WriteBits(bits uint64, nbits int) error {
	if bw.Nbits+nbits > 64 { // if there is not enough room in the buffer to add the new bits
		err := bw.ClearBuffer() // clear the buffer to make room
		if err != nil {
			return err
		}
	}
	if bw.Nbits+nbits > 64 {
		return fmt.Errorf("bitwriter error when writing %d bits: %w", bits, io.ErrShortBuffer)
	}
	// add bits to the current buffer
	bw.Buffer = (bw.Buffer << nbits) + (bits & ((uint64(1) << nbits) - 1))
	// add to the count of unwritten bits
	bw.Nbits += nbits
	return nil
}

func (bw *BitWriter) Flush() (int, error) {
	// pad the bit stream to acheive valid byte length
	padding := (8 - bw.Nbits%8) % 8
	if padding != 0 {
		err := bw.WriteBits(0, padding)
		if err != nil {
			return padding, fmt.Errorf("bitwriter error when flushing: %w", err)
		}
	}
	// then clear the buffer
	err := bw.ClearBuffer()
	if err != nil {
		return padding, fmt.Errorf("bitwriter error when flushing: %w", err)
	}
	return padding, nil
}
