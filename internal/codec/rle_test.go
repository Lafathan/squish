package codec

import (
	"strings"
	"testing"
)

func RLENEncodeDecode(message string, t *testing.T) {
	for i := 1; i < 5; i++ {
		for _, l := range []bool{true, false} {
			rle := RLECodec{byteLength: i, lossless: l}
			coded, err := rle.EncodeBlock([]byte(message))
			if err != nil {
				t.Fatalf("RLE encoding failed")
			}
			decoded, err := rle.DecodeBlock(coded)
			if err != nil {
				t.Fatalf("RLE decoding failed")
			}
			if message != string(decoded) && rle.lossless {
				t.Fatalf("RLE encoding mismatch: got %s - expected %s", string(decoded), message)
			}
		}
	}
}

func TestRLENEncodeDecode(t *testing.T) {
	message := "Hello World!"
	RLENEncodeDecode(message, t)
}

func TestRLENMaxRunLength(t *testing.T) {
	message := "abccdddeeeeeffffffff" +
		strings.Repeat("g", 13) +
		strings.Repeat("h", 21) +
		strings.Repeat("i", 34) +
		strings.Repeat("j", 55) +
		strings.Repeat("k", 89)
	RLENEncodeDecode(message, t)
}

func TestRLENShortMessage(t *testing.T) {
	message := "Hi"
	RLENEncodeDecode(message, t)
}

func TestRLENEmptyMessage(t *testing.T) {
	message := ""
	RLENEncodeDecode(message, t)
}

func TestRLENLossless(t *testing.T) {
	rle := RLECodec{lossless: true}
	if !rle.IsLossless() {
		t.Fatalf("RLE is lossless, but returned lossy")
	}
	rle = RLECodec{lossless: false}
	if rle.IsLossless() {
		t.Fatalf("LRLE is lossy, but returned lossless")
	}
}
