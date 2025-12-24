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
	compressed := make([]byte, frame.MaxBlockSize)
	for {
		block, payload, err := fr.Next()
		if err != nil {
			return err
		}
		if block.BlockType == frame.EOS { // break if you reached the EOS
			break
		}
		in, err := io.ReadFull(payload, compressed) // dump payload to byte slice
		if in != int(block.CSize) {                 // verify compressed payload size
			return fmt.Errorf("Payload does not match CSize")
		}
		if err != nil {
			return err
		}
		blockCS := block.Checksum
		if fr.Header.ChecksumMode&frame.CompressedChecksum > 0 {
			if uint64(crc32.ChecksumIEEE(compressed)) != 0xFF&blockCS {
				return fmt.Errorf("Mismatched checksum for compressed payload")
			}
			blockCS = blockCS >> 8
		}
		currentCodec := codec.CodecMap[fr.Header.Codec] // determine the codec to use
		if block.BlockType == frame.BlockCodec {
			currentCodec = codec.CodecMap[block.Codec]
		}
		uncompressed, err := currentCodec.DecodeBlock(compressed, block.PadBits) // decode it
		if err != nil {
			return err
		}
		if fr.Header.ChecksumMode&frame.UncompressedChecksum > 0 {
			if uint64(crc32.ChecksumIEEE(uncompressed)) != 0xFF&blockCS {
				return fmt.Errorf("Mismatched checksum for uncompressed payload")
			}
		}
		out, err := dst.Write(uncompressed) // write it out
		if out != int(block.USize) {        // verify the uncompressed payload size
			return fmt.Errorf("Payload does not match USize")
		}
		if err != nil {
			return err
		}
	}
	return nil
}
