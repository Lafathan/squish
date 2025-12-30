package bitio

import (
	"fmt"
	"io"
)

type BitWriter struct {
	Writer io.Writer // io.reader for reading a stream
	Buffer byte      // buffer holding current bits to be written
	Nbits  int       // number of bits currently not written to file
}

func NewBitWriter(w io.Writer) *BitWriter {
	return &BitWriter{Writer: w}
}

func (bw *BitWriter) WriteBits(bytes []byte, nbits int) error {
	if nbits > 8*len(bytes) {
		return fmt.Errorf("bitwriter error: not enough bits in byte slice")
	}
	bytesBuffer := make([]byte, (bw.Nbits + nbits - 1) / 8 + 1)
	for i, b := range bytes
		// if old and new bits to be written are over a byte
		if bw.Nbits + nbits - 8 * i >= 8 { 
			// left shift buffer to make room for LSB of right shifted current byte
			bw.Buffer = (bw.Buffer << (8 - bw.Nbits)) | (b >> bw.Nbits)
			// add the new byte to the writing buffer
			bytesBuffer[i] = bw.Buffer 
			// the new buffer is what you didn't write from current byte
			bw.Buffer = b & (1 << bw.Nbits - 1) 
		} else {
			// store the remaining bits if they fit in the buffer
			bw.Buffer = (bw.Buffer << nbits % 8) | b
		}
		bw.Nbits = (bw.Nbits + nbits % 8) % 8
	}
	_, err := bw.Writer.Write(bytesBuffer) // write the bytes
	if err != nil {
		return fmt.Errorf("bitwriter error when writing %d bytes: %v", len(bytesBuffer), err)
	}
	return nil
}

func (bw *BitWriter) Flush() (int, error) {
	// pad the bit stream to acheive valid byte length
	padding := 8 - bw.Nbits%8
	if padding != 0 {
		err := bw.WriteBits([]byte{0}, padding)
		if err != nil {
			return padding, fmt.Errorf("bitwriter error when flushing: %v", err)
		}
	}
	return padding, nil
}
