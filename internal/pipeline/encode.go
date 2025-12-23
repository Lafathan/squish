package pipeline

import (
	"bytes"
	"hash/crc32"
	"io"
	"squish/internal/codec"
	"squish/internal/frame"
)

func Encode(src io.Reader, dst io.Writer, codecID uint8, blockSize int64, checksumMode uint8) error {
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
	defer fw.Close()                                      // defer the close to write EOS block
	blockSize = min(blockSize, int64(frame.MaxBlockSize)) // validate blockSize first
	uncompressed := make([]byte, blockSize)               // make a buffer to hold uncompressed data
	for {
		in, err := io.ReadFull(src, uncompressed) // read in the src data into uncompressed
		if err == io.EOF || in == 0 {
			break
		}
		if err != nil && err != io.ErrUnexpectedEOF {
			return err
		}
		compressed, padBits, err := codec.CodecMap[codecID].EncodeBlock(&uncompressed) // encode it
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
			USize:     uint64(in),
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
