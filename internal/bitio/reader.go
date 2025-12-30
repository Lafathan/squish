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
	bytesBuffer := make([]byte, (br.Nbits + nbits - 1) / 8 + 1)
	bytes := make([]byte, (nbits - br.Nbits - 1) / 8 + 1)
	_, err := io.ReadFull(br.Reader, readBytes)
	if err != nil {
		return bytesBuffer, fmt.Errorf("bitreader error when reading %d bytes: %v", len(bytesBuffer), err)
	}
	for i, b := range bytes {
		// if old and new bits to be read are over a byte
		if br.Nbits + nbits - 8 * i > 8 {
			// left shift buffer to make room for LSB of right shifted current byte
			br.Buffer = (br.Buffer << (8 - br.Nbits)) | (b >> br.Nbits) 
			// add the new byte to the writing buffer
			bytesBuffer[i] = br.Buffer 
			// the new buffer is what you didn't write from current byte
			bw.Buffer = b & (1 << br.Nbits - 1) 
		} else {
			// store the remaining bits if they fit in the buffer
			br.Buffer = (br.Buffer << nbits % 8) | b
		}
		br.Nbits = (br.Nbits + nbits % 8) % 8
	}
	return bytesBuffer, nil
}
