package codec

type XORCodec struct{}

func xor(src []byte, encode bool) ([]byte, error) {
	// encode src using an XOR transform
	if len(src) < 2 {
		return src, nil
	}
	var (
		out = make([]byte, len(src))
	)
	out[0] = src[0]
	if encode {
		for i := 1; i < len(src); i++ {
			out[i] = src[i] ^ src[i-1]
		}
	} else {
		for i := 1; i < len(src); i++ {
			out[i] = src[i] ^ out[i-1]
		}
	}
	return out, nil

}

func (XORCodec) EncodeBlock(src []byte) ([]byte, error) {
	return xor(src, true)
}

func (XORCodec) DecodeBlock(src []byte) ([]byte, error) {
	return xor(src, false)
}

func (XORCodec) IsLossless() bool {
	return true
}
