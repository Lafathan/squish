package bitio

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

func TestReadWrite(t *testing.T) {
	// make a random source so that you pull random numbers of bits from the string
	src := rand.NewSource(10)
	r := rand.New(src)
	input := "Hello World!"
	println("=========================")
	fmt.Printf("String: %08b", []byte(input))
	println()
	println("=========================")
	reader := strings.NewReader(input)
	bitReader := NewBitReader(reader)
	var str strings.Builder
	bitWriter := NewBitWriter(&str)
	bitsLeft := 8 * len(input)
	for bitsLeft > 0 {
		bitLength := min(r.Intn(20), bitsLeft)
		bits, err := bitReader.ReadBits(bitLength)
		println()
		fmt.Printf("Read %d bits: %08b", bitLength, bits)
		println()
		fmt.Printf("      - read buffer (%d bits): %08b", bitReader.Nbits, bitReader.Buffer)
		println()
		bitStart := 8*len(input) - bitsLeft
		bitEnd := bitStart + bitLength
		if err != nil {
			t.Fatalf("Failed to read bits %d to %d: %v", bitStart, bitEnd, err)
		}
		println()
		err = bitWriter.WriteBits(bits, bitLength)
		fmt.Printf("      - write buffer (%d bits): %08b", bitWriter.Nbits, bitWriter.Buffer)
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
	err := bw.WriteBits([]byte{1}, 6)
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
