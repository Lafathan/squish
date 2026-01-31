package codec

import (
	"bytes"
	"testing"
)

func AUTOEncodeDecode(message string, t *testing.T) {
	var err error
	ac := AUTOCodec{}
	coded, err := ac.EncodeBlock([]byte(message))
	if err != nil {
		t.Fatalf("AUTO encoding failed: %v", err)
	}
	decoded := coded
	for i := len(ac.CodecIDs) - 1; i >= 0; i-- {
		decoded, err = CodecMap[uint8(ac.CodecIDs[i])].DecodeBlock(decoded)
		if err != nil {
			t.Fatalf("AUTO decoding failed on codec ID %d: %v", ac.CodecIDs[i], err)
		}
	}
	if message != string(decoded) {
		t.Fatalf("AUTO encoding mismatch: got %s - expected %s", string(decoded), message)
	}
}

func TestAUTOEncodeDecode(t *testing.T) {
	message := "The mellow yellow fellow says hello world!"
	AUTOEncodeDecode(message, t)
}

func TestAUTORunLength(t *testing.T) {
	message := []byte{0, 1}
	a, b := 1, 1
	for i := range 30 {
		a, b = b, a+b
		message = append(message, bytes.Repeat([]byte{byte(i)}, b)...)
	}
	AUTOEncodeDecode(string(message), t)
}

func TestAUTOEmptyMessage(t *testing.T) {
	message := ""
	AUTOEncodeDecode(message, t)
}

func TestAUTOLossless(t *testing.T) {
	ac := AUTOCodec{}
	if !ac.IsLossless() {
		t.Fatalf("AUTO is lossless, but returned lossy")
	}
}
