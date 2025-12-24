package pipeline

import (
	"squish/internal/codec"
	"squish/internal/frame"
	"strings"
	"testing"
)

func TestPipeline(t *testing.T) {
	message := "Hello World!"
	encodeReader := strings.NewReader(message)
	encodeWriter := new(strings.Builder)
	err := Encode(encodeReader, encodeWriter, codec.RAW, frame.MaxBlockSize, frame.CompressedChecksum)
	if err != nil {
		t.Fatalf("Pipeline error during encoding: %v", err)
	}
	decodeReader := strings.NewReader(encodeWriter.String())
	decodeWriter := new(strings.Builder)
	err = Decode(decodeReader, decodeWriter)
	if err != nil {
		t.Fatalf("Pipeline error during decoding: %v", err)
	}
	if decodeWriter.String() != message {
		t.Fatalf("Pipeline messages did not match - expected %s, got %s", message, decodeWriter.String())
	}
}
