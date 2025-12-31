package bitio

import (
	"fmt"
	"io"
)

type BitReader struct {
	Reader io.Reader // io.reader for reading a stream
	Buffer byte      // buffer holding current streamed bits
	Nbits  int       // number of bits currently not read from buffer (cursor)
}

func NewBitReader(r io.Reader) *BitReader {
	return &BitReader{Reader: r}
}

func (br *BitReader) ReadBits(nbits int) ([]byte, error) {
	byteBuffer := make([]byte, (nbits-1)/8+1)
	if nbits <= br.Nbits {
		// easy case - all required bits are already in the buffer
		byteBuffer[0] = br.Buffer >> (br.Nbits - nbits) // put the bits in the output
		br.Buffer &= (1<<(br.Nbits-nbits) - 1)          // shift the buffer by how many wanted
		br.Nbits -= nbits                               // track how many are still unread
	} else {
		// harder case - more bytes are required to be read in
		bytes := make([]byte, (nbits-br.Nbits+7)/8)
		_, err := io.ReadFull(br.Reader, bytes)
		if err != nil {
			return byteBuffer, fmt.Errorf("bitreader error when reading %d bytes: %v", len(byteBuffer), err)
		}
		rem := nbits % 8 // get the remainder of bits desired for the MSByte
		if rem == 0 {
			rem = 8
		}
		idx := 0 // track where you are in the output
		for _, b := range bytes {
			if br.Nbits >= rem {
				// when the entire leading MSB is contained in the buffer bits
				shift := br.Nbits - rem              // determine shift required to take just enough from buffer
				byteBuffer[idx] = br.Buffer >> shift // add MSB to output
				br.Buffer &= (1<<shift - 1)          // make out what you read frombuffer
				br.Nbits = shift                     // reduce unread bits by what you read from buffer
				rem = 8                              // all future output bytes will be 8 bits
				idx++                                // increment your output index
			}
			shift := rem - br.Nbits                                     // determine shift required to take enough from buffer
			byteBuffer[idx] = (br.Buffer << shift) | (b >> (8 - shift)) // shift buffer and append from MSb from read byte
			br.Buffer = b & (1<<(8-shift) - 1)                          // the new buffer is the tail LSb of the read byte
			br.Nbits = 8 - shift                                        // unread bits is updated
			rem = 8                                                     // all future output bytes will be 8 bits
			idx++                                                       // increment your output index
		}
	}
	return byteBuffer, nil
}
