package codec

import (
	"testing"
)

func TestHuffman(t *testing.T) {
	message := "Hello World!"
	hc := HUFFMANCodec{}
	out, pad, err := hc.EncodeBlock([]byte(message))
	if err != nil {
		t.Fatalf("Error in Huffman encoding: %v", err)
	}
	decoded, err := hc.DecodeBlock(out, pad)
	if err != nil {
		t.Fatalf("Error in Huffman decoding: %v", err)
	}
	if message != string(decoded) {
		t.Fatalf("Rle encoding mismatch: got %s - expected %s", string(decoded), message)
	}
}

func TestHuffLossless(t *testing.T) {
	hc := HUFFMANCodec{}
	if !hc.IsLossless() {
		t.Fatalf("HUFFMAN is lossless, but returned lossy")
	}
}
