package codec

import (
	"encoding/binary"
	"squish/internal/sqerr"
	"sync"
)

type BWTCodec struct{}

type bwtWorkspace struct {
	sa      []uint32
	tmpsa   []uint32
	rank    []uint32
	tmpRank []uint32
	count   []uint32
	next    []uint32
	pos     []uint32
}

var bwtPool = sync.Pool{
	// creates a new workspace
	New: func() any {
		return &bwtWorkspace{}
	},
}

func grow32(slice []uint32, length int) []uint32 {
	// function to grow slices
	if cap(slice) < length {
		return make([]uint32, length)
	}
	return slice[:length]
}

func histogram(bytes []byte, count []uint32) {
	// build a byte histogram
	for i := range len(count) {
		count[i] = 0
	}
	for i := range len(bytes) {
		count[bytes[i]]++
	}
}

func cumSum(count []uint32) {
	// create cumulative sum
	var (
		sum uint32 = 0
		val uint32
	)
	for i := range len(count) {
		val = count[i]
		count[i] = sum
		sum += val
	}
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
	histogram(s, pos)
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

func sortBySecondKey(inSA, outSA, rank, count []uint32, k uint32) {
	// perform count-sort of SA based on ranks of second half of prefix for each suffix
	var (
		key    uint32
		length = uint32(len(inSA))
		i      int
		j      uint32
	)
	// wipe your histogram
	for i = range len(count) {
		count[i] = 0
	}
	// get histogram of ranks for second half of suffix prefix
	for i = range len(inSA) {
		j = inSA[i] + k
		if j >= length {
			j -= length // fast modulus
		}
		key = rank[j]
		count[key]++
	}
	// perform count-sort
	cumSum(count)
	for i = range len(inSA) {
		j = inSA[i] + k
		if j >= length {
			j -= length // fast modulus
		}
		key = rank[j]
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
	for i = range len(count) {
		count[i] = 0
	}
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

func buildCircularSuffixArray(s []byte, ws *bwtWorkspace) {
	// create and lexicographically sort a circular suffix array
	// sorting is done by a radix-sort on suffix prefix segments that grow by 2x per iteration
	// each radix element sort is done using a count-sort algorithm
	var (
		maxRank = initializeRank(s, ws.pos, ws.rank, ws.sa) // highest rank achieved per sort
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
		sortBySecondKey(ws.sa, ws.tmpsa, ws.rank, ws.count[:maxRank], k)
		ws.sa, ws.tmpsa = ws.tmpsa, ws.sa
		sortByFirstKey(ws.sa, ws.tmpsa, ws.rank, ws.count[:maxRank])
		ws.sa, ws.tmpsa = ws.tmpsa, ws.sa
		// determine new ranks of suffix array elements
		ws.tmpRank[ws.sa[0]] = 0
		newRank = 1
		for i = 1; i < length; i++ {
			prev = ws.sa[i-1]     // previous element
			cur = ws.sa[i]        // current element
			prevA = ws.rank[prev] // previous first key element ranking
			curA = ws.rank[cur]   // current first ke element ranking
			prev += k
			if prev >= length {
				prev -= length // fast modulus
			}
			prevB = ws.rank[prev] // previous second key element ranking
			if cur+k >= length {
				curB = ws.rank[(cur+k)-length]
			} else {
				curB = ws.rank[cur+k] // current second key element ranking
			}
			// count unique ranks
			if (prevA != curA) || (prevB != curB) {
				newRank += 1
			}
			ws.tmpRank[cur] = newRank - 1
		}
		ws.rank, ws.tmpRank = ws.tmpRank, ws.rank
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
	ws := bwtPool.Get().(*bwtWorkspace)
	defer bwtPool.Put(ws)
	ws.sa = grow32(ws.sa, len(src))           // suffix array indexes to be sorted
	ws.tmpsa = grow32(ws.tmpsa, len(src))     // next iterations of radix sorted suffix array indexes
	ws.rank = grow32(ws.rank, len(src))       // ranking of suffix array indexes
	ws.tmpRank = grow32(ws.tmpRank, len(src)) // next iteration of sorted rankings of suffix array indexes
	ws.count = grow32(ws.count, len(src))     // scratch count sort slice to not re-allocate
	ws.pos = grow32(ws.pos, 256)              // scratch count sort slice for byte values
	buildCircularSuffixArray(src, ws)         // sorted circular suffix array
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
	// instantiate your workspace
	ws := bwtPool.Get().(*bwtWorkspace)
	defer bwtPool.Put(ws)
	ws.pos = grow32(ws.pos, 256)
	ws.next = grow32(ws.next, len(src))
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
	histogram(src, ws.pos)
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
