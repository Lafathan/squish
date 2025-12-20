package frame

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func TestReadBlock(t *testing.T) {
	expected := "Hello World!"
	input := []byte("SQSH")
	input = append(input, []byte{0x00, RAW, NoGlobalChecksum}...)
	input = append(input, []byte{DefaultCodec, 0x0C, 0x0C, 0x00, NoCheckSum}...)
	input = append(input, []byte(expected)...)
	buf := bytes.NewBuffer(input)
	fr := NewFrameReader(buf)
	err := fr.Ready()
	if err != nil {
		t.Fatalf("Failed to ready FrameReader")
	}
	header := Header{
		Key:          MagicKey,
		Flags:        0x00,
		Codec:        RAW,
		ChecksumMode: NoGlobalChecksum,
	}
	block := Block{
		BlockType:      DefaultCodec,
		USize:          12,
		CSize:          12,
		PadBits:        0,
		ChecksumMethod: NoCheckSum,
	}
	if fr.Header != header {
		fmt.Printf("%v", fr.Header)
		t.Fatalf("Mismatch in read header")
	}
	b, payloadReader, err := fr.Next()
	if err != nil {
		fmt.Printf("%v", b)
		t.Fatalf("Failed to read block")
	}
	if b != block {
		fmt.Printf("%v", b)
		t.Fatalf("Mismatch in block header")

	}
	bytes, err := io.ReadAll(payloadReader)
	if string(bytes) != expected {
		t.Fatalf("Mismatch in read data")
	}
}
