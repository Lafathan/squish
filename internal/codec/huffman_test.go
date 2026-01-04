package codec

import (
	"strings"
	"testing"
)

func HuffmanEncodeDecode(message string, t *testing.T) {
	hc := HUFFMANCodec{}
	coded, err := hc.EncodeBlock([]byte(message))
	if err != nil {
		t.Fatalf("Huffman encoding failed")
	}
	decoded, err := hc.DecodeBlock(coded)
	if err != nil {
		t.Fatalf("Huffman decoding failed")
	}
	if message != string(decoded) {
		t.Fatalf("Huffman encoding mismatch: got %s - expected %s", string(decoded), message)
	}
}

func TestHuffmanEncodeDecode(t *testing.T) {
	message := "Hello World!"
	HuffmanEncodeDecode(message, t)
}

func TestHuffmanMaxRunLength(t *testing.T) {
	message := "abccdddeeeeeffffffff" +
		strings.Repeat("g", 13) +
		strings.Repeat("h", 21) +
		strings.Repeat("i", 34) +
		strings.Repeat("j", 55) +
		strings.Repeat("k", 89)
	HuffmanEncodeDecode(message, t)
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
