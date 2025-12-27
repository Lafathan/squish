package codec

type HUFFMANCodec struct{}

type HuffmanNode struct {
	nodeType  int // 0 is leaf, 1 is a node
	parent    *HuffmanNode
	value     byte            // value held by a leaf
	frequency int             // frequency of value, or sum of frequencies of children
	children  [2]*HuffmanNode // children if not a leaf
}

func GetHuffmanLeaves(src []byte) []*HuffmanNode {
	freqMap := map[byte]*HuffmanNode{}
	for _, b := range src {
		_, ok := freqMap[b]
		if !ok {
			freqMap[b] = &HuffmanNode{
				nodeType:  0,
				value:     b,
				frequency: 0,
			}
		}
		freqMap[b].frequency++
	}
	leaves := make([]*HuffmanNode, len(freqMap))
	i := 0
	for _, v := range freqMap {
		leaves[i] = v
		i += 1
	}
	return leaves
}

func TreeFromLeaves(leaves []*HuffmanNode) *HuffmanNode {
	for len(leaves) > 1 {
		leftChildIndex, rightChildIndex := 1, 0
		for i, node := range leaves {
			if node.frequency < leaves[leftChildIndex].frequency {
				rightChildIndex, leftChildIndex = leftChildIndex, i
			} else if node.frequency < leaves[rightChildIndex].frequency {
				rightChildIndex = i
			}
		}
		newNode := HuffmanNode{
			nodeType:  1,
			frequency: leaves[leftChildIndex].frequency + leaves[rightChildIndex].frequency,
			children:  [2]*HuffmanNode{leaves[leftChildIndex], leaves[rightChildIndex]},
		}
		leaves[leftChildIndex].parent = &newNode
		leaves[rightChildIndex].parent = &newNode
		leaves[leftChildIndex] = &newNode
		leaves = append(leaves[:rightChildIndex], leaves[rightChildIndex+1:]...)
	}
	return leaves[0]
}

func (HUFFMANCodec) EncodeBlock(src []byte) ([]byte, uint8, error) {
	unsortedLeaves := GetHuffmanLeaves(src)
	huffmanTree := TreeFromLeaves(unsortedLeaves)
	print(huffmanTree)
	return src, 0, nil
}

func (HUFFMANCodec) DecodeBlock(src []byte, padBits uint8) ([]byte, error) {
	return src, nil
}

func (HUFFMANCodec) IsLossless() bool {
	return false
}
