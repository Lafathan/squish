package codec

type DELTACodec struct{}

func delta(src []byte, encode bool) ([]byte, error) {
	// encode source using a delta encoding
	if len(src) < 2 {
		return src, nil
	}
	var (
		out = make([]byte, len(src))
	)
	out[0] = src[0]
	if encode {
		for i := 1; i < len(src); i++ {
			out[i] = byte(int8(src[i]) - int8(src[i-1]))
		}
	} else {
		for i := 1; i < len(src); i++ {
			out[i] = byte(int8(src[i]) + int8(out[i-1]))
		}
	}
	return out, nil
}

func (DELTACodec) EncodeBlock(src []byte) ([]byte, error) {
	return delta(src, true)
}

func (DELTACodec) DecodeBlock(src []byte) ([]byte, error) {
	return delta(src, false)
}

func (DELTACodec) IsLossless() bool {
	return true
}
