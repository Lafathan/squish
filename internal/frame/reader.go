package frame

import (
	"bufio"
	"encoding/binary"
	"io"
)

type FrameReader struct {
	Reader bufio.Reader // io.reader for reading a stream
	Header Header       // header of the stream
}

func NewFrameReader(r io.Reader) *FrameReader {
	// create a FrameReader from an io reader stream
	bufReader := bufio.NewReader(r)
	// create an empty header for now
	var h Header
	return &FrameReader{Reader: *bufReader, Header: h}
}

func (fr *FrameReader) Ready() error {
	// read in the header of the frame
	bytes, err := fr.ReadBytes(7)
	if err != nil {
		return err
	}

	// assign values to the header of the FrameReader
	fr.Header.Key = string(bytes[:4])
	fr.Header.Flags = bytes[4]
	fr.Header.Codec = bytes[5]
	fr.Header.ChecksumMode = bytes[6]

	// validity check
	headerError := fr.Header.valid()

	return headerError
}

func (fr *FrameReader) Next() (Block, io.Reader, error) {
	// read in the block header
	var b Block
	// read in the blockType first to make sure there is more to read
	blockType, err := fr.ReadBytes(1)
	if err != nil {
		return b, nil, err
	}
	if blockType[0] == 0 {
		return b, nil, nil // EOS block type encountered
	}
	bytes, err := fr.ReadBytes(26)
	if err != nil {
		return b, nil, err
	}

	// assign values to the block
	b.BlockType = blockType[0]
	b.Codec = bytes[0]
	b.USize = binary.BigEndian.Uint64(bytes[1:9])
	b.CSize = binary.BigEndian.Uint64(bytes[9:17])
	b.ChecksumMethod = bytes[17]
	b.Checksum = binary.BigEndian.Uint64(bytes[18:26])

	// validity check
	blockError := b.valid()
	if blockError != nil {
		return b, nil, blockError
	}

	// generate a io.reader for the payload
	payloadReader := io.LimitReader(&fr.Reader, int64(b.CSize))

	return b, payloadReader, nil
}

func (fr *FrameReader) ReadBytes(n int) ([]byte, error) {
	// read n bytes from a FrameReader stream
	bytes := make([]byte, n)
	for i := range bytes {
		newByte, err := fr.Reader.ReadByte()
		if err != nil {
			return bytes, err
		}
		bytes[i] = newByte
	}
	return bytes, nil
}
