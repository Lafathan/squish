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
	primaryRecipes    = []uint8{HUFFMAN, LZSS, RLE, RLE2, RLE3, RLE4}
	subsequentRecipes = []uint8{HUFFMAN, LZSS}
)

type AUTOCodec struct {
	CodecIDs []uint8
}

type result struct {
	codecIDs []uint8 // result for iteration to keep track of payload size and codecs used
	payload  []byte
}

func getPayloadProbe(src []byte) []byte {
	if len(src) < minProbeLen {
		return src
	}
	probeLength := len(src) / 8                 // default to 1/8th of the source payload
	probeLength = max(probeLength, minProbeLen) // clamp it on the low end
	probeLength = min(probeLength, maxProbeLen) // clamp it on the high end
	if probeLength == len(src) {
		return src // compress the entire payload if it is small
	}
	startIdx := (len(src) - probeLength) / 2 // take a chunk from the middle of the payload
	endIdx := startIdx + probeLength
	return src[startIdx:endIdx]
}

func getFilteredResults(results []result) []result {
	sort.Slice(results, func(i, j int) bool {
		return len(results[i].payload) < len(results[j].payload)
	})
	return results[:keepAlong] // keep the 'keepAlong' number of results into next iteration
}

func (AC *AUTOCodec) EncodeBlock(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return src, nil
	}
	var (
		probe   []byte   = getPayloadProbe(src)                   // get the payload test chunk
		results []result = make([]result, 0, len(primaryRecipes)) // make a slice to store results
	)
	for _, codecID := range primaryRecipes {
		resID := []uint8{codecID} // make a results slice for each primary recipe
		resPayload, err := CodecMap[codecID].EncodeBlock(probe)
		if err != nil {
			continue
		}
		results = append(results, result{
			codecIDs: resID,
			payload:  resPayload,
		})
	}
	for range AutoDepth - 1 { // loop through the iterations
		newResults := make([]result, 0, len(subsequentRecipes)*len(results)) // new iteration results
		for j := range len(results) {                                        // go through the old results
			if results[j].codecIDs[len(results[j].codecIDs)-1] == HUFFMAN {
				newResults = append(newResults, results[j]) // carry huffman results into next iteration
			}
			for _, codecID := range subsequentRecipes { // go through the next recipes
				res, err := CodecMap[codecID].EncodeBlock(results[j].payload) // encode further
				if err != nil {
					continue
				}
				base := results[j].codecIDs                                                  // store current codecs
				newCodecIDs := append(append([]uint8(nil), base...), codecID)                // append the new codec id to it
				newResults = append(newResults, result{codecIDs: newCodecIDs, payload: res}) // store the result
			}
		}
		results = getFilteredResults(newResults) // get the 'keepAlong' best results
	}
	AC.CodecIDs = append([]uint8(nil), results[0].codecIDs...) // store the best of the best
	if len(probe) == len(src) {
		return results[0].payload, nil // don't redo the compression if you already did it while probing
	}
	var (
		data []byte = src
		err  error
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
