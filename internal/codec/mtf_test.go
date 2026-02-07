package codec

import (
	"bytes"
	"testing"
)

func MTFEncodeDecode(message string, t *testing.T) {
	lc := MTFCodec{}
	coded, err := lc.EncodeBlock([]byte(message))
	if err != nil {
		t.Fatalf("MTF encoding failed: %v", err)
	}
	decoded, err := lc.DecodeBlock(coded)
	if err != nil {
		t.Fatalf("MTF decoding failed: %v", err)
	}
	if message != string(decoded) {
		t.Fatalf("MTF encoding mismatch: got %s - expected %s", string(decoded), message)
	}
}

func TestMTFEncodeDecode(t *testing.T) {
	message := "The mellow yellow fellow says hello world!"
	MTFEncodeDecode(message, t)
}

func TestMTFRunLength(t *testing.T) {
	message := []byte{0, 1}
	a, b := 1, 1
	for i := range 30 {
		a, b = b, a+b
		message = append(message, bytes.Repeat([]byte{byte(i)}, b)...)
	}
	MTFEncodeDecode(string(message), t)
}

func TestMTFEmptyMessage(t *testing.T) {
	message := ""
	MTFEncodeDecode(message, t)
}

func TestMTFLossless(t *testing.T) {
	lc := MTFCodec{}
	if !lc.IsLossless() {
		t.Fatalf("MTF is lossless, but returned lossy")
	}
}
