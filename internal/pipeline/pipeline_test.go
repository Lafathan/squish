package pipeline

import (
	"squish/internal/codec"
	"squish/internal/frame"
	"strings"
	"testing"
)

func testHelper(t *testing.T, str string, codecID uint8, blockSize uint64, checksumMode uint8) {
	encodeReader := strings.NewReader(str)
	encodeWriter := new(strings.Builder)
	err := Encode(encodeReader, encodeWriter, codecID, blockSize, checksumMode)
	if err != nil {
		t.Fatalf("Pipeline error during encoding: %v", err)
	}
	decodeReader := strings.NewReader(encodeWriter.String())
	decodeWriter := new(strings.Builder)
	err = Decode(decodeReader, decodeWriter)
	if err != nil {
		t.Fatalf("Pipeline error during decoding: %v", err)
	}
	if decodeWriter.String() != str {
		t.Fatalf("Pipeline messages did not match - expected %s, got %s", str, decodeWriter.String())
	}
}

func TestPipelineCompChecksum(t *testing.T) {
	message := "Hello World!"
	testHelper(t, message, codec.RAW, frame.MaxBlockSize, frame.CompressedChecksum)
}

func TestPipelineUncompChecksumSmallBlockSize(t *testing.T) {
	message := "Hello World!"
	testHelper(t, message, codec.RAW, 6, frame.UncompressedChecksum)
}

func TestPipelineChecksumLargeBlockSize(t *testing.T) {
	message := "Hello World!"
	testHelper(t, message, codec.RAW, frame.MaxBlockSize+99, frame.CompressedChecksum|frame.UncompressedChecksum)
}
