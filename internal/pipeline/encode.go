package pipeline

import (
	"bytes"
	"errors"
	"hash/crc32"
	"io"
	"squish/internal/codec"
	"squish/internal/frame"
)

func Encode(src io.Reader, dst io.Writer, codecID uint8, blockSize uint64, checksumMode uint8) error {
	header := frame.Header{ // build your header
		Key:          frame.MagicKey,
		Flags:        0x00,
		Codec:        codecID,
		ChecksumMode: checksumMode,
	}
	fw := frame.NewFrameWriter(dst, header) // make a framewriter
	err := fw.Ready()                       // write the header
	if err != nil {
		return err
	}
	defer fw.Close()                               // defer the close to write EOS block
	blockSize = min(blockSize, frame.MaxBlockSize) // validate blockSize first
	for {
		blockReader := io.LimitedReader{R: src, N: int64(blockSize)} // make a new LimitedReader
		uncompressed, err := io.ReadAll(&blockReader)                // read it all in from the src
		if err != nil {
			return err
		}
		if len(uncompressed) == 0 {
			break
		}
		currentCodec, ok := codec.CodecMap[codecID]
		if !ok {
			return errors.New("Invalid codec ID")
		}
		compressed, padBits, err := currentCodec.EncodeBlock(uncompressed) // encode it
		if err != nil {
			return err
		}
		checksum := uint64(0) // determine the checksum values
		if checksumMode&frame.UncompressedChecksum > 0 {
			checksum = uint64(crc32.ChecksumIEEE(uncompressed))
		}
		if checksumMode&frame.CompressedChecksum > 0 {
			checksum = checksum << (8 * crc32.Size)
			checksum += uint64(crc32.ChecksumIEEE(compressed))
		}
		block := frame.Block{ // build the block
			BlockType: frame.DefaultCodec,
			USize:     uint64(len(uncompressed)),
			CSize:     uint64(len(compressed)),
			PadBits:   padBits,
			Checksum:  checksum,
		}
		err = fw.WriteBlock(block, bytes.NewReader(compressed)) // write the block
		if err != nil {
			return err
		}
	}
	return nil
}
