package pipeline

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"io"
	"squish/internal/codec"
	"squish/internal/frame"
)

func Encode(src io.Reader, dst io.Writer, codecIDs []uint8, blockSize int, checksumMode uint8) error {
	header := frame.Header{ // build your header
		Key:          frame.MagicKey,
		Flags:        0x00,
		Codec:        codecIDs,
		ChecksumMode: checksumMode,
	}
	fw := frame.NewFrameWriter(dst, header) // make a framewriter
	err := fw.Ready()                       // write the header
	if err != nil {
		return fmt.Errorf("failed to ready frame writer: %w", err)
	}
	defer fw.Close()                               // defer the close to write EOS block
	blockSize = min(blockSize, frame.MaxBlockSize) // validate blockSize first
	buffer := make([]byte, blockSize)
	for {
		n, err := src.Read(buffer)
		if n == 0 {
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("failed to read from source: %w", err)
			}
		}
		data := buffer[:n]
		checksum := uint64(0) // determine the checksum values
		if checksumMode&frame.UncompressedChecksum > 0 {
			checksum = uint64(crc32.ChecksumIEEE(data))
		}
		for _, codecID := range codecIDs {
			currentCodec, ok := codec.CodecMap[codecID]
			if !ok {
				return fmt.Errorf("invalid codec ID")
			}
			data, err = currentCodec.EncodeBlock(data) // encode it
			if err != nil {
				return fmt.Errorf("failed to encode block of data with codec %d: %w", codecID, err)
			}
		}
		if checksumMode&frame.CompressedChecksum > 0 {
			checksum = checksum << (8 * crc32.Size)
			checksum += uint64(crc32.ChecksumIEEE(data))
		}
		block := frame.Block{ // build the block
			BlockType: frame.DefaultCodec,
			USize:     uint64(n),
			CSize:     uint64(len(data)),
			Checksum:  checksum,
		}
		err = fw.WriteBlock(block, bytes.NewReader(data)) // write the block
		if err != nil {
			return fmt.Errorf("failed to write encoded block: %w", err)
		}
		if n < blockSize { // break if the last block was not full (partial final block)
			break
		}
	}
	return nil
}
