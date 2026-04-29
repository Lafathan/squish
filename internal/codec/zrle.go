package codec

import (
	"encoding/binary"
	"squish/internal/sqerr"
)

// ======================================================================================
// Zero Run Length Encoding
//
// This codec works by taking runs of zeros and representing them with a zero valued byte
// followed by an unsigned variable length integer representation of the run length.
// All non-zero values are carried over to the output.
//
// ex: [0 0 0 0 0 1 2 2 3 3 3 0 0 0 0 1 2 2 3 3] -> [0 5 1 2 2 3 3 3 0 4 1 2 2 3 2]
//     [0 (repeated 69420 times)  1 2 2 3 3 4 5] -> [0 68 147 44 1 2 2 3 2 4 5]
//
// The variable length integer is defined in the binary package as the following:
// - unsigned integers are serialized 7 bits at a time, starting with the least
//   significant bits
// - the most significant bit (msb) in each output byte indicates if there is a
//   continuation byte (msb = 1)
// ======================================================================================

type ZRLECodec struct{}

func (ZRLECodec) EncodeBlock(src []byte) ([]byte, error) {
	// run length encode all runs of zeros (0x00) using a variable length integer length
	if len(src) == 0 {
		return src, nil
	}
	var (
		runLen uint64 = 0
		srcIdx int    = 0
		out    []byte = make([]byte, 0, len(src))
	)
	for srcIdx < len(src) {
		// encode all zeros with a zero followed by the length of the run
		if src[srcIdx] == 0x00 {
			if runLen == 0 {
				out = append(out, 0x00)
			}
			// count up all the zero valued bytes
			runLen++
		} else {
			if runLen > 0 {
				// encode run length using a variable length integer
				out = binary.AppendUvarint(out, runLen)
				// reset to do it again
				runLen = 0
			}
			out = append(out, src[srcIdx]) // append non-zero literals
		}
		srcIdx++
	}
	// flush last run of zeros
	if runLen > 0 {
		out = binary.AppendUvarint(out, runLen)
	}
	return out, nil
}

func (ZRLECodec) DecodeBlock(src []byte) ([]byte, error) {
	// decode run length encoding by expanding runs for the literal 0x00
	if len(src) == 0 {
		return src, nil
	}
	var (
		srcIdx           = 0
		outLength uint64 = 0
		runLen    uint64
		bytes     int // bytes read per run length varint
	)
	// first pass for counting output length
	for srcIdx < len(src) {
		if src[srcIdx] == 0x00 {
			if srcIdx+1 > len(src) {
				return []byte{}, sqerr.New(sqerr.Corrupt, "Invalid run length format")
			}
			// read the run length value and number of bytes
			runLen, bytes = binary.Uvarint(src[srcIdx+1:])
			if bytes >= len(src)-srcIdx || runLen < 1 {
				return []byte{}, sqerr.New(sqerr.Corrupt, "Invalid run length")
			}
			outLength += runLen
			srcIdx += bytes
		}
		outLength++
		srcIdx++
	}
	// second pass, expand zero runs to full length
	srcIdx = 0
	out := make([]byte, 0, outLength)
	for srcIdx < len(src) {
		if src[srcIdx] == 0x00 {
			// read the run length value and number of bytes
			runLen, bytes = binary.Uvarint(src[srcIdx+1:])
			// continually add zeros to meet run length
			// go defaults slice values to 0, so allocation is all that is required
			out = out[:len(out)+int(runLen)]
			srcIdx += bytes
		} else {
			// add the literal if not a zero
			out = append(out, src[srcIdx])
		}
		srcIdx++
	}
	return out, nil
}

func (ZRLECodec) IsLossless() bool {
	return true
}
