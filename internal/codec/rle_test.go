package codec

import (
	"strings"
	"testing"
)

func RLEEncodeDecode(message string, t *testing.T) {
	rle := RLECodec{}
	coded, err := rle.EncodeBlock([]byte(message))
	if err != nil {
		t.Fatalf("RLE encoding failed")
	}
	decoded, err := rle.DecodeBlock(coded)
	if err != nil {
		t.Fatalf("RLE decoding failed")
	}
	if message != string(decoded) {
		t.Fatalf("RLE encoding mismatch: got %s - expected %s", string(decoded), message)
	}
}

func TestRLEEncodeDecode(t *testing.T) {
	message := "Hello World!"
	RLEEncodeDecode(message, t)
}

func TestRLEMaxRunLength(t *testing.T) {
	message := "abccdddeeeeeffffffff" +
		strings.Repeat("g", 13) +
		strings.Repeat("h", 21) +
		strings.Repeat("i", 34) +
		strings.Repeat("j", 55) +
		strings.Repeat("k", 89)
	RLEEncodeDecode(message, t)
}

func TestRLEEmptyMessage(t *testing.T) {
	message := ""
	RLEEncodeDecode(message, t)
}

func TestInvalidMessageLength(t *testing.T) {
	rle := RLECodec{}
	_, err := rle.DecodeBlock([]byte{1, 1, 2, 2, 3})
	if err == nil {
		t.Fatalf("Missed invalid RLE length")
	}
}

func TestRLELossless(t *testing.T) {
	rle := RLECodec{}
	if !rle.IsLossless() {
		t.Fatalf("RLE is lossless, but returned lossy")
	}
}
