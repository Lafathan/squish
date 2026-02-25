package codec

import (
	"bytes"
	"math"
	"sort"
)

const (
	minProbeLen  int = 1 << 14 // minimum size of payload chunk to test compression
	maxProbeLen  int = 1 << 16 // maximum size of payload chunk to test compression
	maxPipelines int = 4       // maximum number of pipelines to test
)

var transforms = [][]uint8{{RAW}, {DELTA}, {XOR}, {BWT, MTF}}

type AUTOCodec struct {
	CodecIDs []uint8
}

type candidatePipeline struct {
	transform      []uint8
	transformProbe []byte
	pipeline       []uint8
	score          float64
	payloadSize    uint32
}

type features struct {
	e float64 // normalized entropy
	z float64 // zero run ratio
	s float64 // single stride run ratio
	d float64 // double stride run ratio
	t float64 // triple stride run ratio
	q float64 // quadruple stride run ratio
	m float64 // match ratio
	u float64 // unique values ratio
}

func getFeatures(src []byte) features {
	f := features{}
	f.e = entropyNorm(src)
	f.z = zeroRunRatio(src)
	f.s = runRatio(src, 1)
	f.d = runRatio(src, 2)
	f.t = runRatio(src, 3)
	f.q = runRatio(src, 4)
	f.m = matchRatio(src)
	f.u = uniqueRatio(src)
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
	return clampFloat(e/8, 0, 1)
}

func zeroRunRatio(data []byte) float64 {
	// calculate the ratio of zero valued data
	if len(data) == 0 {
		return 0
	}
	var z float64
	for i := 1; i < len(data); i++ {
		if data[i] == 0x00 && data[i-1] == 0x00 {
			z++
		}
	}
	return z / float64(len(data)-1)
}

func runRatio(data []byte, s int) float64 {
	// calculate the ratio of runs
	if len(data) < 3*s || s <= 0 {
		return 0
	}
	var r float64
	for i := 1; i < len(data)/s; i++ {
		a := (i - 1) * s
		b := i * s
		if bytes.Equal(data[a:a+s], data[b:b+s]) {
			r++
		}
	}
	return r / float64(len(data)/s-1)
}

func matchRatio(data []byte) float64 {
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

func getScoredPipelines(f features) []candidatePipeline {
	// choose valid pipelines based on data features
	var (
		candidates []candidatePipeline
		lowEnt     = 1.0 - f.e
		lowUniq    = 1.0 - f.u
		zClump     = math.Sqrt(f.z) * (0.35 + 0.65*f.s)
		runMax     = max(max(f.s, f.d), max(f.t, f.q))
	)
	// HUFFMAN score
	candidates = append(candidates, candidatePipeline{
		pipeline: []uint8{HUFFMAN},
		score:    clampFloat(0.70*lowEnt+0.15*lowUniq-0.20*runMax-0.20*f.m, 0, 1),
	})
	// RLE-HUFFMAN score
	candidates = append(candidates, candidatePipeline{
		pipeline: []uint8{RLE, HUFFMAN},
		score:    0.65*f.s + 0.35*lowEnt,
	})
	// RLE2-HUFFMAN score
	candidates = append(candidates, candidatePipeline{
		pipeline: []uint8{RLE2, HUFFMAN},
		score:    0.65*f.d + 0.35*lowEnt,
	})
	// RLE3-HUFFMAN score
	candidates = append(candidates, candidatePipeline{
		pipeline: []uint8{RLE3, HUFFMAN},
		score:    0.65*f.t + 0.35*lowEnt,
	})
	// RLE4-HUFFMAN score
	candidates = append(candidates, candidatePipeline{
		pipeline: []uint8{RLE4, HUFFMAN},
		score:    0.65*f.q + 0.35*lowEnt,
	})
	// ZRLE-HUFFMAN score
	candidates = append(candidates, candidatePipeline{
		pipeline: []uint8{ZRLE, HUFFMAN},
		score:    0.70*zClump + 0.16*lowEnt + 0.14*f.s,
	})
	// LZSS-HUFFMAN score
	candidates = append(candidates, candidatePipeline{
		pipeline: []uint8{LZSS, HUFFMAN},
		score:    clampFloat(0.58*f.m+0.18*lowEnt+0.12*lowUniq-0.55*runMax-0.20*zClump, 0, 1),
	})
	return candidates
}

func getNBestScoredPipelines(c []candidatePipeline, n int) []candidatePipeline {
	// sort the candidate piplelines based on their score
	sort.Slice(c, func(i, j int) bool {
		return c[i].score > c[j].score
	})
	return c[:min(n, len(c))]
}

func (AC *AUTOCodec) EncodeBlock(src []byte) ([]byte, error) {
	// determines a best-set of transforms and codecs to use and encodes the data
	if len(src) == 0 {
		return src, nil
	}
	var (
		probe      = getPayloadProbe(src)
		candidates []candidatePipeline
	)
	// get the features and then scored pipelines for all transforms
	for _, transform := range transforms {
		transformProbe, err := EncodePipeline(probe, transform)
		if err != nil {
			continue
		}
		features := getFeatures(transformProbe)
		transformCandidates := getScoredPipelines(features)
		for _, tpl := range transformCandidates {
			tpl.transform = transform
			tpl.transformProbe = transformProbe
			candidates = append(candidates, tpl)
		}
	}
	// get the best candidates
	bestCandidates := getNBestScoredPipelines(candidates, maxPipelines)
	// apply all the pipelines to the probe data set
	for i, c := range bestCandidates {
		payload, err := EncodePipeline(c.transformProbe, c.pipeline)
		if err != nil {
			continue
		}
		bestCandidates[i].payloadSize = uint32(len(payload))
	}
	// sort to pick the best one based on resulting payload size
	sort.Slice(bestCandidates, func(i, j int) bool {
		return bestCandidates[i].payloadSize < bestCandidates[j].payloadSize
	})
	// set the AUTOCodec codecIDs so the encoder can put them in the block header
	AC.CodecIDs = append(bestCandidates[0].transform, bestCandidates[0].pipeline...)
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
