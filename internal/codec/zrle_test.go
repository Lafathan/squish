package codec

import (
	"bytes"
	"testing"
)

func ZRLEEncodeDecode(message string, t *testing.T) {
	lc := ZRLECodec{}
	coded, err := lc.EncodeBlock([]byte(message))
	if err != nil {
		t.Fatalf("ZRLE encoding failed: %v", err)
	}
	decoded, err := lc.DecodeBlock(coded)
	if err != nil {
		t.Fatalf("ZRLE decoding failed: %v", err)
	}
	if message != string(decoded) {
		t.Fatalf("ZRLE encoding mismatch: got %s - expected %s", string(decoded), message)
	}
}

func TestZRLEEncodeDecode(t *testing.T) {
	message := "The mellow yellow fellow says hello world!"
	ZRLEEncodeDecode(message, t)
}

func TestZRLERunLength(t *testing.T) {
	message := []byte{0, 1}
	a, b := 1, 1
	for i := range 4 {
		a, b = b, a+b
		message = append(message, bytes.Repeat([]byte{byte(i)}, b)...)
	}
	ZRLEEncodeDecode(string(message), t)
}

func TestZRLEEmptyMessage(t *testing.T) {
	message := ""
	ZRLEEncodeDecode(message, t)
}

func TestZRLELossless(t *testing.T) {
	lc := ZRLECodec{}
	if !lc.IsLossless() {
		t.Fatalf("ZRLE is lossless, but returned lossy")
	}
}
