package bitio

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"strings"
	"testing"
)

func TestReadWrite(t *testing.T) {
	// make a random source so that you pull random numbers of bits from the string
	src := rand.NewSource(10)
	r := rand.New(src)
	input := "Hello World!"
	reader := strings.NewReader(input)
	bitReader := NewBitReader(reader)
	var str strings.Builder
	bitWriter := NewBitWriter(&str)
	bitsLeft := uint8(8 * len(input))
	for bitsLeft > 0 {
		bitLength := uint8(r.Intn(10))
		bitLength = min(bitLength, bitsLeft)
		bits, err := bitReader.ReadBits(bitLength)
		bitStart := uint8(8*len(input)) - bitsLeft
		bitEnd := bitStart + bitLength
		if err != nil {
			t.Fatalf("Failed to read bits %d to %d: %v", bitStart, bitEnd, err)
		}
		err = bitWriter.WriteBits(bits, bitLength)
		if err != nil {
			t.Fatalf("Failed to write bits %d to %d: %v", bitStart, bitEnd, err)
		}
		bitsLeft -= bitLength
	}
	_, err := bitWriter.Flush()
	if err != nil {
		t.Fatalf("Flush error: %v", err)
	}
	if str.String() != input {
		t.Fatalf("String '%s' does not match expected '%s'", str.String(), input)
	}
}

func TestShortBuffer(t *testing.T) {
	// test that the BitReader returns as error when trying
	// to read too many bytes at once
	reader := strings.NewReader("Hello World!")
	bitReader := NewBitReader(reader)
	_, err := bitReader.ReadBits(65)
	if errors.Is(err, io.ErrShortBuffer) {
		t.Fatalf("Failed short buffer check")
	}
}

func TestReadFullError(t *testing.T) {
	// test returning when the io.ReadFull returns an error
	reader := strings.NewReader("Hello World!")
	bitReader := NewBitReader(reader)
	_, err := bitReader.ReadBits(60)
	_, err = bitReader.ReadBits(37)
	if err == nil {
		t.Fatalf("Read past EOS")
	}
}

func TestFlush(t *testing.T) {
	// test that the flush command pads the bits to the nearest byte size
	buf := new(bytes.Buffer)
	bw := NewBitWriter(buf)
	err := bw.WriteBits(1, 6)
	if err != nil {
		t.Fatalf("Failed to write bits before testing flush: %v", err)
	}
	n, err := bw.Flush()
	if err != nil {
		t.Fatalf("Failed during flush of bitWriter: %v", err)
	}
	if n != 2 {
		t.Fatalf("Padded unexpected amount during flush")
	}
}
