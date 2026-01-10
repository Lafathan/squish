package bitio

import (
	"fmt"
	"io"
)

type bitWriter struct {
	writer      io.Writer // io.reader for reading a stream
	buffer      uint64    // buffer holding current streamed bits
	nBits       int       // number of bits currently not written to file
	writeBuffer [8]byte   // bytes to be written when clearing the buffer
}

func NewBitWriter(w io.Writer) *bitWriter {
	return &bitWriter{writer: w}
}

func (bw *bitWriter) clearBuffer() error {
	// how many bites need to be written
	bytesToWrite := bw.nBits / 8
	// slice to store bytes to be written
	for i := range bytesToWrite {
		// same math explained in BitReader
		b := byte((bw.buffer >> (bw.nBits - 8)) & ((1 << 8) - 1))
		// reduce by 8 so you don't re-read bytes
		bw.nBits -= 8
		// mask it down to prevent overflow
		bw.buffer &= (1 << bw.nBits) - 1
		// write the byte to the writing buffer
		bw.writeBuffer[i] = b
	}
	_, err := bw.writer.Write(bw.writeBuffer[:bytesToWrite]) // write the bytes
	if err != nil {
		return fmt.Errorf("bitwriter error when writing %d bytes: %w", bytesToWrite, err)
	}
	return err
}

func (bw *bitWriter) WriteBits(bits uint64, nbits int) error {
	if bw.nBits+nbits > 64 { // if there is not enough room in the buffer to add the new bits
		err := bw.clearBuffer() // clear the buffer to make room
		if err != nil {
			return err
		}
	}
	if bw.nBits+nbits > 64 {
		return fmt.Errorf("bitwriter error when writing %d bits: %w", bits, io.ErrShortBuffer)
	}
	// add bits to the current buffer
	bw.buffer = (bw.buffer << nbits) + (bits & ((uint64(1) << nbits) - 1))
	// add to the count of unwritten bits
	bw.nBits += nbits
	return nil
}

func (bw *bitWriter) Flush() (int, error) {
	// pad the bit stream to acheive valid byte length
	padding := (8 - bw.nBits%8) % 8
	if padding != 0 {
		err := bw.WriteBits(0, padding)
		if err != nil {
			return padding, fmt.Errorf("bitwriter error when flushing: %w", err)
		}
	}
	// then clear the buffer
	err := bw.clearBuffer()
	if err != nil {
		return padding, fmt.Errorf("bitwriter error when flushing: %w", err)
	}
	return padding, nil
}
