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
	length uint8
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

func ShiftByteSliceLeft(bytes []byte, length uint8) []byte {
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

func GetFrequencyMap(src []byte) map[byte]*Node {
	if len(src) == 0 {
		return nil
	}
	freqMap := map[byte]*Node{} // build a dictionary of byte frequencies
	for _, b := range src {     // loop through bytes
		_, ok := freqMap[b] // check for existence
		if !ok {            // create it if it doesn't exist
			freqMap[b] = &Node{
				nodeType:  Leaf,
				value:     b,
				frequency: 0,
			}
		}
		freqMap[b].frequency++ // increment the frequency
	}
	return freqMap // return the frequency mapping
}

func GetHuffmanTreeFromNodes(freqMap map[byte]*Node) *Node {
	leaves := &HuffmanHeap{} // build the heap from the leaf nodes
	heap.Init(leaves)
	for _, v := range freqMap {
		heap.Push(leaves, v)
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
	h := GetFrequencyMap(src)                                // get freq map
	t := GetHuffmanTreeFromNodes(h)                          // build the tree
	d := GetHuffmanDictFromTree(t)                           // get the dictionary of codes
	outBuffer := new(bytes.Buffer)                           // create a new buffer to write to
	_, err := outBuffer.Write(SerializeHuffmanDictionary(d)) // write the dictionary to it
	bw := bitio.NewBitWriter(outBuffer)                      // make a new bitwriter
	for _, b := range src {
		err = bw.WriteBits(d[b].bits, int(d[b].length)) // write the new bits for each symbol
		if err != nil {
			return []byte{}, fmt.Errorf("error while writing bits during huffman encoding: %w", err)
		}
	}
	pad, err := bw.Flush() // flush it and report back the number of pad bits
	if err != nil {
		return []byte{}, fmt.Errorf("error while flushing bitwriter during huffman encoding: %w", err)
	}
	out := append([]byte{byte(pad)}, outBuffer.Bytes()...)
	return out, nil
}

func GetHuffmanTreeFromDict(d map[byte]*HCode) *Node {
	root := Node{nodeType: Branch}                                   // make an empty root node
	var buildTree func(n *Node, val byte, bits []byte, length uint8) // define for recursion
	buildTree = func(n *Node, val byte, bits []byte, length uint8) {
		if length > 0 {
			bitPos := length - 1                         // get the position of the decision bit
			byteIndex := uint8(len(bits)) - 1 - bitPos/8 // get the byte index of that bit
			shift := bitPos % 8                          // shift required to move the decision bit to lsb
			bit := (bits[byteIndex] >> shift) & 1        // isolate the decision bit
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
		buildTree(&root, k, v.bits, v.length)
	}
	return &root
}

func DeserializeHuffmanDictionary(br io.ByteReader) (map[byte]*HCode, error) {
	dict := map[byte]*HCode{} // make a dictionary
	for {
		bitLength, err := br.ReadByte() // read in the bitlength
		if err != nil {
			return dict, fmt.Errorf("error while reading bit length from huffman code dictionary: %w", err)
		}
		if bitLength == 0 { // zero bit length marks the end of the dictionary
			break
		}
		byteVal, err := br.ReadByte() // read in the symbol
		if err != nil {
			return dict, fmt.Errorf("error while reading symbol from huffman code dictionary: %w", err)
		}
		byteArray := []byte{} // read in all the bytes of bits
		for range (bitLength + 7) / 8 {
			b, err := br.ReadByte()
			if err != nil {
				return dict, fmt.Errorf("error while reading %d bits for symbol %s from huffman code dictionary: %w", bitLength, string(byteVal), err)
			}
			byteArray = append(byteArray, b)
		}
		dict[byteVal] = &HCode{bits: byteArray, length: bitLength} // store the HCode in the dictionary
	}
	return dict, nil
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
	t := GetHuffmanTreeFromDict(d)                    // build the Huffman tree
	inBuffer := bitio.NewBitReader(br)                // create a bitreader and traverse the tree with bits
	outBuffer := new(bytes.Buffer)                    //create a new buffer to write to
	padBuffer, err := inBuffer.ReadBits(int(padBits)) // a padding buffer to keep from reading padded bits
	if err != nil {
		return outBuffer.Bytes(), fmt.Errorf("error while reading in initial %d bits from source in huffman decoding: %w", padBits, err)
	}
	node := t
	for {
		if node.nodeType == Branch {
			bit := (padBuffer[0] >> ((padBits - 1) % 8)) & 0x01
			node = node.children[bit]           // got to the appropriate child node
			newBit, err := inBuffer.ReadBits(1) // get the next bit
			if errors.Is(err, io.EOF) {
				break
			} else if err != nil {
				print(err.Error())
				return outBuffer.Bytes(), fmt.Errorf("error while reading bit from source in huffman decoding: %w", err)
			}
			padBuffer = ShiftByteSliceLeft(padBuffer, padBits)
			padBuffer[len(padBuffer)-1] |= newBit[0]
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
