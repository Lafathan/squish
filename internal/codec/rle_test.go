package codec

import (
	"testing"
)

func TestRLEEncodeDecode(t *testing.T) {
	message := "Hello World! Whooooooooooooooohhhhhhhhhhhhhooooooooooooooooo!"
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

func TestRLELossless(t *testing.T) {
	rle := RLECodec{}
	if !rle.IsLossless() {
		t.Fatalf("RLE is lossless, but returned lossy")
	}
}
