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
	if nbits < 1 {
		return nil
	}
	if nbits > 8*len(bytes) {
		return fmt.Errorf("bitwriter error: not enough bits in byte slice")
	}
	out := make([]byte, 0, (nbits-1)/8+1)
	byteIdx := 0        // what input byte are you reading from
	bitIdx := nbits % 8 // what bit in the input byte are you reading
	if bitIdx == 0 {
		bitIdx = 8
	}
	for nbits > 0 {
		bit := (bytes[byteIdx] >> (bitIdx - 1)) & 1 // get the bit value
		bw.Buffer = (bw.Buffer << 1) | bit          // append it to the lsb of the buffer
		bw.Nbits++                                  // keep track of buffer bits
		bitIdx--                                    // count down the bits your reading
		if bitIdx == 0 {
			byteIdx++  // if you finished the byte, go to the next byte
			bitIdx = 8 // and go to it's msb
		}
		if bw.Nbits == 8 {
			out = append(out, bw.Buffer) // dump the buffer when full
			bw.Buffer = 0                // reset the buffer
			bw.Nbits = 0                 // reset the buffed bit count
		}
		nbits--
	}
	_, err := bw.Writer.Write(out) // write the bytes
	if err != nil {
		return fmt.Errorf("bitwriter error when writing %d bytes: %w", len(out), err)
	}
	return nil
}

func (bw *BitWriter) Flush() (int, error) {
	if bw.Nbits == 0 {
		return 0, nil
	}
	// pad the bit stream to acheive valid byte length
	padding := 8 - bw.Nbits
	if padding != 0 {
		err := bw.WriteBits([]byte{0}, padding)
		if err != nil {
			return padding, fmt.Errorf("bitwriter error when flushing: %w", err)
		}
		bw.Buffer = 0
		bw.Nbits = 0
	}
	return padding, nil
}
