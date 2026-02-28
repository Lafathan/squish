package pipeline

import (
	"fmt"
	"hash/crc32"
	"io"
	"squish/internal/codec"
	"squish/internal/frame"
	"squish/internal/sqerr"
	"sync"
)

type decodeWorkspace struct {
	data []byte
}

var decodePool = sync.Pool{
	// creates a new workspace
	New: func() any {
		return &decodeWorkspace{}
	},
}

func growB(slice []byte, length int) []byte {
	// function to grow slices
	if cap(slice) < length {
		return make([]byte, length)
	}
	return slice[:length]
}

func Decode(src io.Reader, dst io.Writer) error {
	// read the header
	fr := frame.NewFrameReader(src)
	if err := fr.Ready(); err != nil {
		return sqerr.CodedError(err, sqerr.ReadErrorCode(err), "failed to read input header")
	}
	for {
		// loop through reading each block
		block, payload, err := fr.Next()
		if err != nil {
			return sqerr.CodedError(err, sqerr.ReadErrorCode(err), "failed to read input block")
		}
		// break if you reach EOS
		if block.BlockType == frame.EOS {
			break
		}
		// instantiate a workspace
		ws := decodePool.Get().(*decodeWorkspace)
		ws.data = growB(ws.data, int(block.CSize))
		// read the block
		n, err := io.ReadFull(payload, ws.data)
		if err != nil {
			return sqerr.CodedError(err, sqerr.ReadErrorCode(err), "failed to read input block")
		}
		// verify compressed size
		if n != int(block.CSize) {
			return sqerr.CodedError(err, sqerr.Corrupt, fmt.Sprintf("mismatched compressed payload size: got %d - expected %d", len(ws.data), block.CSize))
		}
		// verify checksums
		blockCS := block.Checksum
		if fr.Header.ChecksumMode&frame.CompressedChecksum > 0 {
			csm := uint64(crc32.ChecksumIEEE(ws.data))
			exp := (1<<(8*crc32.Size) - 1) & blockCS
			if csm != exp {
				return sqerr.New(sqerr.Corrupt, fmt.Sprintf("mismatched compressed payload checksum: got %08x - expected %08x", csm, exp))
			}
			blockCS = blockCS >> (8 * crc32.Size)
		}
		// apply block level codecs if necessary
		codecList := fr.Header.Codec
		if block.BlockType == frame.BlockCodec {
			codecList = block.Codec
		}
		lossless := true
		// apply the codecs in reverse
		ws.data, err = codec.DecodePipeline(ws.data, codecList)
		if err != nil {
			return err
		}
		// determine if lossy codecs were used
		for i := range len(codecList) {
			if codec.CodecMap[codecList[i]].IsLossless() == false {
				lossless = false
			}
		}
		// verify checksums if there was no lossy compression
		if fr.Header.ChecksumMode&frame.UncompressedChecksum > 0 && lossless {
			csm := uint64(crc32.ChecksumIEEE(ws.data))
			exp := (1<<(8*crc32.Size) - 1) & blockCS
			if csm != exp {
				return sqerr.New(sqerr.Corrupt, fmt.Sprintf("mismatched uncompressed payload checksum: got %08x - expected %08x", csm, exp))
			}
		}
		// write the uncompressed data
		out, err := dst.Write(ws.data)
		// verify uncompressed size
		if out != int(block.USize) && lossless {
			return sqerr.New(sqerr.Corrupt, fmt.Sprintf("mismatched uncompressed payload size: got %d - expected %d", out, block.USize))
		}
		if err != nil {
			return sqerr.CodedError(err, sqerr.IO, "failed to write output")
		}
		decodePool.Put(ws)
	}
	return nil
}
