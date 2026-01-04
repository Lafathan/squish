package frame

import (
	"io"
	"squish/internal/codec"
	"strings"
	"testing"
)

var payloadStr string = "Hello World!"

func TestWriteRead(t *testing.T) {
	headers := []Header{
		{Key: MagicKey, Codec: []uint8{codec.RAW}, ChecksumMode: NoChecksum},
		{Key: MagicKey, Codec: []uint8{codec.RAW}, ChecksumMode: UncompressedChecksum},
		{Key: MagicKey, Codec: []uint8{codec.RAW}, ChecksumMode: CompressedChecksum},
		{Key: MagicKey, Codec: []uint8{codec.RAW}, ChecksumMode: UncompressedChecksum | CompressedChecksum},
	}
	blocks := []Block{
		{BlockType: DefaultCodec, USize: 12, CSize: 12, Checksum: 0},
		{BlockType: BlockCodec, Codec: []uint8{codec.RAW}, USize: 12, CSize: 12, Checksum: 75},
		{BlockType: DefaultCodec, USize: 12, CSize: 12, Checksum: 170},
		{BlockType: DefaultCodec, USize: 12, CSize: 12, Checksum: 345},
	}
	for i, h := range headers {
		var str strings.Builder
		fw := NewFrameWriter(io.Writer(&str), h)
		err := fw.Ready()
		if err != nil {
			t.Fatalf("Failed to ready FrameWriter %s: %v", fw, err)
		}
		testBlocks := []Block{blocks[i], blocks[i]}
		for i, b := range testBlocks {
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
		if !fr.Header.Equal(h) {
			t.Fatalf("Header mismatch in WriteRead test %d\n%s\n%s", i, h, fr.Header)
		}
		for i := range len(testBlocks) - 1 {
			block, payloadReader, err := fr.Next()
			if err != nil {
				t.Fatalf("Failed to read block %d: %v", i, err)
			}
			if !block.Equal(testBlocks[i]) {
				t.Fatalf("Mismatch in header of block %d, %s", i, block)
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
			t.Fatalf("Failed to read block %d: %v", len(testBlocks), err)
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
		b, _, err := fr.Next()
		if b.BlockType != EOS {
			t.Fatalf("Read blocktype %d, expected EOS", b.BlockType)
		}
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
