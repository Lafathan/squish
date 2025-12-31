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
	if nbits-bw.Nbits > 8*len(bytes) {
		return fmt.Errorf("bitwriter error: not enough bits in byte slice")
	}
	byteBuffer := make([]byte, (nbits-1)/8+1)
	if nbits <= bw.Nbits {
		// easy case - all required bits are already in the buffer
		byteBuffer[0] = bw.Buffer >> (bw.Nbits - nbits) // put the bits in the output
		bw.Buffer &= (1<<(bw.Nbits-nbits) - 1)          // shift the buffer by how many wanted
		bw.Nbits -= nbits                               // track how many are still unread
	} else {
		// harder case - more bytes are required to be read in
		rem := nbits % 8 // get the remainder of bits desired for the MSByte
		if rem == 0 {
			rem = 8
		}
		idx := 0 // track where you are in the output
		if bw.Nbits >= rem {
			// when the entire leading MSB is contained in the buffer bits
			shift := bw.Nbits - rem              // determine shift required to take just enough from buffer
			byteBuffer[idx] = bw.Buffer >> shift // add MSB to output
			bw.Buffer &= (1<<shift - 1)          // make out what you read frombuffer
			bw.Nbits = shift                     // reduce unread bits by what you read from buffer
			rem = 8                              // all future output bytes will be 8 bits
			idx++                                // increment your output index
		}
		for _, b := range bytes {
			shift := rem - bw.Nbits                                     // determine shift required to take enough from buffer
			byteBuffer[idx] = (bw.Buffer << shift) | (b >> (8 - shift)) // shift buffer and append from MSb from read byte
			bw.Buffer = b & (1<<(8-shift) - 1)                          // the new buffer is the tail LSb of the read byte
			bw.Nbits = 8 - shift                                        // unread bits is updated
			rem = 8                                                     // all future output bytes will be 8 bits
			idx++                                                       // increment your output index
			if idx >= len(byteBuffer) {
				break
			}
		}
	}
	_, err := bw.Writer.Write(byteBuffer) // write the bytes
	if err != nil {
		return fmt.Errorf("bitwriter error when writing %d bytes: %v", len(byteBuffer), err)
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
