package codec

// ======================================================================================
// Huffman Encoding
//
// This codec replaces bytes that occur more often with smaller bit representations.
//
// ex: [0 0 0 0 0 0 0 0 1 1 1 1 2 2 3 3]
//                            __________________
//                           |                  |
//                           | 0000 0000 -> 0   |
//     byte to bit mapping = | 0000 0001 -> 10  |
//                           | 0000 0010 -> 110 |
//                           | 0000 0011 -> 111 |
//                           |__________________|
//
//     [0 0 0 0 0 0 0 0 1 1 1 1 2 2 3 3] -> [0000000010101010110110111111]
//                                          [0000 0000 1010 1010 1101 1011 1111 0000]
//                                          [        0       170       219       240]
//
//     [0 0 0 0 0 0 0 0 1 1 1 1 2 2 3 3] -> [0 170 219 240]
// ======================================================================================

import (
	"bytes"
	"container/heap"
	"errors"
	"io"
	"math/big"
	"squish/internal/bitio"
	"squish/internal/sqerr"
)

const (
	leaf   = 0
	branch = 1
)

type HUFFMANCodec struct{}

type node struct {
	nodeType  int      // 0 is leaf, 1 is a node
	frequency int32    // frequency of value, or sum of frequencies of children
	children  [2]*node // children if not a leaf
	value     byte
}

type hCode struct {
	bits   *big.Int // byte array to store bits of huffman code
	length int      // valid number of bits in byte array
}

type huffmanHeap []*node

// define function required to implement the heap interface
func (h huffmanHeap) Len() int           { return len(h) }
func (h huffmanHeap) Less(i, j int) bool { return h[i].frequency < h[j].frequency }
func (h huffmanHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *huffmanHeap) Push(n any)        { *h = append(*h, n.(*node)) }
func (h *huffmanHeap) Pop() any {
	n := len(*h)
	x := (*h)[n-1]
	*h = (*h)[0 : n-1]
	return x
}

func getHuffmanTreeFromFreqMap(freqMap []int32) *node {
	// build a huffman tree of nodes from a frequency map using a heap
	// left node is always smallest, right node is second smallest
	var (
		leaves = &huffmanHeap{}
		left   *node
		right  *node
	)
	// build heap of nodes
	heap.Init(leaves)
	for i := range len(freqMap) {
		if freqMap[i] > 0 {
			heap.Push(leaves, &node{nodeType: leaf, value: byte(i), frequency: freqMap[i]})
		}
	}
	// build tree of nodes
	// pop two smallest nodes off heap -> combine into parent node -> add parent back to heap -> repeat
	for leaves.Len() > 1 {
		left = heap.Pop(leaves).(*node)
		right = heap.Pop(leaves).(*node)
		newNode := node{
			nodeType:  branch,
			frequency: left.frequency + right.frequency,
			children:  [2]*node{left, right},
		}
		heap.Push(leaves, &newNode)
	}
	return heap.Pop(leaves).(*node)
}

func getHuffmanLengthsFromTree(tree *node) *[256]uint8 {
	// get huffman code lengths from huffman tree
	var (
		lengths = [256]uint8{} // store the bit lengths for each symbol at the index of that symbol
		getCode func(n *node, l uint8)
	)
	// recursive depth-first search for determining code lengths
	getCode = func(n *node, l uint8) {
		if n.nodeType == leaf {
			if l == 0 {
				// protection from single symbol tree
				l = 1
			}
			// update the list of lengths when you arrive at a leaf
			lengths[n.value] = l
		} else {
			// recurse for children
			getCode(n.children[0], l+1)
			getCode(n.children[1], l+1)
		}
	}
	getCode(tree, 0)
	return &lengths
}

func getHuffmanTreeFromDict(d *[256]hCode) *node {
	// build a huffman tree from a map of byte values to huffman codes
	var (
		root      = node{nodeType: branch}                           // make an empty root node
		buildTree func(n *node, val byte, bits *big.Int, bitPos int) // define recursive elements
		bit       uint
	)
	// recursively build out the tree as you iterate through given codes
	buildTree = func(n *node, val byte, bits *big.Int, bitPos int) {
		if bitPos >= 0 {
			// if you are not at the end of your bit stream
			bit = bits.Bit(bitPos)
			if n.children[bit] == nil {
				// create children nodes when needed
				n.children[bit] = &node{nodeType: branch}
			}
			// continue by recursing into child nodes
			buildTree(n.children[bit], val, bits, bitPos-1)
		} else {
			// if you are at the end of your bit stream, you are at a leaf
			n.nodeType = leaf
			n.value = val
		}
	}
	// build the tree using each given code
	for i := range len(d) {
		if d[i].length > 0 {
			buildTree(&root, byte(i), d[i].bits, d[i].length-1)
		}
	}
	return &root
}

func getHuffmanDictFromLengths(l *[256]uint8) *[256]hCode {
	// build the canonical huffman codes from the code lengths
	var (
		d           = [256]hCode{}  // the dictionary to store the data in
		codeLengths = [256]int{}    // a code length histogram for skipping unnecessary loops
		curBits     = big.NewInt(0) // new big.Int to store bit streams
		one         = big.NewInt(1) // big.Int value of one for big.Int incrementing
	)
	for i := range len(l) {
		codeLengths[l[i]]++
	}
	for bitLen := 1; bitLen < 256; bitLen++ {
		// loop through all possible bit stream lengths
		if codeLengths[bitLen] > 0 {
			// loop through all codes
			for i := range len(l) {
				if l[i] == uint8(bitLen) {
					// build the canonical code for code of matching length
					d[i] = hCode{bits: big.NewInt(0).Set(curBits), length: bitLen}
					// add 1 to get the next canonical code
					curBits.Add(curBits, one)
				}
			}
		}
		// left shift to get the codes of the next size up
		curBits.Lsh(curBits, 1)
	}
	return &d
}

func serializeHuffmanLengths(l *[256]uint8) []byte {
	// serializes canonical Huffman codes by storing only the bit length and symbol
	out := []byte{}
	for bitLen := 1; bitLen < 256; bitLen++ {
		for i := range len(l) {
			if l[i] == uint8(bitLen) {
				// append the bit length and then the symbol
				out = append(out, byte(l[i]))
				out = append(out, byte(i))
			}
		}
	}
	// double zero bytes marks the end of the dictionary
	out = append(out, 0x00)
	out = append(out, 0x00)
	return out
}

func deserializeHuffmanLengths(br io.Reader) (*[256]uint8, error) {
	// read in canonical huffman codes from a reader and return an array mapping symbol byte to huffman code length
	var (
		lengths       = [256]uint8{}
		lenSymbolPair = make([]byte, 2) // 2 byte slice buffer to read into
	)
	for {
		// read in the length and the symbol
		if _, err := io.ReadFull(br, lenSymbolPair); err != nil {
			return &lengths, sqerr.CodedError(err, sqerr.Corrupt, "Error while reading huffman dictionary")
		}
		// zero length marks the end of the dictionary
		if lenSymbolPair[0] == 0x00 {
			break
		} else {
			lengths[lenSymbolPair[1]] = lenSymbolPair[0]
		}
	}
	return &lengths, nil
}

func (HUFFMANCodec) EncodeBlock(src []byte) ([]byte, error) {
	// encode src using canonical huffman encoding
	if len(src) == 0 {
		return src, nil
	}
	var (
		outBuffer     = new(bytes.Buffer)             // create a new buffer to write to
		bw            = bitio.NewBitWriter(outBuffer) // make a new bitwriter
		tmpBig        = big.NewInt(0)                 // big.Int for nibble of bit.Int
		remainingBits int                             // remaining bites to be written
		bitsToWrite   int                             // number of bits to write per pass per symbol
		f             = make([]int32, 256)
	)
	histogram(src, f)
	t := getHuffmanTreeFromFreqMap(f)
	l := getHuffmanLengthsFromTree(t)
	d := getHuffmanDictFromLengths(l)
	_, err := outBuffer.Write(serializeHuffmanLengths(l))
	for i := range len(src) {
		// iteratively write huffman code bits per symbol
		// iterating only really occurs for long symbol lengths (writes 64 bits at a time)
		remainingBits = d[src[i]].length
		for remainingBits > 0 {
			bitsToWrite = min(remainingBits, 64)
			tmpBig.Rsh(d[src[i]].bits, uint(remainingBits-bitsToWrite))
			if err = bw.WriteBits(tmpBig.Uint64(), bitsToWrite); err != nil {
				return nil, sqerr.CodedError(err, sqerr.IO, "Error while writing huffman encoded bits")
			}
			remainingBits -= bitsToWrite
		}
	}
	// flush and pad ouput to be byte aligned, padded bits is prepended to dictionary
	pad, err := bw.Flush()
	if err != nil {
		return nil, sqerr.CodedError(err, sqerr.IO, "Error while flushing bitwriter during huffman encoding")
	}
	out := append([]byte{byte(pad)}, outBuffer.Bytes()...)
	return out, nil
}

func (HUFFMANCodec) DecodeBlock(src []byte) ([]byte, error) {
	// decode src using canonical huffman encoding
	if len(src) == 0 {
		return src, nil
	}
	br := bytes.NewBuffer(src)
	// get the padded bits
	padBits, err := br.ReadByte()
	if err != nil {
		return nil, sqerr.CodedError(err, sqerr.IO, "error while reading padded bits byte in huffman decoding: %w")
	}
	// get the huffman code lengths
	l, err := deserializeHuffmanLengths(br)
	if err != nil {
		return nil, sqerr.CodedError(err, sqerr.Corrupt, "error while deserializing huffman code dictionary")
	}
	d := getHuffmanDictFromLengths(l)
	t := getHuffmanTreeFromDict(d)
	inBuffer := bitio.NewBitReader(br)
	var (
		outBuffer = make([]byte, 0, 4*len(src)) // 4 * src length is just an estimate
		padBuffer uint64
		newBit    uint64
	)
	// create a buffer to hold padded bits, so you don't accidentally process them as huffman codes
	// padBuffer acts as a buffer between what's read in and what's decoded
	// so when you get to the end of the stream, it will be holding all padding bits
	if padBits > 0 {
		padBuffer, err = inBuffer.ReadBits(int(padBits))
		if err != nil {
			return nil, sqerr.CodedError(err, sqerr.IO, "error reading padding bits from source in huffman decoding")
		}
	}
	node := t
	for {
		// read in bit by bit, traversing the tree
		if node.nodeType == branch {
			newBit, err = inBuffer.ReadBits(1)
			if err != nil {
				break
			}
			if padBits > 0 {
				// use the msb of the padded buffer as the decision bit
				node = node.children[(padBuffer>>(padBits-1))&0x01]
				// update the padBuffer with new bits
				padBuffer = (padBuffer << 1) | newBit
			} else {
				// use the new bit as the decision bit if there is no padding.
				node = node.children[newBit]
			}
		} else {
			// append the decoded byte when you get to a leaf and reset to the root node
			outBuffer = append(outBuffer, node.value)
			node = t
		}
	}
	if errors.Is(err, io.EOF) {
		return outBuffer, nil
	} else {
		return nil, sqerr.CodedError(err, sqerr.IO, "error reading bits from source in huffman decoding")
	}
}

func (HUFFMANCodec) IsLossless() bool {
	return true
}
