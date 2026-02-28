package codec

import (
	"math"
	"sort"
)

const (
	maxTProbeLength int = 1 << 13 // max size of test probe for transformed data
	maxPProbeLength int = 1 << 11 // max size of test probe for compressed data
	tCandidates     int = 4       // max number of transforms to test
)

var transforms = [][]uint8{
	{RAW},
	{DELTA},
	{XOR},
	{BWT, MTF},
}

var pipelines = [][]uint8{
	{RAW},
	{HUFFMAN},
	{RLE, HUFFMAN},
	{RLE2, HUFFMAN},
	{RLE3, HUFFMAN},
	{RLE4, HUFFMAN},
	{ZRLE, HUFFMAN},
	{LZSS, HUFFMAN},
}

type AUTOCodec struct {
	CodecIDs []uint8
}

type transformResult struct {
	transform        []uint8
	transformedProbe []byte
	entropy          float64
	uniqueRatio      float64
}

type compressionResult struct {
	pipeline         []uint8
	compressionRatio float64
}

func getProbe(src []byte, size int) []byte {
	// get a smaller data set to transform/compress
	if len(src) <= size {
		return src
	}
	// get the indexes so your data set is from the middle of the payload
	start := (len(src) - size) / 2
	end := start + size
	return src[start:end]
}

func entropyNorm(data []byte) float64 {
	// calculate the normalized entropy of a dataset
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
	return clampFloat(e/8.0, 0, 1)
}

func uniqueRatio(data []byte) float64 {
	// calculate the percentage of possible byte values appearing in data
	if len(data) == 0 {
		return 0
	}
	var (
		seen          = make([]uint32, 256)
		count float64 = 0
	)
	byteHistogram(data, seen)
	for i := range len(seen) {
		if seen[i] > 0 {
			count++
		}
	}
	return count / 256.0
}

func (AC *AUTOCodec) EncodeBlock(src []byte) ([]byte, error) {
	// determines a best-set of transforms and codecs to use and encodes the data
	if len(src) == 0 {
		return src, nil
	}
	var (
		tProbe   = getProbe(src, maxTProbeLength)                           // probe for testing transforms
		tResults = make([]transformResult, 0, len(transforms))              // tranformed entropy results
		cResults = make([]compressionResult, 0, len(pipelines)*tCandidates) // compression ratio results
	)
	// get the entropy and unique values for the transformed probes
	for _, transform := range transforms {
		tr := transformResult{transform: transform}
		transformedProbe, err := EncodePipeline(tProbe, transform)
		if err != nil {
			continue
		}
		tr.transformedProbe = transformedProbe
		tr.entropy = entropyNorm(tr.transformedProbe)
		tr.uniqueRatio = uniqueRatio(tr.transformedProbe)
		tResults = append(tResults, tr)
	}
	// sort the transformed candidates based on entropy and/or unique byte value ratio
	sort.Slice(tResults, func(i, j int) bool {
		if tResults[i].entropy == tResults[j].entropy {
			return tResults[i].uniqueRatio < tResults[j].uniqueRatio
		} else {
			return tResults[i].entropy < tResults[j].entropy
		}
	})
	// apply all the pipelines to the best transform data set probes
	for _, tResult := range tResults[:tCandidates] {
		for _, pipeline := range pipelines {
			pp := getProbe(tResult.transformedProbe, maxPProbeLength)
			c, err := EncodePipeline(pp, pipeline)
			if err != nil {
				continue
			}
			ce := compressionResult{
				pipeline:         append(tResult.transform, pipeline...),
				compressionRatio: float64(len(c)) / float64(len(pp)),
			}
			cResults = append(cResults, ce)
		}
	}
	// sort the compression results based on compression ratio
	sort.Slice(cResults, func(i, j int) bool {
		return cResults[i].compressionRatio < cResults[j].compressionRatio
	})
	// set the AUTOCodec codecIDs so the encoder can put them in the block header
	AC.CodecIDs = cResults[0].pipeline
	// get the encoded data
	out, err := EncodePipeline(src, AC.CodecIDs)
	return out, err
}

func (*AUTOCodec) DecodeBlock(src []byte) ([]byte, error) {
	return src, nil
}

func (*AUTOCodec) IsLossless() bool {
	return true
}
