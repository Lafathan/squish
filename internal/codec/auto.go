package codec

import (
	"sort"
)

const (
	AutoDepth   int = 3       // how many iterations of encodings to test
	keepAlong   int = 3       // how many "best" results to test from prev iteration
	minProbeLen int = 1 << 14 // minimum size of payload chunk to test compression
	maxProbeLen int = 1 << 16 // maximum size of payload chunk to test compression
)

var (
	transforms        = [][]uint8{{RAW}, {BWT, MTF}}
	primaryRecipes    = []uint8{HUFFMAN, LZSS, RLE, RLE2, RLE3, RLE4, ZRLE}
	subsequentRecipes = []uint8{HUFFMAN, LZSS, RLE, RAW}
)

type AUTOCodec struct {
	CodecIDs []uint8
}

type result struct {
	codecIDs []uint8
	payload  []byte
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

func getFilteredResults(results []result) []result {
	// keep a limited number of the best results based on compressed payload size
	sort.Slice(results, func(i, j int) bool {
		return len(results[i].payload) < len(results[j].payload)
	})
	return results[:keepAlong]
}

func (AC *AUTOCodec) EncodeBlock(src []byte) ([]byte, error) {
	// determines a best-set of transforms and codecs to use and encodes the data
	if len(src) == 0 {
		return src, nil
	}
	var (
		probe      []byte   = getPayloadProbe(src)
		results    []result = make([]result, 0, len(primaryRecipes)*len(transforms))
		err        error
		resPayload []byte
	)
	// perform the first transform-and-primary-recipes pass
	for _, transformPipeline := range transforms {
		tProbe := make([]byte, len(probe))
		tProbe = append(tProbe, probe...)
		for _, transformID := range transformPipeline {
			tProbe, err = CodecMap[transformID].EncodeBlock(tProbe)
			if err != nil {
				continue
			}
		}
		for _, codecID := range primaryRecipes {
			resPayload, err = CodecMap[codecID].EncodeBlock(tProbe)
			if err != nil {
				continue
			}
			results = append(results, result{
				codecIDs: append(transformPipeline, codecID),
				payload:  resPayload,
			})
		}
	}
	// perform the iterative subsequent-recipes pass
	for range AutoDepth - 1 {
		newResults := make([]result, 0, len(subsequentRecipes)*len(results))
		for j := range len(results) {
			for _, codecID := range subsequentRecipes {
				resPayload, err = CodecMap[codecID].EncodeBlock(results[j].payload)
				if err != nil {
					continue
				}
				newResults = append(newResults, result{
					codecIDs: append(append([]uint8(nil), results[j].codecIDs...), codecID),
					payload:  resPayload,
				})
			}
		}
		results = getFilteredResults(newResults) // get the 'keepAlong' best results
	}
	AC.CodecIDs = append([]uint8(nil), results[0].codecIDs...) // store the best of the best
	var (
		data []byte = src
	)
	for _, codecID := range results[0].codecIDs {
		data, err = CodecMap[codecID].EncodeBlock(data) // encode it with best codecs
		if err != nil {
			return data, err
		}
	}
	return data, err
}

func (*AUTOCodec) DecodeBlock(src []byte) ([]byte, error) {
	return src, nil
}

func (*AUTOCodec) IsLossless() bool {
	return true
}
