package codec

type HUFFMANCodec struct{}

type HuffmanNode struct {
	nodeType  int                  // 0 is leaf, 1 is a node
	value     byte                 // value held by a leaf
	frequency int                  // frequency of value, or sum of frequencies of children
	children  map[int]*HuffmanNode // children if not a leaf
}

func GetHuffmanLeaves(src []byte) map[byte]*HuffmanNode {
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
		freqMap[b].frequency = freqMap[b].frequency + 1
	}
	return freqMap
}

func (HUFFMANCodec) EncodeBlock(src []byte) ([]byte, uint8, error) {
	return src, 0, nil
}

func (HUFFMANCodec) DecodeBlock(src []byte, padBits uint8) ([]byte, error) {
	return src, nil
}

func (HUFFMANCodec) IsLossless() bool {
	return false
}
