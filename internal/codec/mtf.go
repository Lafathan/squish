package codec

import "container/list"

type MTFCodec struct{}

func getAlphabet() *list.List {
	alphabet := list.New()
	for i := range 255 {
		alphabet.PushFront(byte(i))
	}
	return alphabet
}

func mtf(src []byte, encode bool) ([]byte, error) {
	if len(src) == 0 {
		return src, nil
	}
	var (
		srcIdx            = 0
		index       uint8 = 0
		alphabet          = getAlphabet()
		comparison  byte
		replacement byte
	)
	for srcIdx < len(src) {
		for e := alphabet.Front(); e != nil; e = e.Next() {
			if encode {
				comparison = e.Value.(byte)
				replacement = index
			} else {
				comparison = index
				replacement = e.Value.(byte)
			}
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
