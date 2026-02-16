package codec

import (
	"encoding/binary"
	"squish/internal/sqerr"
)

type ZRLECodec struct {
	byteLength int
	lossless   bool
}

func (ZRLECodec) EncodeBlock(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return src, nil
	}
	var (
		runLen   uint64 = 0                         // length of current run
		srcIdx   int    = 0                         // index as you traverse the source
		outBytes []byte = make([]byte, 0, len(src)) // encoded bytes
	)
	for srcIdx < len(src) { // loop through elements
		if src[srcIdx] == 0x00 { // if it is a zero
			if runLen == 0 {
				outBytes = append(outBytes, 0x00) // make the run start with a zero
			}
			runLen++ // increment the run
		} else { // if it is not a zero
			if runLen > 0 { // if ending a run
				outBytes = binary.AppendUvarint(outBytes, runLen) // append the run length
				runLen = 0                                        // reset the run
			}
			outBytes = append(outBytes, src[srcIdx]) // append the new non-zero literal
		}
		srcIdx++ // go to next element
	}
	if runLen > 0 {
		outBytes = binary.AppendUvarint(outBytes, runLen) // flush last run
	}
	return outBytes, nil
}

func (ZRLECodec) DecodeBlock(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return src, nil
	}
	var (
		srcIdx           = 0 // where you are in the source
		outLength uint64 = 0 // first pass variable for allocating for decoding
		run       uint64     // current run length
		bytes     int        // bytes read per run length varint
	)
	// first pass for counting output length
	for srcIdx < len(src) { // loop through the elements
		if src[srcIdx] == 0x00 { // if it is a zero
			if srcIdx+1 > len(src) {
				return []byte{}, sqerr.New(sqerr.Corrupt, "Invalid run length format")
			}
			run, bytes = binary.Uvarint(src[srcIdx+1:]) // read the varint of the run length
			if bytes >= len(src)-srcIdx || run < 1 {
				return []byte{}, sqerr.New(sqerr.Corrupt, "Invalid run length")
			}
			outLength += run // add the number of bytes to the length
			srcIdx += bytes  // move past the varint bytes
		}
		outLength++ // add the literal to the length
		srcIdx++    // move past the literal byte
	}
	srcIdx = 0                             // start at the beginning again
	outBytes := make([]byte, 0, outLength) // make a slice for the output
	for srcIdx < len(src) {                // loop through the elements
		if src[srcIdx] == 0x00 { // if it is a zero
			run, bytes = binary.Uvarint(src[srcIdx+1:]) // read the varint of the run length
			for range run {
				outBytes = append(outBytes, 0x00) // add a zero run times
			}
			srcIdx += bytes // skip past the varint
		} else {
			outBytes = append(outBytes, src[srcIdx]) // add the literal
		}
		srcIdx++ // move past the literal
	}
	return outBytes, nil
}

func (ZRLECodec) IsLossless() bool {
	return true
}
