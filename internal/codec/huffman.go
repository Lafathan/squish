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
	Leaf   = 0
	Branch = 1
)

type HUFFMANCodec struct{}

type Node struct {
	nodeType  int      // 0 is leaf, 1 is a node
	value     byte     // value held by a leaf
	frequency int      // frequency of value, or sum of frequencies of children
	children  [2]*Node // children if not a leaf
}

type HCode struct {
	bits   []byte
	length int
}

type HuffmanHeap []*Node

// define function required to inherit the heap interface
func (h HuffmanHeap) Len() int           { return len(h) }
func (h HuffmanHeap) Less(i, j int) bool { return h[i].frequency < h[j].frequency }
func (h HuffmanHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *HuffmanHeap) Push(n any)        { *h = append(*h, n.(*Node)) }
func (h *HuffmanHeap) Pop() any {
	n := len(*h)
	x := (*h)[n-1]
	*h = (*h)[0 : n-1]
	return x
}

func ShiftByteSliceLeft(bytes []byte, length int) []byte {
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

func GetFrequencyMap(src []byte) []int {
	freqMap := make([]int, 256)
	for _, b := range src {
		freqMap[b]++
	}
	return freqMap
}

func GetHuffmanTreeFromFreqMap(freqMap []int) *Node {
	leaves := &HuffmanHeap{}       // instantiate a heap
	heap.Init(leaves)              // initialize it
	for b, freq := range freqMap { // add nodes to the heap based on the freq map
		if freq > 0 {
			heap.Push(leaves, &Node{nodeType: Leaf, value: byte(b), frequency: freq})
		}
	}
	for leaves.Len() > 1 {
		l := heap.Pop(leaves).(*Node) // get the smallest left child node
		r := heap.Pop(leaves).(*Node) // get the second smalleset right child node
		newNode := Node{              // create a new parent node for those children
			nodeType:  Branch,
			frequency: l.frequency + r.frequency,
			children:  [2]*Node{l, r},
		}
		heap.Push(leaves, &newNode) // push that new parent back on to the heap
	}
	return heap.Pop(leaves).(*Node)
}

func GetHuffmanDictFromTree(tree *Node) map[byte]*HCode {
	dict := map[byte]*HCode{}           // store the byte - code pairs
	var getCode func(n *Node, c *HCode) // define a func for recursive depth first search
	getCode = func(n *Node, c *HCode) {
		if n.nodeType == Leaf {
			dict[n.value] = c // update the dictionary when you arrive at a leaf
		} else {
			lBits := ShiftByteSliceLeft(c.bits, c.length)
			rBits := append([]byte(nil), lBits...)
			rBits[len(rBits)-1] |= 0x01
			getCode(n.children[0], &HCode{bits: lBits, length: c.length + 1}) // recurse for children
			getCode(n.children[1], &HCode{bits: rBits, length: c.length + 1})
		}
	}
	getCode(tree, &HCode{bits: []byte{}, length: 0})
	return dict
}

func SerializeHuffmanDictionary(d map[byte]*HCode) []byte {
	// uint8 bit length, byte value, packed bytes for bit code
	out := []byte{}
	for k, v := range d {
		out = append(out, uint8(v.length))
		out = append(out, k)
		out = append(out, v.bits...)
	}
	out = append(out, byte(0)) // a zero bit length marks the end of the dictionary
	return out
}

func (HUFFMANCodec) EncodeBlock(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return []byte{}, nil
	}
	f := GetFrequencyMap(src)                                // get freq map
	t := GetHuffmanTreeFromFreqMap(f)                        // build the tree
	d := GetHuffmanDictFromTree(t)                           // get the dictionary of codes
	outBuffer := new(bytes.Buffer)                           // create a new buffer to write to
	_, err := outBuffer.Write(SerializeHuffmanDictionary(d)) // write the dictionary to it
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

func GetHuffmanTreeFromDict(d [256]*HCode) *Node {
	root := Node{nodeType: Branch}                                 // make an empty root node
	var buildTree func(n *Node, val byte, bits []byte, length int) // define for recursion
	buildTree = func(n *Node, val byte, bits []byte, length int) {
		if length > 0 {
			bitPos := length - 1                  // get the position of the decision bit
			byteIndex := len(bits) - 1 - bitPos/8 // get the byte index of that bit
			shift := bitPos % 8                   // shift required to move the decision bit to lsb
			bit := (bits[byteIndex] >> shift) & 1 // isolate the decision bit
			if n.children[bit] == nil {
				n.children[bit] = &Node{nodeType: Branch} // create the child node if it doesn't exist
			}
			buildTree(n.children[bit], val, bits, length-1) // recurse into the child node
		} else {
			n.nodeType = Leaf // if you are at the end of your bit stream, you are at a leaf
			n.value = val
		}
	}
	for k, v := range d {
		if v != nil {
			buildTree(&root, byte(k), v.bits, v.length)
		}
	}
	return &root
}

func DeserializeHuffmanDictionary(br io.Reader) ([256]*HCode, error) {
	HCodeMap := [256]*HCode{} // make an array to store all Huffman codes
	temp := make([]byte, 1)   // make a 1 byte slice buffer to read into
	var (
		bitLength uint8
		byteVal   byte
		err       error
	)
	for {
		_, err = io.ReadFull(br, temp) // read in the bit length
		if err != nil {
			return HCodeMap, fmt.Errorf("error while reading bit length from huffman code dictionary: %w", err)
		}
		bitLength = temp[0]
		if bitLength == 0 { // zero bit length marks the end of the dictionary
			break
		}
		_, err = io.ReadFull(br, temp) // read in the byte value
		if err != nil {
			return HCodeMap, fmt.Errorf("error while reading symbol from huffman code dictionary: %w", err)
		}
		byteVal = temp[0]
		byteArray := make([]byte, (bitLength+7)/8) // make a byte slice to read in the bit stream bytes
		_, err = io.ReadFull(br, byteArray)
		if err != nil {
			return HCodeMap, fmt.Errorf("error while reading bit stream for symbol from huffman code dictionary: %w", err)
		}
		HCodeMap[byteVal] = &HCode{bits: byteArray, length: int(bitLength)} // store the HCode in the array
	}
	return HCodeMap, nil
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
	d, err := DeserializeHuffmanDictionary(br) // get the Huffman code dictionary
	if err != nil {
		return []byte{}, fmt.Errorf("error while deserializing huffman code dictionary: %w", err)
	}
	t := GetHuffmanTreeFromDict(d)     // build the Huffman tree
	inBuffer := bitio.NewBitReader(br) // create a bitreader and traverse the tree with bits
	outBuffer := new(bytes.Buffer)     // create a new buffer to write to
	var (
		padBuffer uint64
		newBit    uint64
	)
	if padBits > 0 {
		padBuffer, err = inBuffer.ReadBits(int(padBits))
		if err != nil {
			return []byte{}, fmt.Errorf("error while reading first %d bits from source in huffman decoding: %w", padBits, err)
		}
	}
	node := t
	for {
		if node.nodeType == Branch {
			newBit, err = inBuffer.ReadBits(1)
			if errors.Is(err, io.EOF) {
				break
			} else if err != nil {
				return outBuffer.Bytes(), fmt.Errorf("error while reading bit from source in huffman decoding: %w", err)
			}
			if padBits > 0 {
				node = node.children[(padBuffer>>(padBits-1))&0x01]
				padBuffer = (padBuffer << 1) | newBit
			} else {
				node = node.children[newBit]
			}
		} else {
			outBuffer.WriteByte(node.value) // if you are at a leaf, you have your value
			node = t                        // reset to the root tree node
		}
	}
	return outBuffer.Bytes(), nil
}

func (HUFFMANCodec) IsLossless() bool {
	return true
}
