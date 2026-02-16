package codec

import (
	"encoding/binary"
	"squish/internal/sqerr"
)

type BWTCodec struct{}

func histogram(bytes []byte) []uint32 {
	var (
		hist = make([]uint32, 256)
	)
	for i := range len(bytes) {
		hist[bytes[i]]++
	}
	return hist
}

func cumSum(bytes []uint32) {
	var (
		sum uint32 = 0
		val uint32
	)
	for i := range len(bytes) {
		val = bytes[i]
		bytes[i] = sum
		sum += val
	}
}

func initializeRank(s []uint8, rank, sa []uint32) uint32 {
	var (
		b byte
		i uint32
		r uint32
	)
	count := histogram(s)          // get histogram
	cumSum(count)                  // get cumulative sum
	for i = range uint32(len(s)) { // build count-sorted array
		b = s[i]         // for each letter
		sa[count[b]] = i // place suffix start index i into SA bucket for byte b
		count[b]++       // increment index for stability
	}
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

func sortBySecondKey(inSA, outSA, rank []uint32, k uint32, count []uint32) {
	var (
		key    uint32
		length = uint32(len(inSA))
		i      int
		j      uint32
	)
	for i = range len(count) {
		count[i] = 0 // wipe your histogram
	}
	for i = range len(inSA) { // get histogram of ranks for second half of suffix prefix
		j = inSA[i] + k
		if j >= length {
			j -= length // fast modulus
		}
		key = rank[j]
		count[key]++
	}
	cumSum(count)             // get cumulative sum of histogram
	for i = range len(inSA) { // count-sort suffix array by ranks
		j = inSA[i] + k
		if j >= length {
			j -= length // fast modulus
		}
		key = rank[j]
		outSA[count[key]] = inSA[i]
		count[key]++
	}
}

func sortByFirstKey(inSA, outSA, rank []uint32, count []uint32) {
	var (
		key uint32
		i   int
	)
	for i = range len(count) {
		count[i] = 0 // wipe your histogram
	}
	for i = range len(inSA) { // get histogram of ranks for first half of suffix prefex
		count[rank[inSA[i]]]++
	}
	cumSum(count)             // get cumulative sum of histogram
	for i = range len(inSA) { // count-sort suffix array by ranks
		key = uint32(rank[inSA[i]])
		outSA[count[key]] = inSA[i]
		count[key]++
	}
}

func buildCircularSuffixArray(s []byte) []uint32 {
	var (
		count   = make([]uint32, len(s))      // scratch count sort slice to not re-allocate
		sa      = make([]uint32, len(s))      // suffix array indexes
		tmpsa   = make([]uint32, len(s))      // next iterations of radix sorted suffix array indexes
		rank    = make([]uint32, len(s))      // sorted ranking of suffix array indexes
		tmpRank = make([]uint32, len(s))      // next iteration of sorted ranking of suffix array indexes
		maxRank = initializeRank(s, rank, sa) // highest rank achieved per sort
		length  = uint32(len(s))              // length of input
		newr    uint32                        // next iteration highest rank achieved
		prev    uint32                        // temp suffix
		cur     uint32                        // temp suffix
		prevA   uint32                        // temp ranks
		curA    uint32                        // temp ranks
		prevB   uint32                        // temp ranks
		curB    uint32                        // temp ranks
		i       uint32                        // iterator variable
	)
	for k := uint32(1); k < length && maxRank < length; k *= 2 {
		sortBySecondKey(sa, tmpsa, rank, k, count[:maxRank+1]) // radix sort suffix array by second key rank[i + k]
		sa, tmpsa = tmpsa, sa                                  // save it off
		sortByFirstKey(sa, tmpsa, rank, count[:maxRank+1])     // radix sort suffix array by first key rank[i]
		sa, tmpsa = tmpsa, sa                                  // save it off
		tmpRank[sa[0]] = 0
		newr = 1
		for i = 1; i < length; i++ { // loop through the suffixes
			prev = sa[i-1]     // previous element
			cur = sa[i]        // current element
			prevA = rank[prev] // previous element ranking
			curA = rank[cur]   // current element ranking
			prev += k
			if prev >= length {
				prev -= length // fast modulus
			}
			prevB = rank[prev]
			if cur+k >= length {
				curB = rank[(cur+k)-length]
			} else {
				curB = rank[cur+k]
			}
			if (prevA != curA) || (prevB != curB) { // if they are not equal in rank
				newr += 1 // new max rank is increased
			}
			tmpRank[cur] = newr - 1
		}
		rank, tmpRank = tmpRank, rank // save off the newly calculated ranks
		maxRank = newr                // save off the new max rank
	}
	return sa
}

func (BWTCodec) EncodeBlock(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return src, nil
	}
	var (
		outBytes        = make([]byte, len(src), len(src)+8)
		primary  uint32 = 0 // row of original/unrotated data in sorted suffix array
		sa              = buildCircularSuffixArray(src)
		p        uint32
		prev     uint32
	)
	for i := range len(src) {
		p = sa[i] // get the current suffix
		if p == 0 {
			prev = uint32(len(src)) - 1
		} else {
			prev = p - 1
		}
		outBytes[i] = src[prev] // save the element prior to the start of that suffix
		if p == 0 {
			primary = uint32(i) // if you are at 0 in SA (whole input) you found your primary index
		}
	}
	outBytes = binary.BigEndian.AppendUint32(outBytes, primary) // save 4 byte big-endian primary to tail of data
	return outBytes, nil
}

func (BWTCodec) DecodeBlock(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return src, nil
	}
	primary := uint32(binary.BigEndian.Uint32(src[len(src)-4:])) // decode the primary value
	src = src[:len(src)-4]                                       // chop off the primary value
	if primary >= uint32(len(src)) {
		return []byte{}, sqerr.New(sqerr.Corrupt, "Primary BWT value is too large")
	}
	count := histogram(src) // get the histogram
	cumSum(count)           // get the cumulative sum (prefix sums)
	var (
		outBytes = make([]byte, len(src))   // make an output slice
		seen     = make([]uint32, 256)      // helper for counting element occurrences
		occ      = make([]uint32, len(src)) // previous occurrence count for elements
		i        int
		b        byte
	)
	for i = range len(occ) { // this keep track of how many times we've seen each element so far
		b = src[i] // essentially generate a running histogram
		occ[i] = seen[b]
		seen[b]++
	}
	for i = len(src) - 1; i >= 0; i-- { // start at the primary row
		b = src[primary]                  // get the current byte
		outBytes[i] = b                   // build it into your output
		primary = count[b] + occ[primary] // step back one rotation
	}
	return outBytes, nil
}

func (BWTCodec) IsLossless() bool {
	return true
}
