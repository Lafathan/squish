package codec

import (
	"bytes"
	"container/heap"
	"errors"
	"fmt"
	"io"
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
	bits   []byte
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

func shiftByteSliceLeft(bytes []byte, length int) []byte {
	bitsByteLength := len(bytes)
	if length%8 == 0 {
		bitsByteLength++
	}
	newBits := make([]byte, bitsByteLength) // make byte slice to hold children codes
	offset := 0                             // get an offset if the children have an additional byte
	if len(newBits) > len(bytes) {
		offset = 1
	}
	for i := range len(bytes) {
		newBits[i+offset] = (bytes[i] << 1) & 0xFE // shift the parent
		if i+offset > 0 {                          // store the carryover if necessary
			newBits[i+offset-1] |= (bytes[i] >> 7) & 0x01
		}
	}
	return newBits
}

func getFrequencyMap(src []byte) []int {
	freqMap := make([]int, 256)
	for _, b := range src {
		freqMap[b]++
	}
	return freqMap
}

func getHuffmanTreeFromFreqMap(freqMap []int) *node {
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

func getHuffmanDictFromTree(tree *node) *[256]hCode {
	dict := [256]hCode{}                // store the byte - code pairs
	var getCode func(n *node, c *hCode) // define a func for recursive depth first search
	getCode = func(n *node, c *hCode) {
		if n.nodeType == leaf {
			dict[n.value] = *c // update the dictionary when you arrive at a leaf
		} else {
			lBits := shiftByteSliceLeft(c.bits, c.length)
			rBits := append([]byte(nil), lBits...)
			rBits[len(rBits)-1] |= 0x01
			getCode(n.children[0], &hCode{bits: lBits, length: c.length + 1}) // recurse for children
			getCode(n.children[1], &hCode{bits: rBits, length: c.length + 1})
		}
	}
	getCode(tree, &hCode{bits: []byte{}, length: 0})
	return &dict
}

func serializeHuffmanDictionary(d *[256]hCode) []byte {
	// uint8 bit length, byte value, packed bytes for bit code
	out := []byte{}
	for k, v := range d {
		if v.length > 0 {
			out = append(out, byte(v.length))
			out = append(out, byte(k))
			out = append(out, v.bits...)
		}
	}
	out = append(out, byte(0)) // a zero bit length marks the end of the dictionary
	return out
}

func (HUFFMANCodec) EncodeBlock(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return []byte{}, nil
	}
	f := getFrequencyMap(src)                                // get freq map
	t := getHuffmanTreeFromFreqMap(f)                        // build the tree
	d := getHuffmanDictFromTree(t)                           // get the dictionary of codes
	outBuffer := new(bytes.Buffer)                           // create a new buffer to write to
	_, err := outBuffer.Write(serializeHuffmanDictionary(d)) // write the dictionary to it
	bw := bitio.NewBitWriter(outBuffer)                      // make a new bitwriter
	var outInt uint64
	for _, b := range src {
		if d[b].length > 64 { // if the bit stream is too large for the bitwriter
			for _, curByte := range d[b].bits { // write each byte independently
				err = bw.WriteBits(uint64(curByte), d[b].length) // write the new bits for each symbol
				if err != nil {
					return []byte{}, fmt.Errorf("error while writing large bit stream during huffman encoding: %w", err)
				}
			}
		} else {
			outInt = 0
			for _, curByte := range d[b].bits {
				outInt = (outInt << 8) | uint64(curByte) // if it is not too big, make a uint64 that holds the bits
			}
			err = bw.WriteBits(outInt, int(d[b].length)) // write the new bits for each symbol
			if err != nil {
				return []byte{}, fmt.Errorf("error while writing bit stream during huffman encoding: %w", err)
			}
		}
	}
	pad, err := bw.Flush() // flush it and report back the number of pad bits
	if err != nil {
		return []byte{}, fmt.Errorf("error while flushing bitwriter during huffman encoding: %w", err)
	}
	out := append([]byte{byte(pad)}, outBuffer.Bytes()...)
	return out, nil
}

func getHuffmanTreeFromDict(d *[256]hCode) *node {
	root := node{nodeType: branch}                                 // make an empty root node
	var buildTree func(n *node, val byte, bits []byte, bitPos int) // define for recursion
	buildTree = func(n *node, val byte, bits []byte, bitPos int) {
		if bitPos >= 0 {
			byteIndex := len(bits) - 1 - bitPos/8 // get the byte index of that bit
			shift := bitPos % 8                   // shift required to move the decision bit to lsb
			bit := (bits[byteIndex] >> shift) & 1 // isolate the decision bit
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

func deserializeHuffmanDictionary(br io.Reader) (*[256]hCode, error) {
	hCodeMap := [256]hCode{} // make an array to store all Huffman codes
	temp := make([]byte, 1)  // make a 1 byte slice buffer to read into
	var (
		bitLength uint8
		err       error
	)
	for {
		_, err = io.ReadFull(br, temp) // read in the bit length
		if err != nil {
			return &hCodeMap, fmt.Errorf("error while reading bit length from huffman dictionary: %w", err)
		}
		bitLength = temp[0]
		if bitLength == 0 { // zero bit length marks the end of the dictionary
			break
		}
		byteArray := make([]byte, (bitLength+7)/8+1) // make a byte slice to read in the byte value and bit stream
		_, err = io.ReadFull(br, byteArray)
		if err != nil {
			return &hCodeMap, fmt.Errorf("error while reading value or bit stream from huffman dictionary: %w", err)
		}
		hCodeMap[byteArray[0]] = hCode{bits: byteArray[1:], length: int(bitLength)} // store the HCode in the array
	}
	return &hCodeMap, nil
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
	d, err := deserializeHuffmanDictionary(br) // get the Huffman code dictionary
	if err != nil {
		return []byte{}, fmt.Errorf("error while deserializing huffman code dictionary: %w", err)
	}
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
