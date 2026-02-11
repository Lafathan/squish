package codec

import (
	"encoding/binary"
	"squish/internal/sqerr"
)

type BWTCodec struct{}

func histogram(bytes []byte) []int {
	var (
		hist = make([]int, 256)
	)
	for i := range len(bytes) {
		hist[bytes[i]]++
	}
	return hist
}

func cumSum(bytes []int) {
	var (
		sum = 0
		val int
	)
	for i := range len(bytes) {
		val = bytes[i]
		bytes[i] = sum
		sum += val
	}
}

func initializeRank(s []uint8, rank, sa []int) int {
	var (
		b byte
		i int
	)
	count := histogram(s)  // get histogram
	cumSum(count)          // get cumulative sum
	for i = range len(s) { // build count-sorted array
		b = s[i]         // for each letter
		sa[count[b]] = i // place suffix start index i into SA bucket for byte b
		count[b]++       // increment index for stability
	}
	rank[sa[0]] = 0
	r := 1
	for i = 1; i < len(s); i++ {
		if s[sa[i]] != s[sa[i-1]] {
			r++
		}
		rank[sa[i]] = r - 1
	}
	return r
}

func sortBySecondKey(inSA, outSA, rank []int, k, r int) {
	var (
		count = make([]int, r+2)
		key   int
		i     int
		j     int
	)
	for i = range len(inSA) { // get histogram of ranks for second half of suffix prefix
		j = inSA[i] + k
		key = 0
		if j < len(inSA) {
			key = rank[j] + 1
		}
		count[key]++
	}
	cumSum(count)             // get cumulative sum of histogram
	for i = range len(inSA) { // count-sort suffix array by ranks
		j = inSA[i] + k
		key = 0
		if j < len(inSA) {
			key = rank[j] + 1
		}
		outSA[count[key]] = inSA[i]
		count[key]++
	}
}

func sortByFirstKey(inSA, outSA, rank []int, r int) {
	var (
		count = make([]int, r+1)
		key   int
		i     int
	)
	for i = range len(inSA) { // get histogram of ranks for first half of suffix prefex
		count[rank[inSA[i]]]++
	}
	cumSum(count)             // get cumulative sum of histogram
	for i = range len(inSA) { // count-sort suffix array by ranks
		key = int(rank[inSA[i]])
		outSA[count[key]] = inSA[i]
		count[key]++
	}
}

func buildSuffixArray(s []byte) []int {
	var (
		sa      = make([]int, len(s))         // suffix array indexes
		tmpsa   = make([]int, len(s))         // next iterations of radix sorted suffix array indexes
		rank    = make([]int, len(s))         // sorted ranking of suffix array indexes
		tmpRank = make([]int, len(s))         // next iteration of sorted ranking of suffix array indexes
		maxRank = initializeRank(s, rank, sa) // highest rank achieved per sort
		newr    int                           // next iteration highest rank achieved
		prev    int                           // temp suffix
		cur     int                           // temp suffix
		prevA   int                           // temp ranks
		curA    int                           // temp ranks
		prevB   int                           // temp ranks
		curB    int                           // temp ranks
		k       = 1                           // suffix prefix length
		i       int                           // iterator variable
	)
	for k < len(s) && maxRank < len(s) {
		sortBySecondKey(sa, tmpsa, rank, k, maxRank) // radix sort suffix array by second key rank[i + k]
		sa, tmpsa = tmpsa, sa                        // save it off
		sortByFirstKey(sa, tmpsa, rank, maxRank)     // radix sort suffix array by first key rank[i]
		sa, tmpsa = tmpsa, sa                        // save it off
		tmpRank[sa[0]] = 0
		newr = 1
		for i = 1; i < len(s); i++ { // loop through the suffixes
			prev = sa[i-1]     // previous element
			cur = sa[i]        // current element
			prevA = rank[prev] // previous element ranking
			curA = rank[cur]   // current element ranking
			prevB = -1         // assume second element of radix sort key is past input length
			curB = -1
			if prev+k < len(s) { // if it is not too long
				prevB = rank[prev+k] // grab the ranking of the second prefix chunk
			}
			if cur+k < len(s) { // if it is not too long
				curB = rank[cur+k] // grab the ranking of the second prefix chunk
			}
			if (prevA != curA) || (prevB != curB) { // if they are not equal in rank
				newr += 1 // new max rank is increased
			}
			tmpRank[cur] = newr - 1
		}
		rank, tmpRank = tmpRank, rank // save off the newly calculated ranks
		maxRank = newr                // save off the new max rank
		k *= 2                        // double the suffix prefix length
	}
	return sa
}

func (BWTCodec) EncodeBlock(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return src, nil
	}
	var (
		outBytes        = make([]byte, len(src), len(src)+8)
		primary  uint64 = 0 // row of original/unrotated data in sorted suffix array
		sa              = buildSuffixArray(src)
		p        int
	)
	for i := range len(src) {
		p = sa[i] // get the current suffix
		if p == 0 {
			outBytes[i] = src[len(src)-1] // element to add is index of suffix array - 1 (wrap around)
			primary = uint64(i)           // if you are at 0 in SA (whole input) you found your primary index
		} else {
			outBytes[i] = src[p-1] // element to add is index of suffix array - 1
		}
	}
	outBytes = binary.BigEndian.AppendUint64(outBytes, primary) // save the 8 byte big-endian primary index to the tail of your data
	return outBytes, nil
}

func (BWTCodec) DecodeBlock(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return src, nil
	}
	primary := int(binary.BigEndian.Uint64(src[len(src)-8:])) // decode the primary value
	src = src[:len(src)-8]                                    // chop off the primary value
	if primary >= len(src) {
		return []byte{}, sqerr.New(sqerr.Corrupt, "Primary BWT value is too large")
	}
	count := histogram(src) // get the histogram
	cumSum(count)           // get the cumulative sum (prefix sums)
	var (
		outBytes = make([]byte, len(src)) // make an output slice
		seen     = make([]int, 256)       // helper for counting element occurrences
		occ      = make([]int, len(src))  // previous occurrence count for elements
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
