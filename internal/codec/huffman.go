package codec

import (
	"bytes"
	"container/heap"
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

func GetHuffmanTree(freqMap map[byte]*Node) *Node {
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

func GetHuffmanDictionary(tree *Node) map[byte]*HCode {
	dict := map[byte]*HCode{} // store the byte - code pairs
	curHCode := HCode{bits: []byte{}, length: 0}
	var getCode func(n *Node, c *HCode) // define a func for recursive depth first search
	getCode = func(n *Node, c *HCode) {
		if n.nodeType == Leaf {
			dict[n.value] = c // update the dictionary when you arrive at a leaf
		} else {
			childBitsByteLength := len(c.bits)
			if c.length%8 == 0 {
				childBitsByteLength++
			}
			leftChildBits := make([]byte, childBitsByteLength) // make byte arrays to hold children codes
			rightChildBits := make([]byte, childBitsByteLength)
			for i := range childBitsByteLength {
				if i < len(c.bits) {
					leftChildBits[i] = c.bits[i] << 1 // each child byte is the parent shifted 1 to the left
					rightChildBits[i] = c.bits[i] << 1
				} else {
					leftChildBits[i] = byte(0) // more bytes than parent? make it zero
					rightChildBits[i] = byte(0)
				}
				if i > 0 {
					leftChildBits[i] |= c.bits[i-1] >> 7 // handle carryover bits between bytes
					rightChildBits[i] |= c.bits[i-1] >> 7
				}
			}
			rightChildBits[0] |= 0x01
			getCode(n.children[0], &HCode{bits: leftChildBits, length: c.length + 1}) // recurse for children
			getCode(n.children[1], &HCode{bits: rightChildBits, length: c.length + 1})
		}
	}
	getCode(tree, &curHCode)
	return dict
}

func SerializeHuffmanDictionary(d map[byte]*HCode) []byte {
	// uint8 bit length, byte value, packed bytes for bit code - 0 bit length marks the end of the table
	out := []byte{}
	for k, v := range d {
		out = append(out, v.length)
		out = append(out, k)
		out = append(out, v.bits...)
	}
	out = append(out, byte(0))
	return out
}

func (HUFFMANCodec) EncodeBlock(src []byte) ([]byte, uint8, error) {
	h := GetFrequencyMap(src)
	t := GetHuffmanTree(h)
	d := GetHuffmanDictionary(t)
	outBuffer := new(bytes.Buffer)
	_, err := outBuffer.Write(SerializeHuffmanDictionary(d))
	bw := bitio.NewBitWriter(outBuffer)
	// this is awful
	// bitreader and writer should work off packed bits, not be limited to uint64
	// fixing bitreader and writer will fix this to just have to write the byte array
	for _, b := range src {
		curByte := len(d[b].bits) - 1
		var err error
		for curByte >= 0 {
			if curByte > 0 {
				err = bw.WriteBits(uint64(d[b].bits[curByte]), 8)
			} else {
				err = bw.WriteBits(uint64(d[b].bits[curByte]), d[b].length%4)
			}
			curByte--
			if err != nil {
				return []byte{}, 0, err
			}
		}
	}
	pad, err := bw.Flush()
	if err != nil {
		return []byte{}, 0, err
	}
	return outBuffer.Bytes(), pad, nil
}

func (HUFFMANCodec) DecodeBlock(src []byte, padBits uint8) ([]byte, error) {
	return src, nil
}

func (HUFFMANCodec) IsLossless() bool {
	return false
}
