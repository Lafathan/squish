package codec

import (
	"encoding/binary"
	"squish/internal/sqerr"
	"sync"
)

type BWT2Codec struct{}

type bwt2Workspace struct {
	src  []int32 // int32 cast of src input
	sa   []int32 // suffix array indexes to be sorted
	tmp  []int32 // temp storage for scratch space
	bktH []int32 // bucket min indexes
	bktT []int32 // bucket max indexes
	pos  []int32 // scratch count sort slice for byte values
	next []int32 // last-first mapping for decoding
}

var bwt2Pool = sync.Pool{
	// creates a new workspace
	New: func() any {
		return &bwt2Workspace{}
	},
}

func sais(data, sa, tmp, bktH, bktT []int32, dataMax int) {
	if len(data) == 0 {
		return
	}
	if len(data) == 1 {
		sa[0] = 0
		return
	}
	clear(tmp)
	// freq, lengths := tmp[:len(tmp)/2], tmp[len(tmp)/2:]
	freq := make([]int32, dataMax)
	lengths := make([]int32, 2*len(sa))
	// data value frequency and bucket-sort offsets
	getBuckets(data, freq, bktH, bktT, dataMax)
	// fmt.Printf("Data:         %v\n", data)
	numLMS := placeLMS(data, sa, bktT, lengths)
	// fmt.Printf("Place LMS:    %v\n", sa)
	// fmt.Printf("NumLMS:       %v\n", numLMS)
	// reset buckets for induction sort
	getBuckets(data, freq, bktH, bktT, dataMax)
	if numLMS > 1 {
		// perform induction sort
		induceLeft(data, sa, bktH)
		// fmt.Printf("Induce Left:  %v\n", sa)
		induceRight(data, sa, bktT)
		// fmt.Printf("Induce Right: %v\n", sa)
		// create sub-problem ids
		maxID := createIDs(data, sa, lengths)
		// map them to create the sub-problem
		mapID(sa, lengths)
		// generate sa for sub-problem
		sais(sa[:numLMS], sa[len(sa)-numLMS:], freq, bktH, bktT, maxID)
		// reset buckets for placing the newly sorted LMS substring suffixes
		getBuckets(data, freq, bktH, bktT, dataMax)
		// unmap ids to LMS substring idexes
		unmapID(data, sa, lengths, bktT, numLMS)
	}
	// reset buckets for induction again
	getBuckets(data, tmp, bktH, bktT, dataMax)
	// perform induction sort
	// fmt.Printf("New SA:       %v\n", sa)
	induceLeft(data, sa, bktH)
	// fmt.Printf("Induce Left:  %v\n", sa)
	induceRight(data, sa, bktT)
	// fmt.Printf("Induce Right: %v\n", sa)
}

func getBuckets(data, tmp, bktH, bktT []int32, dataMax int) {
	freq := grow32(tmp, max(256, dataMax))
	histogram(data, freq)
	total := int32(0)
	for i, n := range freq {
		bktH[i] = total
		total += n
		bktT[i] = total
	}
}

func placeLMS(data, sa, bktT, lengths []int32) int {
	clear(lengths)
	numLMS := 0
	// determine LMS status on the fly
	c0, c1, isTypeS, prevLMS := data[len(data)-1], int32(0), false, int32(len(data))-1
	// scan backwards placing LMS indexes in the tails of their buckets
	for i := int32(len(data) - 2); i >= 0; i-- {
		c0, c1 = data[i], c0
		if c0 < c1 {
			isTypeS = true
		} else if c0 > c1 && isTypeS {
			isTypeS = false
			// decrement the tail of the bucket
			b := bktT[c1] - 1
			bktT[c1] = b
			// put the LMS index into the suffix array
			sa[b] = int32(i + 1)
			// count the LMS substrings placed
			numLMS++
			// store the length of the LMS substring
			lengths[i+1] = prevLMS - i + 1
			prevLMS = i
		}
	}
	// if data[0] < data[1] {
	// 	// decrement the tail of the bucket
	// 	b := bktT[data[0]] - 1
	// 	bktT[data[0]] = b
	// 	// put the LMS index into the suffix array
	// 	sa[b] = 0
	// 	// count the LMS substrings placed
	// 	numLMS++
	// 	// store the length of the LMS substring
	// 	lengths[0] = prevLMS + 1
	//
	// }
	return numLMS
}

func createIDs(data, sa, lengths []int32) int {
	id := int32(0)
	lastLen := int32(-1)
	lastPos := int32(0)
	// loop through the sorted sa
	for _, j := range sa {
		// if length at that index is 0, it is not an LMS substring index
		if lengths[j] == 0 {
			continue
		}
		// get the start and length of the LMS substring
		curLen := lengths[j]
		curPos := j
		// protect the last LMS string from reaching beyond the length of the input
		if curPos+curLen >= int32(len(sa)) {
			curLen--
		}
		// they obviously don't match if they are different lengths
		if curLen != lastLen {
			// increment the id and update the last LMS substring compared to
			id++
			lastPos = curPos
			lastLen = curLen
		} else {
			// compare the data
			this := data[curPos:][:curLen]
			last := data[lastPos:][:lastLen]
			for k := range curLen {
				if this[k] != last[k] {
					// increment the id and update the last LMS substring compared to
					id++
					lastPos = curPos
					lastLen = curLen
				}
			}
		}
		// replace the length with the id
		lengths[j] = id
	}
	return int(id)
}

func induceLeft(data, sa, bktH []int32) {
	k := len(data) - 1
	// cache the recently used bucket
	cachedChar := data[k]
	cachedBucket := bktH[cachedChar]
	sa[cachedBucket] = int32(k)
	cachedBucket++
	// loop through the sa placing l-type indexes
	for i := range len(sa) {
		j := sa[i]
		if j <= 0 {
			continue
		}
		k := j - 1
		currentChar := data[k]
		if k >= 0 {
			if currentChar < data[j] {
				// skip s-type indexes
				continue
			}
		}
		// cache the new bucket
		if currentChar != cachedChar {
			bktH[cachedChar] = cachedBucket
			cachedChar = currentChar
			cachedBucket = bktH[cachedChar]
		}
		// place induced l-type index into array
		sa[cachedBucket] = k
		// increment bucket head
		cachedBucket++
	}
}

func induceRight(data, sa, bktT []int32) {
	// cache the recently used bucket
	cachedChar := data[0]
	cachedBucket := bktT[cachedChar]
	// loop backwards through the sa placing s-type indexes
	for i := len(sa) - 1; i >= 0; i-- {
		j := sa[i]
		if j <= 0 {
			continue
		}
		k := j - 1
		currentChar := data[k]
		if k >= 0 {
			if currentChar > data[j] {
				// skip l-type indexes
				continue
			}
		}
		// cache the new bucket
		if currentChar != cachedChar {
			bktT[cachedChar] = cachedBucket
			cachedChar = currentChar
			cachedBucket = bktT[cachedChar]
		}
		// decrement bucket tail
		cachedBucket--
		// place induced s-type index into array
		sa[cachedBucket] = k
	}
}

func mapID(sa, ids []int32) {
	idx := 0
	// loop through sa
	for i := range sa {
		sa[i] = 0
		if ids[i] > 0 {
			// put the sorted LMS substring indexes at the tail end of ids for unmapping
			ids[len(ids)-1-idx] = int32(i)
			// put the mapped IDs in the head of the sa for sub-problem processing
			sa[idx] = ids[i] - 1
			idx++
		}
	}
}

func unmapID(data, sa, ids, bktT []int32, numLMS int) {
	// copy(ids[:numLMS+1], sa[len(sa)-numLMS:])
	copy(ids[:numLMS], sa[len(sa)-numLMS:])
	clear(sa)
	end := len(ids) - 1
	// cache recently used bucket
	cachedChar := ids[end-numLMS+int(ids[0])]
	cachedBucket := bktT[cachedChar]
	// loop through the new sub-problem sa
	for i := numLMS - 1; i >= 0; i-- {
		// get the current LMS and character
		current_LMS := ids[end-int(ids[i])]
		current_char := data[current_LMS]
		if current_char != cachedChar {
			// cache the new bucket
			bktT[cachedChar] = cachedBucket
			cachedChar = current_char
			cachedBucket = bktT[cachedChar]
		}
		// decrement bucket tail
		cachedBucket--
		// place the sub-problem induction sorted LMS indexes
		sa[cachedBucket] = current_LMS
	}
}

func (BWT2Codec) EncodeBlock(src []byte) ([]byte, error) {
	// encode src using the burrows-wheeler transform (BWT)
	if len(src) == 0 {
		return src, nil
	}
	var (
		length         = int32(len(src))
		out            = make([]byte, length, length+4) // output
		primary uint32 = 0                              // row of unrotated data in sorted suffix array
	)
	// instantiate the workspace
	ws := bwt2Pool.Get().(*bwt2Workspace)
	defer bwt2Pool.Put(ws)
	ws.src = grow32(ws.src, int(2*length))
	ws.sa = grow32(ws.sa, int(2*length))
	s := max(256, int(length))
	ws.tmp = grow32(ws.tmp, 4*s)
	ws.bktH = grow32(ws.bktH, s)
	ws.bktT = grow32(ws.bktT, s)
	// double the input to create a cyclic sa
	for i := range len(src) {
		v := int32(src[i])
		ws.src[i] = v
		ws.src[i+len(src)] = v
	}
	// build the suffix array using induced sorting
	sais(ws.src, ws.sa, ws.tmp, ws.bktH, ws.bktT, 256)
	// extract valid rotations
	k := int32(0)
	for _, p := range ws.sa {
		if p >= length {
			continue
		}
		prev := p - 1
		if prev < 0 {
			prev = int32(length - 1)
		}
		out[k] = src[prev]
		if p == 0 {
			primary = uint32(k)
		}
		k++
		if k == length {
			break
		}
	}
	// save off the 4-byte big-endian primary to the tail
	out = binary.BigEndian.AppendUint32(out, primary)
	return out, nil
}

func (BWT2Codec) DecodeBlock(src []byte) ([]byte, error) {
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
	length := int32(len(src) - 4)
	primary := int32(binary.BigEndian.Uint32(src[length:]))
	src = src[:length]
	if primary >= length {
		return []byte{}, sqerr.New(sqerr.Corrupt, "Primary BWT value is too large")
	}
	// build the count array (cumulative sum)
	histogram(src, ws.pos)
	cumSum(ws.pos)
	var (
		out = make([]byte, length) // make an output slice
	)
	// build stable count mapping for last column to first column
	for i := range length {
		b := src[i]
		ws.next[ws.pos[b]] = i
		ws.pos[b]++
	}
	// traverse last-first mapping starting with primary
	for i := range length {
		primary = ws.next[primary]
		out[i] = src[primary]
	}
	return out, nil
}

func (BWT2Codec) IsLossless() bool {
	return true
}
