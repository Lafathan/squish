package codec

import (
	"encoding/binary"
	"squish/internal/sqerr"
	"sync"
)

type BWTCodec struct{}

type bwtWorkspace struct {
	sa      []uint32 // suffix array indexes to be sorted
	tmpsa   []uint32 // next iterations of radix sorted suffix array indexes
	rank    []uint32 // ranking of suffix array indexes
	tmpRank []uint32 // next iteration of sorted rankings of suffix array indexes
	rankIdx []uint32 // scratch index list for second key sort
	count   []uint32 // scratch count sort slice to not re-allocate
	pos     []uint32 // scratch count sort slice for byte values
	next    []uint32
}

var bwtPool = sync.Pool{
	// creates a new workspace
	New: func() any {
		return &bwtWorkspace{}
	},
}

func initializeRank(s []uint8, pos, rank, sa []uint32) uint32 {
	// initialize ranks of rotations
	// ranks are decided by count-sorting based on first character
	var (
		b byte
		i uint32
		r uint32
	)
	// perform count-sort
	byteHistogram(s, pos)
	cumSum(pos)
	for i = range uint32(len(s)) {
		b = s[i]       // for each letter
		sa[pos[b]] = i // place suffix start index i into SA bucket for byte b
		pos[b]++       // increment index of b in SA for stability
	}
	// assign new compact ranks (compress equal keys)
	rank[sa[0]] = 0
	r = 1
	for i = 1; i < uint32(len(s)); i++ {
		if s[sa[i]] != s[sa[i-1]] {
			r++
		}
		rank[sa[i]] = r - 1
	}
	return r
}

func sortBySecondKey(inSA, outSA, rank, rankIdx, count []uint32, k uint32) {
	// perform count-sort of SA based on ranks of second half of prefix for each suffix
	var (
		key    uint32
		length = uint32(len(inSA))
		i      int
	)
	// wipe your histogram
	wipeSlice(count)
	// get the rank indexes to use in makeing the histogram and performing the count-sort
	for i = range len(rankIdx) {
		rankIdx[i] = inSA[i] + k
		if rankIdx[i] >= length { // fast modulus
			rankIdx[i] -= length
		}
	}
	// get histogram of ranks for second half of suffix prefix
	for i = range len(rankIdx) {
		key = rank[rankIdx[i]]
		count[key]++
	}
	// perform count-sort
	cumSum(count)
	for i = range len(inSA) {
		key = rank[rankIdx[i]]
		outSA[count[key]] = inSA[i]
		count[key]++
	}
}

func sortByFirstKey(inSA, outSA, rank, count []uint32) {
	// perform count-sort of SA based on ranks of first half of prefix for each suffix
	var (
		key uint32
		i   int
	)
	// wipe your histogram
	wipeSlice(count)
	// get histogram of ranks for first half of suffix prefex
	for i = range len(inSA) {
		count[rank[inSA[i]]]++
	}
	// perform count-sort
	cumSum(count)
	for i = range len(inSA) {
		key = uint32(rank[inSA[i]])
		outSA[count[key]] = inSA[i]
		count[key]++
	}
}

func buildCircularSuffixArray(s []byte, pos, count, rank, tmpRank, rankIdx, sa, tmpsa []uint32) {
	// create and lexicographically sort a circular suffix array
	// sorting is done by a radix-sort on suffix prefix segments that grow by 2x per iteration
	// each radix element sort is done using a count-sort algorithm
	var (
		maxRank = initializeRank(s, pos, rank, sa) // highest rank achieved per sort
		length  = uint32(len(s))
		newRank uint32
		prev    uint32
		cur     uint32
		prevA   uint32
		curA    uint32
		prevB   uint32
		curB    uint32
		i       uint32
	)
	// iteratively sort until either:
	// 1. you reach the max length of the suffixes (k == length)
	// 2. every suffix has a unique rank (maxRank == length)
	for k := uint32(1); k < length && maxRank < length; k *= 2 {
		// perform radix sort on second and first suffix prefix
		sortBySecondKey(sa, tmpsa, rank, rankIdx, count[:maxRank], k)
		sa, tmpsa = tmpsa, sa
		sortByFirstKey(sa, tmpsa, rank, count[:maxRank])
		sa, tmpsa = tmpsa, sa
		// determine new ranks of suffix array elements
		tmpRank[sa[0]] = 0
		newRank = 1
		for i = 1; i < length; i++ {
			prev = sa[i-1]     // previous element
			cur = sa[i]        // current element
			prevA = rank[prev] // previous first key element ranking
			curA = rank[cur]   // current first ke element ranking
			prev += k
			if prev >= length {
				prev -= length // fast modulus
			}
			prevB = rank[prev] // previous second key element ranking
			if cur+k >= length {
				curB = rank[(cur+k)-length]
			} else {
				curB = rank[cur+k] // current second key element ranking
			}
			// count unique ranks
			if (prevA != curA) || (prevB != curB) {
				newRank += 1
			}
			tmpRank[cur] = newRank - 1
		}
		rank, tmpRank = tmpRank, rank
		maxRank = newRank
	}
}

func (BWTCodec) EncodeBlock(src []byte) ([]byte, error) {
	// encode src using the burrows-wheeler transform (BWT)
	if len(src) == 0 {
		return src, nil
	}
	var (
		out            = make([]byte, len(src), len(src)+4) // output
		primary uint32 = 0                                  // row of unrotated data in sorted suffix array
		length  uint32 = uint32(len(src))                   // length of input
		p       uint32                                      // index of suffix
		prev    uint32                                      // index of last element in rotation
	)
	// instantiate the workspace
	ws := bwtPool.Get().(*bwtWorkspace)
	defer bwtPool.Put(ws)
	ws.sa = grow32(ws.sa, len(src))
	ws.tmpsa = grow32(ws.tmpsa, len(src))
	ws.rank = grow32(ws.rank, len(src))
	ws.tmpRank = grow32(ws.tmpRank, len(src))
	ws.rankIdx = grow32(ws.rankIdx, len(src))
	ws.count = grow32(ws.count, len(src))
	ws.pos = grow32(ws.pos, 256)
	// build the sorted circular suffix array
	buildCircularSuffixArray(src, ws.pos, ws.count, ws.rank, ws.tmpRank, ws.rankIdx, ws.sa, ws.tmpsa)
	// loop through each sorted circular suffix and grab the last element
	for i := range length {
		p = ws.sa[i]
		if p == 0 {
			prev = length - 1
		} else {
			prev = p - 1
		}
		out[i] = src[prev]
		if p == 0 {
			primary = uint32(i)
		}
	}
	// save off the 4-byte big-endian primary to the tail
	out = binary.BigEndian.AppendUint32(out, primary)
	return out, nil
}

func (BWTCodec) DecodeBlock(src []byte) ([]byte, error) {
	// decode src using the inverse burrows-wheeler transform
	if len(src) == 0 {
		return src, nil
	}
	// instantiate the workspace
	ws := bwtPool.Get().(*bwtWorkspace)
	defer bwtPool.Put(ws)
	ws.next = grow32(ws.next, len(src)) // last column to first column mapping array
	ws.pos = grow32(ws.pos, 256)        // count sort slice
	// check for primary value
	if len(src) < 4 {
		return []byte{}, sqerr.New(sqerr.Corrupt, "BWT is missing primary value")
	}
	// grab the primary value and chop it off the source
	primary := uint32(binary.BigEndian.Uint32(src[len(src)-4:]))
	src = src[:len(src)-4]
	if primary >= uint32(len(src)) {
		return []byte{}, sqerr.New(sqerr.Corrupt, "Primary BWT value is too large")
	}
	// build the count array (cumulative sum)
	byteHistogram(src, ws.pos)
	cumSum(ws.pos)
	var (
		out    = make([]byte, len(src)) // make an output slice
		length = uint32(len(src))
		i      uint32
		b      byte
	)
	// build stable count mapping for last column to first column
	for i = range length {
		b = src[i]
		ws.next[ws.pos[b]] = i
		ws.pos[b]++
	}
	// traverse last-first mapping starting with primary
	for i = range length {
		primary = ws.next[primary]
		out[i] = src[primary]
	}
	return out, nil
}

func (BWTCodec) IsLossless() bool {
	return true
}
