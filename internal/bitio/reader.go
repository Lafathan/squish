package bitio

import (
	"fmt"
	"io"
)

type bitReader struct {
	reader io.Reader // io.reader for reading a stream
	buffer uint64    // buffer holding current streamed bits
	nBits  int       // number of bits currently not read from buffer (cursor)
}

func NewBitReader(r io.Reader) *bitReader {
	return &bitReader{reader: r}
}

func (br *bitReader) ReadBits(bits int) (uint64, error) {
	// read more bytes to have enough bits
	if br.nBits < bits {
		// calculate the number of bytes needed
		bytesToRead := (int(bits) - int(br.nBits) + 7) / 8
		// return if reading too many bytes at once
		if int(br.nBits)+bytesToRead*8 > 64 {
			return 0, fmt.Errorf("bitreader error when reading %d bytes: %w", bytesToRead, io.ErrShortBuffer)
		}
		bytesBuffer := make([]byte, bytesToRead)
		_, err := io.ReadFull(br.reader, bytesBuffer)
		if err != nil {
			return 0, fmt.Errorf("bitreader error when reading %d bytes: %w", bytesToRead, err)
		}
		for _, b := range bytesBuffer {
			// pad the buffer and 'or' it add the new byte to the buffer
			br.buffer = (br.buffer << 8) | uint64(b)
			// add to the total of bits contained in the buffer
			br.nBits += 8
		}
	}
	// you want 6 bits
	// buffer = 10
	// buffer = 1011001100 (read in another byte)
	// mask = 1000000 - 1 = 0111111 (bit mask for six bits you want)
	// right shift the buffer by unread bits - desired bits (10 - 6 = 4)
	// shifted buffer = 0000101100
	// and with mask  =     111111 (prevent high bits from leaking through)
	// result         =     101100var mask uint64
	var mask uint64
	if bits == 64 {
		mask = ^uint64(0)
	} else {
		mask = (uint64(1) << bits) - 1
	}
	out := (br.buffer >> (br.nBits - bits)) & mask
	br.nBits -= bits                 // count down to not re-read bits
	br.buffer &= (1 << br.nBits) - 1 // mask it down to prevent overflow
	return out, nil
}
