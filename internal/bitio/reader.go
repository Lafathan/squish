package bitio

import (
	"fmt"
	"io"
)

type bitReader struct {
	reader     io.Reader // io.reader for reading a stream
	buffer     uint64    // buffer holding current streamed bits
	nBits      int       // number of bits currently not read from buffer (cursor)
	readBuffer [8]byte   // bytes to be read when filling in the buffer
}

func NewBitReader(r io.Reader) *bitReader {
	return &bitReader{reader: r}
}

func (br *bitReader) ReadBits(nbits int) (uint64, error) {
	if br.nBits < nbits { // read more bytes to have enough bits
		bytesToRead := (int(nbits) - int(br.nBits) + 7) / 8 // calculate the number of bytes needed
		if int(br.nBits)+bytesToRead*8 > 64 {               // return if reading too many bytes at once
			return 0, fmt.Errorf("bitreader error when reading %d bytes: %w", bytesToRead, io.ErrShortBuffer)
		}
		_, err := io.ReadFull(br.reader, br.readBuffer[:bytesToRead]) // read in the new data
		if err != nil {
			return 0, fmt.Errorf("bitreader error when reading %d bytes: %w", bytesToRead, err)
		}
		for i := range bytesToRead { // add in the new data to the buffer
			br.buffer = (br.buffer << 8) | uint64(br.readBuffer[i]) // shift buffer and add the new byte
			br.nBits += 8                                           // add to total bits in the buffer
		}
	}
	// you want 6 bits
	// nBits  = 10
	// buffer = 1011001100 (read in another byte)
	// mask = 1000000 - 1 = 0111111 (bit mask for six bits you want)
	// right shift the buffer by unread bits - desired bits (10 - 6 = 4)
	// shifted buffer = 0000101100
	// and with mask  =     111111 (prevent high bits from leaking through)
	// result         =     101100var mask uint64
	var mask uint64
	if nbits == 64 {
		mask = ^uint64(0)
	} else {
		mask = (uint64(1) << nbits) - 1
	}
	out := (br.buffer >> (br.nBits - nbits)) & mask
	br.nBits -= nbits                // count down to not re-read bits
	br.buffer &= (1 << br.nBits) - 1 // mask it down to prevent overflow
	return out, nil
}
