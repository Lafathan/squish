package bitio

import (
	"bufio"
	"io"
)

type BitReader struct {
	Reader bufio.Reader // io.reader for reading a stream
	Buffer uint64       // buffer holding current streamed bits
	Nbits  uint8        // number of bits currently not read from buffer (cursor)
}

func NewBitReader(r io.Reader) *BitReader {
	// creates a bit reader from and io reader stream
	bufReader := bufio.NewReader(r)
	return &BitReader{Reader: *bufReader}
}

func (br *BitReader) ReadBits(bits uint8) (uint64, error) {
	// return if trying to read too many bytes at once
	if bits+br.Nbits > 64 {
		return 0, io.ErrShortBuffer
	}
	// if not enough bits are in the buffer, read more bytes until you have enough
	for bits > br.Nbits {
		// read the next byte
		b, err := br.Reader.ReadByte()
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		if err != nil {
			return 0, err
		}
		// pad the buffer with zeros
		br.Buffer <<= 8
		// or it to "append" the new byte to the buffer
		br.Buffer |= uint64(b)
		// add to the total of bits contained in the buffer
		br.Nbits += 8
	}

	// you want 6 bits
	// buffer = 10
	// buffer = 1011001100 (read in another byte)
	// mask = 1000000 - 1 = 0111111 (bit mask for six bits you want)
	// right shift the buffer by unread bits - desired bits (10 - 6 = 4)
	// shifted buffer = 0000101100
	// and with mask  =     111111 (prevent high bits from leaking through)
	// result         =     101100
	out := (br.Buffer >> (br.Nbits - bits)) & (1<<bits - 1)

	// count down to not re-read bits
	br.Nbits -= bits
	return out, nil
}
