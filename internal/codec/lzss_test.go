package codec

import (
	"bytes"
	"testing"
)

func LZSSEncodeDecode(message string, t *testing.T) {
	lc := LZSSCodec{}
	coded, err := lc.EncodeBlock([]byte(message))
	if err != nil {
		t.Fatalf("LZSS encoding failed: %v", err)
	}
	decoded, err := lc.DecodeBlock(coded)
	if err != nil {
		t.Fatalf("LZSS decoding failed: %v", err)
	}
	if message != string(decoded) {
		t.Fatalf("LZSS encoding mismatch: got %s - expected %s", string(decoded), message)
	}
}

func TestLZSSEncodeDecode(t *testing.T) {
	message := "The mellow yellow fellow says hello world!"
	LZSSEncodeDecode(message, t)
}

func TestLZSSRunLength(t *testing.T) {
	message := []byte{0, 1}
	a, b := 1, 1
	for i := range 30 {
		a, b = b, a+b
		message = append(message, bytes.Repeat([]byte{byte(i)}, b)...)
	}
	LZSSEncodeDecode(string(message), t)
}

func TestLZSSEmptyMessage(t *testing.T) {
	message := ""
	LZSSEncodeDecode(message, t)
}

func TestLZSSLossless(t *testing.T) {
	hc := LZSSCodec{}
	if !hc.IsLossless() {
		t.Fatalf("LZSS is lossless, but returned lossy")
	}
}
