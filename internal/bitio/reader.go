package bitio

import (
	"fmt"
	"io"
)

type bitReader struct {
	reader       io.Reader // io.reader for reading a stream
	buffer       uint64    // buffer holding current streamed bits
	nBits        int       // number of bits currently in the buffer (cursor)
	sReadBuffer  []byte    // scratch
	sBytesToRead int       // scratch
}

func NewBitReader(r io.Reader) *bitReader {
	return &bitReader{reader: r, sReadBuffer: make([]byte, 8)}
}

func (br *bitReader) ReadBits(nbits int) (uint64, error) {
	// reads bytes from an io.reader source and returns an unsigned 64 bit integer containing the requested number of bits
	// extra bits read in when reading bytes are saved in a buffer and prepended to the next request of bits.
	if br.nBits < nbits {
		// if you don't have enough bits in the buffer, determine how many more bytes you need and read them into the buffer
		br.sBytesToRead = (nbits - br.nBits + 7) / 8
		if br.nBits+br.sBytesToRead*8 > 64 {
			return 0, fmt.Errorf("bitreader error when reading %d bytes: %w", br.sBytesToRead, io.ErrShortBuffer)
		}
		_, err := io.ReadFull(br.reader, br.sReadBuffer[:br.sBytesToRead])
		if err != nil {
			return 0, fmt.Errorf("bitreader error when reading %d bytes: %w", br.sBytesToRead, err)
		}
		for i := range br.sBytesToRead {
			// shift the current buffer data and add the new bytes
			br.buffer = (br.buffer << 8) | uint64(br.sReadBuffer[i])
			br.nBits += 8
		}
	}
	// shift the buffer down to just what you need, mask to the desired bits
	out := (br.buffer >> (br.nBits - nbits)) & mask64(nbits)
	// keep track of the bit count and the new buffer (masked down to preven overflow)
	br.nBits -= nbits
	br.buffer &= mask64(br.nBits)
	return out, nil
}
