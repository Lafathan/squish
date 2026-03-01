package bitio

import (
	"fmt"
	"io"
)

type bitWriter struct {
	writer       io.Writer // io.reader for reading a stream
	buffer       uint64    // buffer holding current streamed bits
	nBits        int       // number of bits currently not written to file
	sWriteBuffer [8]byte   // scrach
	sBufShift    int       // scratch
	sBitShift    int       // scratch
	sBuffer      uint64    // scratch
	sBitMask     uint64    // scratch
	sByte        byte      // scratch
}

func NewBitWriter(w io.Writer) *bitWriter {
	return &bitWriter{writer: w}
}

func (bw *bitWriter) clearBuffer() error {
	// writes as many bytes as possible from the buffer to an io.reader
	bytesToWrite := bw.nBits / 8
	for i := range bytesToWrite {
		// shift buffer down to a byte, add the byte to the writeBuffer, repeat for all bytes
		bw.sByte = byte((bw.buffer >> (bw.nBits - 8)) & mask64(8))
		bw.nBits -= 8
		bw.buffer &= mask64(bw.nBits)
		bw.sWriteBuffer[i] = bw.sByte
	}
	// write the bytes
	if _, err := bw.writer.Write(bw.sWriteBuffer[:bytesToWrite]); err != nil {
		return fmt.Errorf("bitwriter error when writing %d bytes: %w", bytesToWrite, err)
	}
	return nil
}

func (bw *bitWriter) WriteBits(bits uint64, nbits int) error {
	// writes bytes to an io.reader, storing extra non-byte-aligned bits in a buffer
	if nbits < 1 {
		return nil
	}
	if nbits > 64 {
		return fmt.Errorf("bitwriter can only write up to 64 bits per call: %w", io.ErrShortBuffer)
	}
	// move the buffer left to fit new bytes and bits
	bw.sBufShift = min(64-bw.nBits, nbits)
	bw.sBitShift = nbits - bw.sBufShift
	bw.sBuffer = bw.buffer << bw.sBufShift
	// mask to prevent overflow
	bw.sBitMask = mask64(nbits - bw.sBitShift)
	// add the new bits to the buffer
	bw.buffer = bw.sBuffer | (bits >> bw.sBitShift & bw.sBitMask)
	// get the number of bits to be added and the new number of bits in the buffer
	nbits = bw.sBitShift
	bw.nBits += bw.sBufShift
	// clear the buffer if it is full
	if bw.nBits == 64 {
		if err := bw.clearBuffer(); err != nil {
			return err
		}
	}
	if nbits > 0 {
		// repeat it all again if there was leftover bits
		bw.sBuffer = bw.buffer << nbits
		bw.sBitMask = mask64(nbits)
		bw.buffer = bw.sBuffer | (bits & bw.sBitMask)
		bw.nBits += nbits
	}
	return nil
}

func (bw *bitWriter) Flush() (int, error) {
	// pad to make byte-aligned then write the buffer
	padding := (8 - bw.nBits%8) % 8
	if padding != 0 {
		if err := bw.WriteBits(0, padding); err != nil {
			return padding, fmt.Errorf("bitwriter error when flushing: %w", err)
		}
	}
	// clear the buffer
	if err := bw.clearBuffer(); err != nil {
		return padding, fmt.Errorf("bitwriter error when flushing: %w", err)
	}
	return padding, nil
}
