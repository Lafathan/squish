package codec

import (
	"math/rand/v2"
	"slices"
	"testing"
)

var lossy = []uint8{
	StringToCodecIDMap["LRLE"],
	StringToCodecIDMap["LRLE2"],
	StringToCodecIDMap["LRLE3"],
	StringToCodecIDMap["LRLE4"],
}

var testSizes = []int{
	0,        // empty
	2,        // 2 B
	2256,     // 256 B
	4 << 10,  // 4 KiB
	64 << 10, // 64 KiB
	1 << 20,  // 1 MiB
}

var datasets = []struct {
	name string
	fn   func(n int) []byte
}{
	{"random", makeRandom},
	{"zeros", makeZeros},
	{"repeating", makeRepeating},
	{"ramp", makeRamp},
	{"fibonacci", makeFibonacci},
}

func makeRandom(n int) []byte {
	out := make([]byte, n)
	var seed [32]byte
	cha := rand.NewChaCha8(seed)
	_, _ = cha.Read(out)
	return out
}

func makeZeros(n int) []byte {
	return make([]byte, n)
}

func makeRepeating(n int) []byte {
	pattern := "My name is Ozymandias, King of Kings: Look on my works, ye Mighty, and despair!\n"
	out := make([]byte, n)
	for i := range n {
		out[i] = pattern[i%len(pattern)]
	}
	return out
}

func makeRamp(n int) []byte {
	out := make([]byte, n)
	for i := range n {
		out[i] = byte(i)
	}
	return out
}

func makeFibonacci(n int) []byte {
	out := make([]byte, n)
	if n < 3 {
		return out
	}
	idx := 2
	fib1, fib2 := 1, 1
	out[0] = 0
	out[1] = 1
	val := byte(2)
	for idx < n {
		fib1, fib2 = fib2, fib1+fib2
		for range fib2 {
			out[idx] = val
			idx++
			if idx >= n {
				break
			}
		}
		val++
	}
	return out
}

func TestCodecs(t *testing.T) {
	for codecName, codecID := range StringToCodecIDMap {
		c := CodecMap[codecID]
		for _, ds := range datasets {
			for _, n := range testSizes {
				input := ds.fn(n)
				if codecID == AUTO {
					encoded, err := c.EncodeBlock([]byte(input))
					if err != nil {
						t.Fatalf("Codec AUTO failed to encode data type %s of length %d: %v", ds.name, n, err)
					}
					decoded := encoded
					for i := len(c.(*AUTOCodec).CodecIDs) - 1; i >= 0; i-- {
						decoded, err = CodecMap[uint8(c.(*AUTOCodec).CodecIDs[i])].DecodeBlock(decoded)
						if err != nil {
							t.Fatalf("Codec AUTO failed to decode encoded data: %v", err)
						}
					}
					if string(input) != string(decoded) {
						t.Fatalf("AUTO encoding mismatch: got %s - expected %s", decoded, input)
					}

				} else {
					encoded, err := c.EncodeBlock(input)
					if err != nil {
						t.Fatalf("Codec %s failed to encode data type %s of length %d: %v", codecName, ds.name, n, err)
					}
					decoded, err := c.DecodeBlock(encoded)
					if err != nil {
						t.Fatalf("Codec %s failed to decode encoded data type %s of length %d: %v", codecName, ds.name, len(encoded), err)
					}
					isLossy := slices.Contains(lossy, codecID)
					if (isLossy && c.IsLossless()) || (!isLossy && !c.IsLossless()) {
						t.Fatalf("Codec %s is lossy, but reported lossless or vice versa", codecName)
					}
					if c.IsLossless() && string(input) != string(decoded) {
						t.Fatalf("Codec %s failed to reproduce input in roundtrip (%s, %v)", codecName, ds.name, n)
					}
				}
			}
		}
	}
}
