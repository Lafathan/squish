package frame

import (
	"io"
	"strings"
	"testing"
)

var h Header = Header{Key: MagicKey, Flags: 0x00, Codec: RAW, ChecksumMode: UncompressedChecksum | CompressedChecksum}

var b0 Block = Block{BlockType: DefaultCodec, USize: 12, CSize: 12, PadBits: 0}
var b1 Block = Block{BlockType: BlockCodec, Codec: RAW, USize: 12, CSize: 12, PadBits: 0, Checksum: 75}
var b2 Block = Block{BlockType: DefaultCodec, USize: 12, CSize: 12, PadBits: 0, Checksum: 170}
var b3 Block = Block{BlockType: DefaultCodec, USize: 12, CSize: 12, Checksum: 345}

var blocks []Block = []Block{b0, b1, b2, b3}

var payload string = "Hello World!"

func TestWriteRead(t *testing.T) {
	var str strings.Builder
	fw := NewFrameWriter(io.Writer(&str), h)
	err := fw.Ready()
	if err != nil {
		t.Fatalf("Failed to ready FrameWriter")
	}
	for i, b := range blocks {
		err = fw.WriteBlock(b, []byte(payload))
		if err != nil {
			t.Fatalf("Failed writing block %d", i)
		}
	}
	fw.Close()
	fr := NewFrameReader(strings.NewReader(str.String()))
	err = fr.Ready()
	if err != nil {
		t.Fatalf("Failed to ready FrameReader")
	}
	if fr.Header != h {
		t.Fatalf("Header mismatch in WriteRead test\n%s\n%s", h, fr.Header)
	}
	for i := range len(blocks) {
		block, payloadReader, err := fr.Next()
		if err != nil {
			t.Fatalf("Failed to read block %d", i)
		}
		if block != blocks[i] {
			t.Fatalf("Mismatch in header of block %d", i)
		}
		bytes, err := io.ReadAll(payloadReader)
		if err != nil {
			t.Fatalf("Failed to read payload in block %d", i)
		}
		if string(bytes) != payload {
			t.Fatalf("Mismatch in payload data in block %d", i)
		}
	}
}

func TestHeaderValid(t *testing.T) {
	var badHeader Header
	badHeader = Header{Key: "SQSh"}
	err := badHeader.Valid()
	if err == nil {
		t.Fatalf("Missed invalid magic key")
	}
	badHeader = Header{Key: MagicKey, ChecksumMode: 4}
	err = badHeader.Valid()
	if err == nil {
		t.Fatalf("Missed invalid maximum uncompressed size")
	}
}

func TestBlockValid(t *testing.T) {
	var badBlock Block
	badBlock = Block{BlockType: 4}
	err := badBlock.Valid()
	if err == nil {
		t.Fatalf("Missed invalid blocktype")
	}
	badBlock = Block{BlockType: DefaultCodec, USize: MaxBlockSize + 1}
	err = badBlock.Valid()
	if err == nil {
		t.Fatalf("Missed invalid maximum uncompressed size")
	}
}
