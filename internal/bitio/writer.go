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
	sBufShift   int       // scratch
	sBitShift   int       // scratch
	sBuffer     uint64    // scratch
	sBits       uint64    // scratch
	sBitMask    uint64    // scratch
	sByte       byte      // scratch
}

func NewBitWriter(w io.Writer) *bitWriter {
	return &bitWriter{writer: w}
}

func mask64(n int) uint64 {
	if n == 64 {
		return ^uint64(0)
	}
	return (1 << n) - 1
}

func (bw *bitWriter) clearBuffer() error {
	bytesToWrite := bw.nBits / 8 // how many bites need to be written
	for i := range bytesToWrite {
		bw.sByte = byte((bw.buffer >> (bw.nBits - 8)) & ((1 << 8) - 1)) // same math explained in BitReader
		bw.nBits -= 8                                                   // reduce by 8 to not re-read bytes
		bw.buffer &= (1 << bw.nBits) - 1                                // mask it down to prevent overflow
		bw.writeBuffer[i] = bw.sByte                                    // write the byte to the writing buffer
	}
	_, err := bw.writer.Write(bw.writeBuffer[:bytesToWrite]) // write the bytes
	if err != nil {
		return fmt.Errorf("bitwriter error when writing %d bytes: %w", bytesToWrite, err)
	}
	return nil
}

func (bw *bitWriter) WriteBits(bits uint64, nbits int) error {
	if nbits < 1 {
		return nil
	}
	if nbits > 64 {
		return fmt.Errorf("bitwriter can only write up to 64 bits per call: %w", io.ErrShortBuffer)
	}
	bw.sBufShift = min(64-bw.nBits, nbits)            // how much to move the buffer left to fit the new bits
	bw.sBitShift = nbits - bw.sBufShift               // how much to shift the bits right to put msb in buffer
	bw.sBuffer = bw.buffer << bw.sBufShift            // shift the buffer
	bw.sBits = bits >> bw.sBitShift                   // shift the bits
	bw.sBitMask = mask64(nbits - bw.sBitShift)        // make a mask so there is no overflow/errors
	bw.buffer = bw.sBuffer | (bw.sBits & bw.sBitMask) // get the new buffer with added bits on the end
	nbits = bw.sBitShift                              // get the new number of bits to be added
	bw.nBits += bw.sBufShift                          // get the new number of bits in the buffer
	if bw.nBits == 64 {
		err := bw.clearBuffer() // clear the buffer if it is full
		if err != nil {
			return err
		}
	}
	if nbits > 0 {
		bw.sBuffer = bw.buffer << nbits               // shift the buffer to hold left over bits
		bw.sBitMask = mask64(nbits)                   // make a new mask
		bw.sBitMask = uint64((1 << nbits) - 1)        // make a new mask
		bw.buffer = bw.sBuffer | (bits & bw.sBitMask) // add bits to the current buffer
		bw.nBits += nbits                             // add to the count of unwritten bits
	}
	return nil
}

func (bw *bitWriter) Flush() (int, error) {
	padding := (8 - bw.nBits%8) % 8 // pad the bit stream to acheive valid byte length
	if padding != 0 {
		err := bw.WriteBits(0, padding)
		if err != nil {
			return padding, fmt.Errorf("bitwriter error when flushing: %w", err)
		}
	}
	err := bw.clearBuffer() // then clear the buffer
	if err != nil {
		return padding, fmt.Errorf("bitwriter error when flushing: %w", err)
	}
	return padding, nil
}
