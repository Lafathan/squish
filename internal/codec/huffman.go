package codec

import (
	"bytes"
	"container/heap"
	"errors"
	"fmt"
	"io"
	"math/big"
	"squish/internal/bitio"
)

const (
	leaf   = 0
	branch = 1
)

type HUFFMANCodec struct{}

type node struct {
	nodeType  int      // 0 is leaf, 1 is a node
	value     byte     // value held by a leaf
	frequency int      // frequency of value, or sum of frequencies of children
	children  [2]*node // children if not a leaf
}

type hCode struct {
	bits   *big.Int
	length int
}

type huffmanHeap []*node

// define function required to inherit the heap interface
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

func getFrequencyMap(src []byte) *[256]int {
	freqMap := [256]int{}
	for _, b := range src {
		freqMap[b]++
	}
	return &freqMap
}

func getHuffmanTreeFromFreqMap(freqMap *[256]int) *node {
	leaves := &huffmanHeap{}       // instantiate a heap
	heap.Init(leaves)              // initialize it
	for b, freq := range freqMap { // add nodes to the heap based on the freq map
		if freq > 0 {
			heap.Push(leaves, &node{nodeType: leaf, value: byte(b), frequency: freq})
		}
	}
	for leaves.Len() > 1 {
		l := heap.Pop(leaves).(*node) // get the smallest left child node
		r := heap.Pop(leaves).(*node) // get the second smalleset right child node
		newNode := node{              // create a new parent node for those children
			nodeType:  branch,
			frequency: l.frequency + r.frequency,
			children:  [2]*node{l, r},
		}
		heap.Push(leaves, &newNode) // push that new parent back on to the heap
	}
	return heap.Pop(leaves).(*node)
}

func getHuffmanLengthsFromTree(tree *node) *[256]uint8 {
	lengths := [256]uint8{}            // store the bit lengths for each symbol at the index of that symbol
	var getCode func(n *node, l uint8) // define a func for recursive depth first search
	getCode = func(n *node, l uint8) {
		if n.nodeType == leaf {
			if l == 0 {
				l = 1 // protection from single symbol tree
			}
			lengths[n.value] = l // update the list of lengths when you arrive at a leaf
		} else {
			getCode(n.children[0], l+1) // recurse for children
			getCode(n.children[1], l+1)
		}
	}
	getCode(tree, 0)
	return &lengths
}

func getHuffmanTreeFromDict(d *[256]hCode) *node {
	root := node{nodeType: branch} // make an empty root node
	var (
		buildTree func(n *node, val byte, bits *big.Int, bitPos int) // define recursive elements
		bit       uint
	)
	buildTree = func(n *node, val byte, bits *big.Int, bitPos int) {
		if bitPos >= 0 {
			bit = bits.Bit(bitPos)
			if n.children[bit] == nil {
				n.children[bit] = &node{nodeType: branch} // create the child node if it doesn't exist
			}
			buildTree(n.children[bit], val, bits, bitPos-1) // recurse into the child node
		} else {
			n.nodeType = leaf // if you are at the end of your bit stream, you are at a leaf
			n.value = val
		}
	}
	for k, v := range d {
		if v.length > 0 {
			buildTree(&root, byte(k), v.bits, v.length-1)
		}
	}
	return &root
}

func getHuffmanDictFromLengths(l *[256]uint8) *[256]hCode {
	// this functional builds the cononical huffman codes from the code lengths
	var (
		d           = [256]hCode{}  // the dictionary to store the data in
		codeLengths = [256]int{}    // a code length counter for skipping unnecessary loops
		curBits     = big.NewInt(0) // new big int to store bit streams
		one         = big.NewInt(1) // big.Int value of one for big.Int incrementing
		symbol      int             // current symbol
		length      uint8           // current length
	)
	for _, length := range l {
		codeLengths[length]++ // count the lengths
	}
	for bitLen := 1; bitLen < 256; bitLen++ { // loop through all possible lengths
		if codeLengths[bitLen] > 0 {
			for symbol, length = range l { // loop through the symbol - length pairs
				if length == uint8(bitLen) { // we only care about matching length symbols
					d[symbol] = hCode{bits: big.NewInt(0).Set(curBits), length: bitLen}
					curBits.Add(curBits, one)
				}
			}
		}
		curBits.Lsh(curBits, 1)
	}
	return &d
}

func serializeHuffmanLengths(l *[256]uint8) []byte {
	var (
		symbol int
		length uint8
	)
	// uses cononical huffman encodings by storing only the bit length and symbol
	out := []byte{}                           // make a place to store the output
	for bitLen := 1; bitLen < 256; bitLen++ { // loop through all possible lengths in increasing order
		for symbol, length = range l { // loop through the symbol array
			if length == uint8(bitLen) { // if the code length is right
				out = append(out, byte(length)) // append the bit length
				out = append(out, byte(symbol)) // and symbol to the output
			}
		}
	}
	out = append(out, 0x00)
	out = append(out, 0x00)
	return out
}

func deserializeHuffmanLengths(br io.Reader) (*[256]uint8, error) {
	var (
		lengths       = [256]uint8{}    // make an array to store all Huffman codes
		lenSymbolPair = make([]byte, 2) // make a 1 byte slice buffer to read into
		err           error
	)
	for {
		_, err = io.ReadFull(br, lenSymbolPair) // read in the bit length and symbol
		if err != nil {
			return &lengths, fmt.Errorf("error while reading huffman dictionary: %w", err)
		}
		if lenSymbolPair[0] == 0x00 {
			break // zero bit length marks the end of the dictionary
		} else {
			lengths[lenSymbolPair[1]] = lenSymbolPair[0]
		}
	}
	return &lengths, nil
}

func (HUFFMANCodec) EncodeBlock(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return []byte{}, nil
	}
	var (
		outBuffer     = new(bytes.Buffer)             // create a new buffer to write to
		bw            = bitio.NewBitWriter(outBuffer) // make a new bitwriter
		tmpBig        = big.NewInt(0)                 // big.Int for nibble of bit.Int
		remainingBits int                             // remaining bites to be written
		bitsToWrite   int                             // number of bits to writer per pass per symbol
	)
	f := getFrequencyMap(src)                             // get freq map
	t := getHuffmanTreeFromFreqMap(f)                     // build the tree
	l := getHuffmanLengthsFromTree(t)                     // get lengths of all of codes
	d := getHuffmanDictFromLengths(l)                     // build the canonical dictionary from the lengths
	_, err := outBuffer.Write(serializeHuffmanLengths(l)) // write the lengths to it
	for _, b := range src {                               // for symbol in src
		remainingBits = d[b].length // how many bits need written
		for remainingBits > 0 {
			bitsToWrite = min(remainingBits, 64)                   // determine how many bits (max 64 due to bitwriter)
			tmpBig.Rsh(d[b].bits, uint(remainingBits-bitsToWrite)) // shift it to get bits of interest in LSB
			err = bw.WriteBits(tmpBig.Uint64(), bitsToWrite)       // write it
			if err != nil {
				return []byte{}, fmt.Errorf("error while writing huffman encoded bits: %w", err)
			}
			remainingBits -= bitsToWrite // cound down the bits to be written
		}
	}
	pad, err := bw.Flush() // flush it and report back the number of pad bits
	if err != nil {
		return []byte{}, fmt.Errorf("error while flushing bitwriter during huffman encoding: %w", err)
	}
	out := append([]byte{byte(pad)}, outBuffer.Bytes()...)
	return out, nil
}

func (HUFFMANCodec) DecodeBlock(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return []byte{}, nil
	}
	br := bytes.NewBuffer(src)    // create a byte buffer for reading bytes
	padBits, err := br.ReadByte() // read in the padded bits byte
	if err != nil {
		return []byte{}, fmt.Errorf("error while reading padded bits byte in huffman decoding: %w", err)
	}
	l, err := deserializeHuffmanLengths(br) // get the Huffman code dictionary
	if err != nil {
		return []byte{}, fmt.Errorf("error while deserializing huffman code dictionary: %w", err)
	}
	d := getHuffmanDictFromLengths(l)  // build the canonical huffman dictionary from the lengths
	t := getHuffmanTreeFromDict(d)     // build the Huffman tree
	inBuffer := bitio.NewBitReader(br) // create a bitreader and traverse the tree with bits
	outBuffer := make([]byte, 0, 2*len(src))
	var (
		padBuffer uint64
		newBit    uint64
	)
	if padBits > 0 {
		padBuffer, err = inBuffer.ReadBits(int(padBits)) // read in the padded bits if there are any
		if err != nil {
			return []byte{}, fmt.Errorf("error while reading first %d bits from source in huffman decoding: %w", padBits, err)
		}
	}
	node := t
	for {
		if node.nodeType == branch {
			newBit, err = inBuffer.ReadBits(1) // read in a bit
			if err != nil {
				break
			}
			if padBits > 0 {
				node = node.children[(padBuffer>>(padBits-1))&0x01] // use the msb of the padded buffer as the decision bit
				padBuffer = (padBuffer << 1) | newBit               // shift the padded buffer left 1 and add the new bit
			} else {
				node = node.children[newBit] // use the new bit as the decision bit if there is no padding.
			}
		} else {
			outBuffer = append(outBuffer, node.value) // if you are at a leaf, you have your value
			node = t                                  // reset to the root tree node
		}
	}
	if errors.Is(err, io.EOF) {
		return outBuffer, nil
	} else {
		return outBuffer, fmt.Errorf("error while reading bit from source in huffman decoding: %w", err)
	}
}

func (HUFFMANCodec) IsLossless() bool {
	return true
}
