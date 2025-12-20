package frame

import (
	"io"
	"strings"
	"testing"
)

func TestWriteBlock(t *testing.T) {
	inputStr := "Hello World!"
	var str strings.Builder
	writer := io.Writer(&str)
	h := Header{
		Key:          MagicKey,
		Flags:        0x00,
		Codec:        RAW,
		ChecksumMode: NoGlobalChecksum,
	}
	fw := NewFrameWriter(writer, h)
	error := fw.Ready()
	if error != nil {
		t.Fatalf("Failed to ready the FrameWriter")
	}
	b := Block{
		BlockType:      DefaultCodec,
		USize:          12,
		CSize:          12,
		PadBits:        0,
		ChecksumMethod: NoCheckSum,
	}
	error = fw.WriteBlock(b, []byte(inputStr))
	expected := []byte{}
	expected = append(expected, []byte(MagicKey)...)
	expected = append(expected, []byte{0x00, RAW, NoGlobalChecksum, DefaultCodec, 0x0C, 0x0C, 0x00, NoCheckSum}...)
	expected = append(expected, []byte(inputStr)...)
	if str.String() != string(expected) {
		t.Fatalf("Mismatch in written block: %s -> %s", str.String(), expected)
	}
}
