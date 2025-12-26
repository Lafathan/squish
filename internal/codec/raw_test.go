package codec

import "testing"

func TestRAWEncodeDecode(t *testing.T) {
	message := "Hello World!"
	raw := RAWCodec{}
	coded, pad, err := raw.EncodeBlock([]byte(message))
	if err != nil {
		t.Fatalf("RAW encoding failed")
	}
	decoded, err := raw.DecodeBlock(coded, pad)
	if err != nil {
		t.Fatalf("RAW Decoding failed")
	}
	if message != string(decoded) {
		t.Fatalf("Raw encoding mismatch: got %s - expected %s", string(decoded), message)
	}
}

func TestRAWLossless(t *testing.T) {
	raw := RAWCodec{}
	if !raw.IsLossless() {
		t.Fatalf("RAW is lossless, but returned lossy")
	}
}
