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
	// read in the header of the frame
	header, err := readHeader(fr.reader)
	if err != nil {
		return fmt.Errorf("failed to read frame header: %w", err)
	}
	fr.Header = header
	return fr.Header.valid()
}

func (fr *frameReader) Next() (Block, io.Reader, error) {
	// grab next payload after verifying current payload has been read
	if fr.activePayload != nil && fr.activePayload.N > 0 {
		return Block{}, nil, sqerr.New(sqerr.Internal, "failed to read payload, previous payload still active")
	}
	// read in the block header
	block, err := readBlock(fr)
	if err != nil {
		return block, nil, fmt.Errorf("failed to read block: %w", err)
	}
	// check the validity
	blockError := block.valid()
	if blockError != nil {
		return block, nil, blockError
	}
	// create a payload reader to pass to the pipeline
	fr.activePayload = &io.LimitedReader{R: fr.reader, N: int64(block.CSize)}
	return block, fr.activePayload, nil
}

func (fr *frameReader) Drop() error {
	// drop the current payload
	if fr.activePayload != nil && fr.activePayload.N > 0 {
		_, err := io.Copy(io.Discard, fr.activePayload)
		if err != nil {
			return fmt.Errorf("failed to skip payload: %w", err)
		}
	}
	fr.activePayload = nil
	return nil
}

func (fr *frameReader) ReadBytes(n int) ([]byte, error) {
	// read bytes from a FrameReader stream
	bytes := make([]byte, n)
	_, err := io.ReadFull(fr.reader, bytes)
	if err != nil {
		return bytes, fmt.Errorf("failed to read bytes from frame reader: %w", err)
	}
	return bytes, nil
}

func (fr *frameReader) ReadByte() (byte, error) {
	// read a single byte - implements io.reader
	bytes, err := fr.ReadBytes(1)
	return bytes[0], err
}
