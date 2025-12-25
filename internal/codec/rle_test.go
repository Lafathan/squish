package codec

import "testing"

func TestRLEEncode(t *testing.T) {
	message := "Hello World! Whooooooooooooooohhhhhhhhhhhhhooooooooooooooooo!"
	rle := RLECodec{}
	out, _, err := rle.EncodeBlock([]byte(message))
	if err != nil {
		t.Fatalf("RLE encoding failed")
	}
	if len(out) != 36 {
		t.Fatalf("Unexpected RLE encoding output length: got %d - expected 36", len(out))
	}
}

func TestRLELossless(t *testing.T) {
	rle := RLECodec{}
	if !rle.IsLossless() {
		t.Fatalf("RLE is lossless, but returned lossy")
	}
}
