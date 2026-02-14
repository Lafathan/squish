package codec

import "encoding/binary"

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
	for srcIdx < len(src) {
		if src[srcIdx] == 0x00 {
			if runLen == 0 {
				outBytes = append(outBytes, 0x00)
			}
			runLen++
		} else {
			if runLen > 0 {
				outBytes = binary.AppendUvarint(outBytes, runLen)
				runLen = 0
			}
			outBytes = append(outBytes, src[srcIdx])
		}
		srcIdx++
	}
	if runLen > 0 {
		outBytes = binary.AppendUvarint(outBytes, runLen)
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
		run       uint64
		bytes     int
	)
	for srcIdx < len(src) {
		if src[srcIdx] == 0x00 {
			run, bytes = binary.Uvarint(src[srcIdx+1:])
			outLength += run
			srcIdx += bytes
		}
		outLength++
		srcIdx++
	}
	srcIdx = 0
	outBytes := make([]byte, 0, outLength)
	for srcIdx < len(src) {
		if src[srcIdx] == 0x00 {
			run, bytes = binary.Uvarint(src[srcIdx+1:])
			for range run {
				outBytes = append(outBytes, 0x00)
			}
			srcIdx += bytes
		} else {
			outBytes = append(outBytes, src[srcIdx])
		}
		srcIdx++
	}
	return outBytes, nil
}

func (ZRLECodec) IsLossless() bool {
	return true
}
