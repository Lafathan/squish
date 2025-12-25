package codec

type RLECodec struct{}

func (RLECodec) EncodeBlock(src []byte) ([]byte, uint8, error) {
	out := []byte{}
	count := uint8(0)
	curRun := src[0]
	for _, b := range src {
		if b == curRun && curRun < uint8((1<<8)-1) {
			count += 1
		} else {
			out = append(out, count)
			out = append(out, curRun)
			count = 1
			curRun = b
		}
	}
	out = append(out, count)
	out = append(out, curRun)
	return out, 0, nil
}

func (RLECodec) DecodeBlock(src []byte, padBits uint8) ([]byte, error) {
	return src, nil
}

func (RLECodec) IsLossless() bool {
	return true
}
