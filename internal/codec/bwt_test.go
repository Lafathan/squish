package codec

import (
	"bytes"
	"testing"
)

func BWTEncodeDecode(message string, t *testing.T) {
	bc := BWTCodec{}
	coded, err := bc.EncodeBlock([]byte(message))
	if err != nil {
		t.Fatalf("BWT encoding failed: %v", err)
	}
	decoded, err := bc.DecodeBlock(coded)
	if err != nil {
		t.Fatalf("BWT decoding failed: %v", err)
	}
	if message != string(decoded) {
		t.Fatalf("BWT encoding mismatch: got %s... - expected %s...", string(decoded), message)
	}
}

func TestBWTEncodeDecode(t *testing.T) {
	message := "The mellow yellow fellow says hello world!"
	BWTEncodeDecode(message, t)
}

func TestBWTRunLength(t *testing.T) {
	message := []byte{0, 1}
	a, b := 1, 1
	for i := range 30 {
		a, b = b, a+b
		message = append(message, bytes.Repeat([]byte{byte(i)}, b)...)
	}
	BWTEncodeDecode(string(message), t)
}

func TestBWTEmptyMessage(t *testing.T) {
	message := ""
	BWTEncodeDecode(message, t)
}

func TestBWTLossless(t *testing.T) {
	lc := BWTCodec{}
	if !lc.IsLossless() {
		t.Fatalf("BWT is lossless, but returned lossy")
	}
}
