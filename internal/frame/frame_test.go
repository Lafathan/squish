package frame

import (
	"io"
	"squish/internal/codec"
	"strings"
	"testing"
)

var h Header = Header{Key: MagicKey, Codec: codec.RAW, ChecksumMode: UncompressedChecksum | CompressedChecksum}

var b0 Block = Block{BlockType: DefaultCodec, USize: 12, CSize: 12, PadBits: 0}
var b1 Block = Block{BlockType: BlockCodec, Codec: codec.RAW, USize: 12, CSize: 12, PadBits: 0, Checksum: 75}
var b2 Block = Block{BlockType: DefaultCodec, USize: 12, CSize: 12, PadBits: 0, Checksum: 170}
var b3 Block = Block{BlockType: DefaultCodec, USize: 12, CSize: 12, Checksum: 345}
var b4 Block = Block{BlockType: DefaultCodec, USize: 12, CSize: 12, Checksum: 0}

var blocks []Block = []Block{b0, b1, b2, b3, b4}

var payloadStr string = "Hello World!"

func TestWriteRead(t *testing.T) {
	var str strings.Builder
	fw := NewFrameWriter(io.Writer(&str), h)
	err := fw.Ready()
	if err != nil {
		t.Fatalf("Failed to ready FrameWriter: %v", err)
	}
	for i, b := range blocks {
		payload := strings.NewReader(payloadStr)
		err = fw.WriteBlock(b, payload)
		if err != nil {
			t.Fatalf("Failed writing block %d: %v", i, err)
		}
	}
	fw.Close()
	fr := NewFrameReader(strings.NewReader(str.String()))
	err = fr.Ready()
	if err != nil {
		t.Fatalf("Failed to ready FrameReader: %v", err)
	}
	if fr.Header != h {
		t.Fatalf("Header mismatch in WriteRead test\n%s\n%s", h, fr.Header)
	}
	for i := range len(blocks) - 1 {
		block, payloadReader, err := fr.Next()
		if err != nil {
			t.Fatalf("Failed to read block %d: %v", i, err)
		}
		if block != blocks[i] {
			t.Fatalf("Mismatch in header of block %d", i)
		}
		bytes, err := io.ReadAll(payloadReader)
		if err != nil {
			t.Fatalf("Failed to read payload in block %d: %v", i, err)
		}
		if string(bytes) != payloadStr {
			t.Fatalf("Mismatch in payload data in block %d", i)
		}
	}
	_, _, err = fr.Next()
	if err != nil {
		t.Fatalf("Failed to read block %d: %v", len(blocks), err)
	}
	_, _, err = fr.Next()
	if err == nil {
		t.Fatalf("Missed early read error")
	} else if err.Error() != "early read, previous payload still active" {
		t.Fatalf("Missed early read error: %v", err)
	}
	err = fr.Drop()
	if fr.ActivePayload != nil {
		t.Fatalf("Failed to drop active payload")
	}
}

func TestHeaderValid(t *testing.T) {
	var badHeader Header
	badHeader = Header{Key: "SQSh"}
	err := badHeader.Valid()
	if err == nil {
		t.Fatalf("Missed invalid magic key: %v", err)
	}
	badHeader = Header{Key: MagicKey, ChecksumMode: 4}
	err = badHeader.Valid()
	if err == nil {
		t.Fatalf("Missed invalid maximum uncompressed size: %v", err)
	}
}

func TestBlockValid(t *testing.T) {
	var badBlock Block
	badBlock = Block{BlockType: 4}
	err := badBlock.Valid()
	if err == nil {
		t.Fatalf("Missed invalid blocktype: %v", err)
	}
	badBlock = Block{BlockType: DefaultCodec, USize: MaxBlockSize + 1}
	err = badBlock.Valid()
	if err == nil {
		t.Fatalf("Missed invalid maximum uncompressed size: %v", err)
	}
}
