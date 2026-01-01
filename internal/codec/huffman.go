package codec

import (
	"bytes"
	"container/heap"
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
	dict := map[byte]*HCode{} // store the byte - code pairs
	curHCode := HCode{bits: []byte{}, length: 0}
	var getCode func(n *Node, c *HCode) // define a func for recursive depth first search
	getCode = func(n *Node, c *HCode) {
		if n.nodeType == Leaf {
			dict[n.value] = c // update the dictionary when you arrive at a leaf
		} else {
			bitsByteLength := len(c.bits)
			if c.length%8 == 0 {
				bitsByteLength++
			}
			lBits := make([]byte, bitsByteLength) // make byte arrays to hold children codes
			rBits := make([]byte, bitsByteLength)
			for i := range bitsByteLength {
				currentByteIndex := len(c.bits) - 1 - i
				childByteIndex := bitsByteLength - 1 - i
				lBits[childByteIndex] = c.bits[currentByteIndex] << 1
				rBits[childByteIndex] = c.bits[currentByteIndex] << 1
				if i > 0 && currentByteIndex > 0 {
					lBits[childByteIndex] &= ((c.bits[currentByteIndex-1] >> 7) & 1)
					rBits[childByteIndex] &= ((c.bits[currentByteIndex-1] >> 7) & 1)
				}
				//if i < len(c.bits) {
				//lBits[i] = c.bits[i] << 1 // each child byte is the parent shifted 1 to the left
				//rBits[i] = c.bits[i] << 1
				//} else {
				//lBits[i] = byte(0) // more bytes than parent? make it zero
				//rBits[i] = byte(0)
				//}
				//if i >= 1 {
				//lBits[i] |= c.bits[i-1] >> 7 // handle carryover bits between bytes
				//rBits[i] |= c.bits[i-1] >> 7
				//}
			}
			rBits[0] |= 0x01
			getCode(n.children[0], &HCode{bits: lBits, length: c.length + 1}) // recurse for children
			getCode(n.children[1], &HCode{bits: rBits, length: c.length + 1})
		}
	}
	getCode(tree, &curHCode)
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

func (HUFFMANCodec) EncodeBlock(src []byte) ([]byte, int, error) {
	h := GetFrequencyMap(src)                                // get freq map
	t := GetHuffmanTreeFromNodes(h)                          // build the tree
	d := GetHuffmanDictFromTree(t)                           // get the dictionary of codes
	outBuffer := new(bytes.Buffer)                           //create a new buffer to write to
	_, err := outBuffer.Write(SerializeHuffmanDictionary(d)) // write the dictionary to it
	bw := bitio.NewBitWriter(outBuffer)                      // make a new bitwriter
	for _, b := range src {
		err = bw.WriteBits(d[b].bits, d[b].length) // write the new bits for each symbol
		if err != nil {
			return []byte{}, 0, err
		}
	}
	pad, err := bw.Flush() // flush it and report back the number of pad bits
	if err != nil {
		return []byte{}, 0, err
	}
	return outBuffer.Bytes(), pad, nil
}

func GetHuffmanTreeFromDict(d map[byte]*HCode) *Node {
	return nil
}

func DeserializeHuffmanDictionary(br io.ByteReader) (map[byte]*HCode, error) {
	dict := map[byte]*HCode{} // make a dictionary
	for {
		bitLength, err := br.ReadByte() // read in the bitlength
		if err != nil {
			return dict, err
		}
		if bitLength == 0 { // zero bit length marks the end of the dictionary
			break
		}
		byteVal, err := br.ReadByte() // read in the symbol
		if err != nil {
			return dict, err
		}
		byteArray := []byte{} // read in all the bytes of bits
		for range (bitLength - 1) / 8 {
			b, err := br.ReadByte()
			if err != nil {
				return dict, err
			}
			byteArray = append([]byte{b}, byteArray...)
		}
		dict[byteVal] = &HCode{bits: byteArray, length: int(bitLength)} // store the HCode in the dictionary
	}
	return dict, nil
}

func (HUFFMANCodec) DecodeBlock(src []byte, padBits uint8) ([]byte, error) {
	//br := bytes.NewBuffer(src)
	//d, err := DeserializeHuffmanDictionary(br)
	//t := GetHuffmanTreeFromDict(d)
	return src, nil
}

func (HUFFMANCodec) IsLossless() bool {
	return false
}
