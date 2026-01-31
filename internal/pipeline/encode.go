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
	header := frame.Header{ // build your header
		Key:          frame.MagicKey,
		Flags:        0x00,
		Codec:        codecIDs,
		ChecksumMode: checksumMode,
	}
	fw := frame.NewFrameWriter(dst, header) // make a framewriter
	err := fw.Ready()                       // write the header
	if err != nil {
		return sqerr.CodedError(err, sqerr.IO, "failed to ready frame writer")
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
				return sqerr.CodedError(err, sqerr.IO, "failed to read from source")
			}
		}
		data := buffer[:n]
		checksum := uint64(0) // determine the checksum values
		if checksumMode&frame.UncompressedChecksum > 0 {
			checksum = uint64(crc32.ChecksumIEEE(data))
		}
		autoCodecIDs := make([]uint8, 1, codec.AutoDepth)
		for _, codecID := range codecIDs {
			currentCodec, ok := codec.CodecMap[codecID]
			if !ok {
				return sqerr.New(sqerr.Unsupported, "unsupported codec ID")
			}
			data, err = currentCodec.EncodeBlock(data) // encode it
			if err != nil {
				return sqerr.CodedError(err, sqerr.Internal, fmt.Sprintf("failed to encode block of data with codec %d", codecID))
			}
			if codecID == codec.AUTO { // grab the codecs used if in auto mode
				autoCodecIDs = append(autoCodecIDs, currentCodec.(codec.AUTOCodec).CodecIDs...)
				break
			}
		}
		if checksumMode&frame.CompressedChecksum > 0 {
			checksum = checksum << (8 * crc32.Size)
			checksum += uint64(crc32.ChecksumIEEE(data))
		}
		bType := frame.DefaultCodec
		if codecIDs[0] == codec.AUTO {
			bType = frame.BlockCodec
		}
		block := frame.Block{ // build the block
			BlockType: uint8(bType),
			USize:     uint64(n),
			CSize:     uint64(len(data)),
			Checksum:  checksum,
			Codec:     autoCodecIDs,
		}
		err = fw.WriteBlock(block, bytes.NewReader(data)) // write the block
		if err != nil {
			return sqerr.CodedError(err, sqerr.IO, "failed to write encoded block")
		}
		if n < blockSize { // break if the last block was not full (partial final block)
			break
		}
	}
	return nil
}
