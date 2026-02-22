package pipeline

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"io"
	"slices"
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
	// determine if AUTO is used
	useAUTO := slices.Contains(codecIDs, codec.AUTO)
	for {
		bType := frame.DefaultCodec
		blockCodecIDs := append([]uint8(nil), codecIDs...)
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
		if useAUTO {
			// if AUTO codec is being used
			bType = frame.BlockCodec
			currentCodec := codec.AUTOCodec{}
			data, err = currentCodec.EncodeBlock(data)
			if err != nil {
				return sqerr.CodedError(err, sqerr.Internal, fmt.Sprintf("failed to encode block of data with codec %d", codec.AUTO))
			}
			blockCodecIDs = currentCodec.CodecIDs
		} else {
			// if AUTO is not being used
			for _, codecID := range codecIDs {
				// apply all codecs in pipeline
				currentCodec, ok := codec.CodecMap[codecID]
				if !ok {
					return sqerr.New(sqerr.Unsupported, "unsupported codec ID")
				}
				data, err = currentCodec.EncodeBlock(data)
				if err != nil {
					return sqerr.CodedError(err, sqerr.Internal, fmt.Sprintf("failed to encode block of data with codec %d", codecID))
				}
			}
		}
		// perform compressed checksums
		if checksumMode&frame.CompressedChecksum > 0 {
			checksum = checksum << (8 * crc32.Size)
			checksum += uint64(crc32.ChecksumIEEE(data))
		}
		// build and write the block
		block := frame.Block{
			BlockType: bType,
			USize:     uint64(n),
			CSize:     uint64(len(data)),
			Checksum:  checksum,
			Codec:     blockCodecIDs,
		}
		if err = fw.WriteBlock(block, bytes.NewReader(data)); err != nil {
			return sqerr.CodedError(err, sqerr.IO, "failed to write encoded block")
		}
		// break if the last block was not full (partial final block)
		if n < blockSize {
			break
		}
	}
	return nil
}
