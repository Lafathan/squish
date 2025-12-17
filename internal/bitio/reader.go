package bitio

import (
	"io"
)

type BitReader struct {
	Reader io.Reader // io.reader for reading a stream
	Buffer uint64    // buffer holding current streamed bits
	Nbits  uint8     // number of bits currently not read from buffer (cursor)
}

func NewBitReader(r io.Reader) *BitReader {
	// creates a bit reader from an io reader stream
	return &BitReader{Reader: r}
}

func (br *BitReader) ReadBits(bits uint8) (uint64, error) {
	// if not enough bits are in the buffer, read enough bytes to have enough
	if br.Nbits < bits {
		// calculate the number of bytes needed
		bytesToRead := (int(bits) - int(br.Nbits) + 7) / 8
		// return if trying to read too many bytes at once
		if int(br.Nbits)+bytesToRead*8 > 64 {
			return 0, io.ErrShortBuffer
		}
		bytesBuffer := make([]byte, bytesToRead)
		_, err := io.ReadFull(br.Reader, bytesBuffer)
		if err != nil {
			return 0, err
		}
		for _, b := range bytesBuffer {
			// pad the buffer and or it to "append" the new byte to the buffer
			br.Buffer = (br.Buffer << 8) | uint64(b)
			// add to the total of bits contained in the buffer
			br.Nbits += 8
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
	out := (br.Buffer >> (br.Nbits - bits)) & mask

	// count down to not re-read bits
	br.Nbits -= bits
	return out, nil
}
