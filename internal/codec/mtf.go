package codec

import "container/list"

type MTFCodec struct{}

func getAlphabet() *list.List {
	alphabet := list.New() // new doubly linked list
	for i := range 255 {   // for all byte values
		alphabet.PushFront(byte(i)) // add them to the front of the list
	}
	return alphabet
}

func mtf(src []byte, encode bool) ([]byte, error) {
	if len(src) == 0 {
		return src, nil
	}
	var (
		srcIdx            = 0             // where you are in the input
		index       uint8 = 0             // for counting depth of value in dictionary
		alphabet          = getAlphabet() // the alphabet
		comparison  byte                  // what value to compare to
		replacement byte                  // what value to replace with
	)
	for srcIdx < len(src) { // for each value in the input
		for e := alphabet.Front(); e != nil; e = e.Next() { // for each value in the alphabet
			if encode {
				comparison = e.Value.(byte) // swap the input byte with the
				replacement = index         // index of the value in the alphabet
			} else {
				comparison = index           // swap the input byte with the
				replacement = e.Value.(byte) // value at index in the alphabet
			}
			if src[srcIdx] == comparison {
				src[srcIdx] = replacement
				alphabet.MoveToFront(e) // move the match to the front of the alphabet
				index = 0               // start at the beginning again
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
