package pipeline

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"io"
	"squish/internal/codec"
	"squish/internal/frame"
	"squish/internal/sqerr"
)

func Encode(src io.Reader, dst io.Writer, codecIDs []uint8, blockSize int, checksumMode uint8) error {
	// build and write the header
	header := frame.Header{
		Key:          frame.MagicKey,
		Flags:        0x00,
		Codec:        codecIDs,
		ChecksumMode: checksumMode,
	}
	fw := frame.NewFrameWriter(dst, header)
	if err := fw.Ready(); err != nil {
		return sqerr.CodedError(err, sqerr.IO, "failed to ready frame writer")
	}
	defer fw.Close()
	// validate blockSize first
	blockSize = min(blockSize, frame.MaxBlockSize)
	buffer := make([]byte, blockSize)
	for {
		// read in the source data
		n, err := src.Read(buffer)
		if n == 0 {
			if err == io.EOF {
				break
			}
			if err != nil {
				return sqerr.CodedError(err, sqerr.IO, "failed to read from source")
			}
		}
		data := buffer[:n]
		// perform uncompressed checksums
		checksum := uint64(0)
		if checksumMode&frame.UncompressedChecksum > 0 {
			checksum = uint64(crc32.ChecksumIEEE(data))
		}
		// apply the codecs noting if AUTO is used
		autoCodecIDs := make([]uint8, 0, codec.AutoDepth)
		for _, codecID := range codecIDs {
			currentCodec, ok := codec.CodecMap[codecID]
			if !ok {
				return sqerr.New(sqerr.Unsupported, "unsupported codec ID")
			}
			data, err = currentCodec.EncodeBlock(data)
			if err != nil {
				return sqerr.CodedError(err, sqerr.Internal, fmt.Sprintf("failed to encode block of data with codec %d", codecID))
			}
			if codecID == codec.AUTO {
				// grab the actual codecs used if in AUTO mode
				autoCodecIDs = append(autoCodecIDs, currentCodec.(*codec.AUTOCodec).CodecIDs...)
				break
			}
		}
		// perform compressed checksums
		if checksumMode&frame.CompressedChecksum > 0 {
			checksum = checksum << (8 * crc32.Size)
			checksum += uint64(crc32.ChecksumIEEE(data))
		}
		// special AUTO case
		bType := frame.DefaultCodec
		bCodecsID := codecIDs
		if codecIDs[0] == codec.AUTO {
			bType = frame.BlockCodec
			bCodecsID = autoCodecIDs
		}
		// build and write the block
		block := frame.Block{
			BlockType: uint8(bType),
			USize:     uint64(n),
			CSize:     uint64(len(data)),
			Checksum:  checksum,
			Codec:     bCodecsID,
		}
		if err = fw.WriteBlock(block, bytes.NewReader(data)); err != nil {
			return sqerr.CodedError(err, sqerr.IO, "failed to write encoded block")
		}
		if n < blockSize { // break if the last block was not full (partial final block)
			break
		}
	}
	return nil
}
