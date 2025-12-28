package codec

const (
	Leaf   = 0
	Branch = 1
)

type HUFFMANCodec struct{}

type HuffmanNode struct {
	nodeType  int // 0 is leaf, 1 is a node
	parent    *HuffmanNode
	value     byte            // value held by a leaf
	frequency int             // frequency of value, or sum of frequencies of children
	children  [2]*HuffmanNode // children if not a leaf
}

func GetHuffmanLeaves(src []byte) []*HuffmanNode {
	if len(src) == 0 {
		return nil
	}
	freqMap := map[byte]*HuffmanNode{} // build a dictionary of byte frequencies
	for _, b := range src {            // loop through bytes
		_, ok := freqMap[b] // check for existence
		if !ok {            // create it if it doesn't exist
			freqMap[b] = &HuffmanNode{
				nodeType:  Leaf,
				value:     b,
				frequency: 0,
			}
		}
		freqMap[b].frequency++ // increment the frequency
	}
	leaves := make([]*HuffmanNode, len(freqMap)) // make a list of leaves
	i := 0
	for _, v := range freqMap {
		leaves[i] = v
		i++
	}
	return leaves // return the list
}

func TreeFromLeaves(leaves []*HuffmanNode) *HuffmanNode {
	for len(leaves) > 1 { // while you are not at the root...
		leftChildIndex := 0
		for i, node := range leaves { // loop through the nodes picking the smallest frequency leaf
			if node.frequency < leaves[leftChildIndex].frequency {
				leftChildIndex = i
			}
		}
		rightChildIndex := (leftChildIndex + 1) % len(leaves) // now do it for the right child
		for i, node := range leaves {
			if node.frequency < leaves[rightChildIndex].frequency && i != leftChildIndex {
				rightChildIndex = i // pick the right child ensuring it is unique to the left
			}
		}
		newNode := HuffmanNode{ // build a new node containing the two children
			nodeType:  Branch,
			frequency: leaves[leftChildIndex].frequency + leaves[rightChildIndex].frequency,
			children:  [2]*HuffmanNode{leaves[leftChildIndex], leaves[rightChildIndex]},
		}
		leaves[leftChildIndex].parent = &newNode
		leaves[rightChildIndex].parent = &newNode
		maxIndex := max(leftChildIndex, rightChildIndex)           // pick the rightmost index
		leaves = append(leaves[:maxIndex], leaves[maxIndex+1:]...) // delete it
		minIndex := min(leftChildIndex, rightChildIndex)           // pick the leftmost index
		leaves = append(leaves[:minIndex], leaves[minIndex+1:]...) // delete it
		leaves = append(leaves, &newNode)                          // append the new node you created
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
