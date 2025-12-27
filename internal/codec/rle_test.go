package codec

import (
	"strings"
	"testing"
)

func EncodeDecode(message string, t *testing.T) {
	rle := RLECodec{}
	coded, pad, err := rle.EncodeBlock([]byte(message))
	if err != nil {
		t.Fatalf("RLE encoding failed")
	}
	decoded, err := rle.DecodeBlock(coded, pad)
	if err != nil {
		t.Fatalf("RLE decoding failed")
	}
	if message != string(decoded) {
		t.Fatalf("Rle encoding mismatch: got %s - expected %s", string(decoded), message)
	}
}

func TestRLEEncodeDecode(t *testing.T) {
	message := "Hello World! Whooooooooooooooohhhhhhhhhhhhhooooooooooooooooo!"
	EncodeDecode(message, t)
}

func TestRLEMaxRunLength(t *testing.T) {
	message := "aaa" + strings.Repeat("b", 300) + "cccc"
	EncodeDecode(message, t)
}

func TestEmptyMessage(t *testing.T) {
	message := ""
	EncodeDecode(message, t)
}

func TestRLELossless(t *testing.T) {
	rle := RLECodec{}
	if !rle.IsLossless() {
		t.Fatalf("RLE is lossless, but returned lossy")
	}
}
