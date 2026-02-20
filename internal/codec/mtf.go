package codec

import "container/list"

type MTFCodec struct{}

func getAlphabet() *list.List {
	// build a doubly linked list for an byte alphabet
	alphabet := list.New()
	for i := range 256 {
		alphabet.PushFront(byte(i))
	}
	return alphabet
}

func mtf(src []byte, encode bool) ([]byte, error) {
	// encode src using move-to-front transform (MTF)
	if len(src) == 0 {
		return src, nil
	}
	var (
		srcIdx            = 0
		index       uint8 = 0 // for counting depth of value in dictionary
		alphabet          = getAlphabet()
		comparison  byte  // what value to compare to
		replacement byte  // what value to replace with
	)
	for srcIdx < len(src) {
		// go through the alphabet until you encounter the current value
		for e := alphabet.Front(); e != nil; e = e.Next() {
			if encode {
				// for encoding compare input to alphabet value, replace with index of match
				comparison = e.Value.(byte)
				replacement = index
			} else {
				// for decoding compare input to indiex value, replace with value of alphabet
				comparison = index
				replacement = e.Value.(byte)
			}
			// perform comparison, moving matching alphabet element to the front
			if src[srcIdx] == comparison {
				src[srcIdx] = replacement
				alphabet.MoveToFront(e)
				index = 0
				break
			} else {
				index++
			}
		}
		srcIdx++
	}
	return src, nil
}

func (MTFCodec) EncodeBlock(src []byte) ([]byte, error) {
	return mtf(src, true)
}

func (MTFCodec) DecodeBlock(src []byte) ([]byte, error) {
	return mtf(src, false)
}

func (MTFCodec) IsLossless() bool {
	return true
}
