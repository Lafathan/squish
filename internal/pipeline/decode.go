package pipeline

import (
	"fmt"
	"hash/crc32"
	"io"
	"squish/internal/codec"
	"squish/internal/frame"
	"squish/internal/sqerr"
)

func Decode(src io.Reader, dst io.Writer) error {
	fr := frame.NewFrameReader(src) // instantiate a FrameReader
	err := fr.Ready()               // read in the header of the stream
	if err != nil {
		return sqerr.CodedError(err, sqerr.ReadErrorCode(err), "failed to read input header")
	}
	for {
		block, payload, err := fr.Next()
		if err != nil {
			return sqerr.CodedError(err, sqerr.ReadErrorCode(err), "failed to read input block")
		}
		if block.BlockType == frame.EOS { // break if you reached the EOS
			break
		}
		data := make([]byte, block.CSize)
		n, err := io.ReadFull(payload, data)
		if err != nil {
			return sqerr.CodedError(err, sqerr.ReadErrorCode(err), "failed to read input block")
		}
		if n != int(block.CSize) {
			return sqerr.CodedError(err, sqerr.Corrupt, fmt.Sprintf("mismatched compressed payload size: got %d - expected %d", len(data), block.CSize))
		}
		blockCS := block.Checksum
		if fr.Header.ChecksumMode&frame.CompressedChecksum > 0 {
			csm := uint64(crc32.ChecksumIEEE(data))
			exp := (1<<(8*crc32.Size) - 1) & blockCS
			if csm != exp {
				return sqerr.New(sqerr.Corrupt, fmt.Sprintf("mismatched compressed payload checksum: got %08x - expected %08x", csm, exp))
			}
			blockCS = blockCS >> (8 * crc32.Size)
		}
		codecList := fr.Header.Codec
		if block.BlockType == frame.BlockCodec {
			codecList = block.Codec
		}
		lossless := true
		for i := range len(codecList) {
			currentCodec, ok := codec.CodecMap[codecList[len(codecList)-1-i]] // determine the codec to use
			if !ok {
				return sqerr.New(sqerr.Unsupported, "unsupported codec ID")
			}
			data, err = currentCodec.DecodeBlock(data) // decode it
			if err != nil {
				return sqerr.CodedError(err, sqerr.Corrupt, "failed to decode block")
			}
			if currentCodec.IsLossless() == false {
				lossless = false
			}
		}
		if fr.Header.ChecksumMode&frame.UncompressedChecksum > 0 && lossless {
			csm := uint64(crc32.ChecksumIEEE(data))
			exp := (1<<(8*crc32.Size) - 1) & blockCS
			if csm != exp {
				return sqerr.New(sqerr.Corrupt, fmt.Sprintf("mismatched uncompressed payload checksum: got %08x - expected %08x", csm, exp))
			}
		}
		out, err := dst.Write(data)              // write it out
		if out != int(block.USize) && lossless { // verify the uncompressed payload size
			return sqerr.New(sqerr.Corrupt, fmt.Sprintf("mismatched uncompressed payload size: got %d - expected %d", out, block.USize))
		}
		if err != nil {
			return sqerr.CodedError(err, sqerr.IO, "failed to write output")
		}
	}
	return nil
}
