package frame

import (
	"errors"
	"fmt"
	"io"
)

type frameReader struct {
	reader        io.Reader         // io.reader for reading a stream
	Header        Header            // header of the stream
	activePayload *io.LimitedReader // active payload
}

func NewFrameReader(r io.Reader) *frameReader {
	return &frameReader{reader: r}
}

func (fr *frameReader) Ready() error {
	// read in the header of the frame
	header, err := readHeader(fr.reader)
	if err != nil {
		return fmt.Errorf("frame error when reading header: %w", err)
	}
	fr.Header = header
	return fr.Header.valid()
}

func (fr *frameReader) Next() (Block, io.Reader, error) {
	// double check that there is not an active payload
	if fr.activePayload != nil && fr.activePayload.N > 0 {
		return Block{}, nil, errors.New("early read, previous payload still active")
	}
	// read in the block header
	block, err := readBlock(fr)
	if err != nil {
		return block, nil, fmt.Errorf("frame error when reading block: %w", err)
	}
	// validity check
	blockError := block.valid()
	if blockError != nil {
		return block, nil, blockError
	}
	// generate an io.reader for the payload
	fr.activePayload = &io.LimitedReader{R: fr.reader, N: int64(block.CSize)}

	return block, fr.activePayload, nil
}

func (fr *frameReader) Drop() error {
	// drop current payload
	if fr.activePayload != nil && fr.activePayload.N > 0 {
		_, err := io.Copy(io.Discard, fr.activePayload)
		if err != nil {
			return fmt.Errorf("frame error when skipping payload: %w", err)
		}
	}
	fr.activePayload = nil
	return nil
}

func (fr *frameReader) ReadBytes(n int) ([]byte, error) {
	// read n bytes from a FrameReader stream
	bytes := make([]byte, n)
	_, err := io.ReadFull(fr.reader, bytes)
	if err != nil {
		return bytes, fmt.Errorf("frame error when reading %d bits: %w", n, err)
	}
	return bytes, nil
}

func (fr *frameReader) ReadByte() (byte, error) {
	// read single byte
	bytes, err := fr.ReadBytes(1)
	return bytes[0], err
}
