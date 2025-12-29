package codec

import (
	"bytes"
	"container/heap"
	"encoding/binary"
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
	bits   uint32
	length uint8
}

type HuffmanHeap []*Node

// define function required to inherit the heap interface
func (h HuffmanHeap) Len() int           { return len(h) }
func (h HuffmanHeap) Less(i, j int) bool { return h[i].frequency < h[j].frequency }
func (h HuffmanHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *HuffmanHeap) Push(n any)        { *h = append(*h, n.(*Node)) }
func (h *HuffmanHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func GetHuffmanHeap(src []byte) *HuffmanHeap {
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
	h := &HuffmanHeap{} // build the heap from the leaf nodes
	heap.Init(h)
	for _, v := range freqMap {
		heap.Push(h, v)
	}
	return h // return the heap
}

func GetHuffmanTree(leaves *HuffmanHeap) *Node {
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
	curHCode := HCode{bits: 0, length: 0}
	var getCode func(n *Node, c *HCode) // define a func for recursive depth first search
	getCode = func(n *Node, c *HCode) {
		if n.nodeType == Leaf {
			dict[n.value] = c // update the dictionary when you arrive at a leaf
		} else {
			getCode(n.children[0], &HCode{bits: c.bits << 1, length: c.length + 1}) // recurse for children
			getCode(n.children[1], &HCode{bits: c.bits<<1 | 1, length: c.length + 1})
		}
	}
	getCode(tree, &curHCode)
	return dict
}

func SerializeHuffmanDictionary(d map[byte]*HCode) []byte {
	// byte value, uint32 code, uint8 bit length - all 0's mark the end of the table
	out := []byte{}
	for k, v := range d {
		out = append(out, k)
		out = binary.BigEndian.AppendUint32(out, v.bits)
		out = append(out, v.length)
	}
	out = append(out, byte(0))
	out = binary.BigEndian.AppendUint32(out, 0)
	out = append(out, byte(0))
	return out
}

func (HUFFMANCodec) EncodeBlock(src []byte) ([]byte, uint8, error) {
	h := GetHuffmanHeap(src)
	t := GetHuffmanTree(h)
	d := GetHuffmanDictionary(t)
	outBuffer := new(bytes.Buffer)
	_, err := outBuffer.Write(SerializeHuffmanDictionary(d))
	bw := bitio.NewBitWriter(outBuffer)
	for _, b := range src {
		err := bw.WriteBits(uint64(d[b].bits), d[b].length)
		if err != nil {
			return []byte{}, 0, err
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
