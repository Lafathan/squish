package frame

import (
	"fmt"
	"io"
	"squish/internal/sqerr"
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
	header, err := readHeader(fr.reader) // read in the header of the frame
	if err != nil {
		return fmt.Errorf("failed to read frame header: %w", err)
	}
	fr.Header = header
	return fr.Header.valid()
}

func (fr *frameReader) Next() (Block, io.Reader, error) {
	if fr.activePayload != nil && fr.activePayload.N > 0 { // double check for an active payload
		return Block{}, nil, sqerr.New(sqerr.Internal, "failed to read payload, previous payload still active")
	}
	block, err := readBlock(fr) // read in the block header
	if err != nil {
		return block, nil, fmt.Errorf("failed to read block: %w", err)
	}
	blockError := block.valid() // validity check
	if blockError != nil {
		return block, nil, blockError
	}
	fr.activePayload = &io.LimitedReader{R: fr.reader, N: int64(block.CSize)} // create payload io.reader
	return block, fr.activePayload, nil
}

func (fr *frameReader) Drop() error {
	if fr.activePayload != nil && fr.activePayload.N > 0 { // drop current payload
		_, err := io.Copy(io.Discard, fr.activePayload)
		if err != nil {
			return fmt.Errorf("failed to skip payload: %w", err)
		}
	}
	fr.activePayload = nil
	return nil
}

func (fr *frameReader) ReadBytes(n int) ([]byte, error) {
	bytes := make([]byte, n) // read n bytes from a FrameReader stream
	_, err := io.ReadFull(fr.reader, bytes)
	if err != nil {
		return bytes, fmt.Errorf("failed to read bytes from frame reader: %w", err)
	}
	return bytes, nil
}

func (fr *frameReader) ReadByte() (byte, error) {
	bytes, err := fr.ReadBytes(1) // read single byte
	return bytes[0], err
}
