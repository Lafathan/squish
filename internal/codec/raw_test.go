package codec

import "testing"

func TestRAWEncode(t *testing.T) {
	message := "Hello World!"
	raw := RAWCodec{}
	out, _, err := raw.EncodeBlock([]byte(message))
	if err != nil {
		t.Fatalf("RAW encoding failed")
	}
	if message != string(out) {
		t.Fatalf("Raw encoding mismatch: got %s - expected %s", string(out), message)
	}
}

func TestRAWLossless(t *testing.T) {
	raw := RAWCodec{}
	if !raw.IsLossless() {
		t.Fatalf("RAW is lossless, but returned lossy")
	}
}
