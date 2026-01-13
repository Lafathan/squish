package codec

import (
	"bytes"
	"testing"
)

func HuffmanEncodeDecode(message string, t *testing.T) {
	hc := HUFFMANCodec{}
	coded, err := hc.EncodeBlock([]byte(message))
	if err != nil {
		t.Fatalf("Huffman encoding failed: %v", err)
	}
	decoded, err := hc.DecodeBlock(coded)
	if err != nil {
		t.Fatalf("Huffman decoding failed: %v", err)
	}
	if message != string(decoded) {
		t.Fatalf("Huffman encoding mismatch: got %s - expected %s", string(decoded), message)
	}
}

func TestHuffmanEncodeDecode(t *testing.T) {
	message := "Hello World!"
	HuffmanEncodeDecode(message, t)
}

func TestHuffmanRunLength(t *testing.T) {
	message := []byte{0, 1}
	a, b := 1, 1
	for i := range 30 {
		a, b = b, a+b
		message = append(message, bytes.Repeat([]byte{byte(i)}, b)...)
	}
	HuffmanEncodeDecode(string(message), t)
}

func TestHuffmanEmptyMessage(t *testing.T) {
	message := ""
	HuffmanEncodeDecode(message, t)
}

func TestHuffLossless(t *testing.T) {
	hc := HUFFMANCodec{}
	if !hc.IsLossless() {
		t.Fatalf("HUFFMAN is lossless, but returned lossy")
	}
}
