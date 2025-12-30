package bitio

import (
	"fmt"
	"io"
)

type BitWriter struct {
	Writer io.Writer // io.reader for reading a stream
	Buffer uint64    // buffer holding current streamed bits
	Nbits  uint8     // number of bits currently not written to file
}

func NewBitWriter(w io.Writer) *BitWriter {
	return &BitWriter{Writer: w}
}

func (bw *BitWriter) WriteBits(bits uint64, nbits uint8) error {
	// add bits to the current buffer
	bw.Buffer = (bw.Buffer << nbits) + (bits & ((uint64(1) << nbits) - 1))
	// add to the count of unwritten bits
	bw.Nbits += nbits
	// how many bites need to be written
	bytesToWrite := bw.Nbits / 8
	// slice to store bytes to be written
	bytesBuffer := make([]byte, bytesToWrite)
	for i := range bytesToWrite {
		// same math explained in BitReader
		b := byte((bw.Buffer >> (bw.Nbits - 8)) & ((1 << 8) - 1))
		// reduce by 8 so you don't re-read bytes
		bw.Nbits -= 8
		// write the byte to the writing buffer
		bytesBuffer[bytesToWrite-1-i] = b
	}
	_, err := bw.Writer.Write(bytesBuffer) // write the bytes
	if err != nil {
		return fmt.Errorf("bitwriter error when writing %d bytes: %v", len(bytesBuffer), err)
	}
	return nil
}

func (bw *BitWriter) WriteBitsFromSlice(byteSlice []byte, nbits uint8) error {
	// writes n bits to a writer from a byte array
	// []byte{00110011, 00001111} with nbits 13 -> 11100110011
	// note: lower index is less significant
	if int(nbits) > 8*len(byteSlice) {
		return fmt.Errorf("Too many bits to write from byte slice")
	}
	// determine which element in the slice to start on
	byteIndex := (nbits - 1) / 8
	// preempt writing 8 bits per byte
	bitLength := uint8(8)
	// write all the full bytes
	for byteIndex > 0 {
		err := bw.WriteBits(uint64(byteSlice[byteIndex]), bitLength)
		if err != nil {
			return fmt.Errorf("bitwriter error when writing byte slice: %v", err)
		}
		byteIndex--
	}
	// write the remaining bits of the last byte
	err := bw.WriteBits(uint64(byteSlice[byteIndex]), nbits%8)
	if err != nil {
		return fmt.Errorf("bitwriter error when writing byte slice: %v", err)
	}
	return nil
}

func (bw *BitWriter) Flush() (uint8, error) {
	// pad the bit stream to acheive valid byte length
	padding := (8 - bw.Nbits%8) % 8
	if padding != 0 {
		err := bw.WriteBits(0, padding)
		if err != nil {
			return padding, fmt.Errorf("bitwriter error when flushing: %v", err)
		}
	}
	return padding, nil
}
