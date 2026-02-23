package codec

import (
	"math"
	"sort"
)

const (
	minProbeLen int = 1 << 14 // minimum size of payload chunk to test compression
	maxProbeLen int = 1 << 16 // maximum size of payload chunk to test compression
)

type AUTOCodec struct {
	CodecIDs []uint8
}

type result struct {
	codecIDs    []uint8
	payloadSize uint32
}

type features struct {
	e float64 // entropy
	z float64 // zero ratio
	r float64 // run ratio
	d float64 // duplicate ratio
}

func getFeatures(src []byte) features {
	f := features{}
	f.e = entropy(src)
	f.z = zeroRatio(src)
	f.r = runRatio(src)
	f.d = duplicateRatio(src)
	return f
}

func getPayloadProbe(src []byte) []byte {
	// get a smaller data set to compress for determining the best codecs
	if len(src) < minProbeLen {
		return src
	}
	// default to 1/8 the block size and clamp it
	probeLength := len(src) / 8
	probeLength = max(probeLength, minProbeLen)
	probeLength = min(probeLength, maxProbeLen)
	if probeLength == len(src) {
		// compress the entire payload if it is small
		return src
	}
	// get the indexes so your data set is from the middle of the payload
	startIdx := (len(src) - probeLength) / 2
	endIdx := startIdx + probeLength
	return src[startIdx:endIdx]
}

func entropy(data []byte) float64 {
	// calculate the entropy of a dataset
	if len(data) == 0 {
		return 0
	}
	var (
		p = [256]float64{}
		e float64
	)
	for i := range len(data) {
		p[data[i]]++
	}
	n := float64(len(data))
	for i := range len(p) {
		p[i] /= n
	}
	for i := range len(p) {
		if p[i] > 0 {
			e -= p[i] * math.Log2(p[i])
		}
	}
	return e
}

func zeroRatio(data []byte) float64 {
	// calculate the ratio of zero valued data
	if len(data) == 0 {
		return 0
	}
	var (
		z float64
	)
	for i := range len(data) {
		if data[i] == 0x00 {
			z++
		}
	}
	return z / float64(len(data))
}

func runRatio(data []byte) float64 {
	// calculate the ratio of runs
	if len(data) < 3 {
		return 0
	}
	var (
		r float64
	)
	for i := 3; i < len(data); i++ {
		if data[i] == data[i-1] && data[i] == data[i-2] {
			r++
		}
	}
	return r / float64(len(data)-2)
}

func duplicateRatio(data []byte) float64 {
	// calculate the ratio of repeated 4 byte segments
	if len(data) < 4 {
		return 0
	}
	var (
		byteLength = 4
		lookback   = 1 << 12
		seen       = make(map[uint32]int, len(data))
		d          float64
		h          = uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
	)
	seen[h] = 0
	for i := 1; i < len(data)-byteLength; i++ {
		h = (h << 8) | uint32(data[i+byteLength-1])
		if last, ok := seen[h]; ok && i-last <= lookback {
			d++
		}
		seen[h] = i
	}
	return d / float64(len(data)-byteLength+1)
}

func getNextCodecs(f features) [][]uint8 {
	// choose valid pipelines based on data features
	codecs := [][]uint8{{RAW}}
	if f.z > 0.03 || (f.z > 0.015 && f.e <= 6.8) {
		codecs = append(codecs, []uint8{ZRLE, HUFFMAN})
	}
	if f.r > 0.04 && f.e < 7.6 {
		codecs = append(codecs, []uint8{RLE, HUFFMAN})
	}
	if f.d > 0.05 {
		codecs = append(codecs, []uint8{LZSS, HUFFMAN})
	}
	if f.e < 7.2 && f.d > 0.03 {
		codecs = append(codecs, []uint8{BWT, MTF, HUFFMAN})
		codecs = append(codecs, []uint8{BWT, MTF, ZRLE, HUFFMAN})
	}
	if f.e < 7.4 && f.d < 0.04 && f.r < 0.03 {
		codecs = append(codecs, []uint8{HUFFMAN})
	}
	return codecs
}

func (AC *AUTOCodec) EncodeBlock(src []byte) ([]byte, error) {
	// determines a best-set of transforms and codecs to use and encodes the data
	if len(src) == 0 {
		return src, nil
	}
	var (
		probe     = getPayloadProbe(src)
		f         = getFeatures(probe)
		pipelines = getNextCodecs(f)
		results   = make([]result, 0, len(pipelines))
		err       error
	)
	// apply all the pipelines to the probe data set
	for _, pipeline := range pipelines {
		payload := append([]byte(nil), probe...)
		success := true
		for _, codecID := range pipeline {
			payload, err = CodecMap[codecID].EncodeBlock(payload)
			if err != nil {
				success = false
				break
			}
		}
		if !success {
			continue
		}
		results = append(results, result{
			codecIDs:    append([]byte(nil), pipeline...),
			payloadSize: uint32(len(payload)),
		})
	}
	// sort to pick the best one based on payload size
	sort.Slice(results, func(i, j int) bool {
		return results[i].payloadSize < results[j].payloadSize
	})
	// set the AUTOCodec codecIDs so the encoder can put them in the block header
	AC.CodecIDs = append([]byte(nil), results[0].codecIDs...)
	// get the encoded data
	for _, codecID := range AC.CodecIDs {
		src, err = CodecMap[codecID].EncodeBlock(src)
		if err != nil {
			return src, err
		}
	}
	return src, nil
}

func (*AUTOCodec) DecodeBlock(src []byte) ([]byte, error) {
	return src, nil
}

func (*AUTOCodec) IsLossless() bool {
	return true
}
