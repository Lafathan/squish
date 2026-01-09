package pipeline

import (
	"fmt"
	"hash/crc32"
	"io"
	"squish/internal/codec"
	"squish/internal/frame"
)

func Decode(src io.Reader, dst io.Writer) error {
	fr := frame.NewFrameReader(src) // instantiate a FrameReader
	err := fr.Ready()               // read in the header of the stream
	if err != nil {
		return err
	}
	for {
		block, payload, err := fr.Next()
		if err != nil {
			return err
		}
		if block.BlockType == frame.EOS { // break if you reached the EOS
			break
		}
		data, err := io.ReadAll(payload)
		if len(data) != int(block.CSize) {
			return fmt.Errorf("compressed payload does not match CSize: got %d - expected %d", len(data), block.CSize)
		}
		if err != nil {
			return fmt.Errorf("failed to read in payload: %w", err)
		}
		blockCS := block.Checksum
		if fr.Header.ChecksumMode&frame.CompressedChecksum > 0 {
			csm := uint64(crc32.ChecksumIEEE(data))
			exp := (1<<(8*crc32.Size) - 1) & blockCS
			if csm != exp {
				return fmt.Errorf("mismatched compressed payload checksum: got %08x - expected %08x", csm, exp)
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
				return fmt.Errorf("invalid codec ID")
			}
			data, err = currentCodec.DecodeBlock(data) // decode it
			if err != nil {
				return err
			}
			if currentCodec.IsLossless() == false {
				lossless = false
			}
		}
		if fr.Header.ChecksumMode&frame.UncompressedChecksum > 0 && lossless {
			csm := uint64(crc32.ChecksumIEEE(data))
			exp := (1<<(8*crc32.Size) - 1) & blockCS
			if csm != exp {
				return fmt.Errorf("mismatched uncompressed payload checksum: got %08x - expected %08x", csm, exp)
			}
		}
		out, err := dst.Write(data)              // write it out
		if out != int(block.USize) && lossless { // verify the uncompressed payload size
			return fmt.Errorf("uncompressed payload does not match USize: got %d - expected %d", out, block.USize)
		}
		if err != nil {
			return fmt.Errorf("error when writing output of decoding")
		}
	}
	return nil
}
