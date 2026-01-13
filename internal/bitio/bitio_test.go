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
	bitsLeft := 8 * len(input)
	for bitsLeft > 0 {
		bitLength := min(r.Intn(20), bitsLeft)
		bits, err := bitReader.ReadBits(bitLength)
		bitStart := 8*len(input) - bitsLeft
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

func TestReadTooLarge(t *testing.T) {
	// test trying to read >64 bits
	reader := strings.NewReader("Hello World!")
	bitReader := NewBitReader(reader)
	_, err := bitReader.ReadBits(65)
	if !errors.Is(err, io.ErrShortBuffer) {
		t.Fatalf("Tried to read more bytes than bitReader buffer can hold")
	}
}

func TestReadFull64BitMask(t *testing.T) {
	// test reading in 64 bits
	reader := strings.NewReader("Hello World!")
	bitReader := NewBitReader(reader)
	data, err := bitReader.ReadBits(64)
	if err != nil {
		t.Fatalf("Error when reading 64 bits: %v", err)
	}
	val := uint64(0)
	for _, b := range []byte("Hello Wo") {
		val = (val << 8) | uint64(b)
	}
	if val != data {
		t.Fatalf("Mismatched result in 64BitMask test")
	}
}

func TestFlush(t *testing.T) {
	// test that the flush command pads the bits to the nearest byte size
	buf := new(bytes.Buffer)
	bw := NewBitWriter(buf)
	err := bw.WriteBits(0b00000001, 6)
	if err != nil {
		t.Fatalf("Failed to write bits before testing flush: %v", err)
	}
	n, err := bw.Flush()
	if err != nil {
		t.Fatalf("Failed during flush of bitWriter: %v", err)
	}
	if n != 2 {
		t.Fatalf("Padded unexpected amount during byte-misaligned flush")
	}
	n, err = bw.Flush()
	if err != nil {
		t.Fatalf("Failed during flush of empty bitWriter: %v", err)
	}
	if n != 0 {
		t.Fatalf("Padded unexpected amount during byte-aligned flush")
	}
}

func TestWriteBitsClearsFullBuffer(t *testing.T) {
	buf := new(bytes.Buffer)
	bw := NewBitWriter(buf)
	err := bw.WriteBits(0x0102030405060708, 64)
	if err != nil {
		t.Fatalf("Failed to write initial 64 bits: %v", err)
	}
	err = bw.WriteBits(0xAA, 8)
	if err != nil {
		t.Fatalf("Failed to write trailing byte: %v", err)
	}
	_, err = bw.Flush()
	if err != nil {
		t.Fatalf("Failed to flush after full buffer write: %v", err)
	}
	expected := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0xAA}
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Fatalf("Unexpected buffer contents: %v", buf.Bytes())
	}
}

func TestReadBitsShortRead(t *testing.T) {
	reader := strings.NewReader("A")
	bitReader := NewBitReader(reader)
	_, err := bitReader.ReadBits(16)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("Expected unexpected EOF, got %v", err)
	}
}

func TestWriteBitsSpansBuffer(t *testing.T) {
	buf := new(bytes.Buffer)
	bw := NewBitWriter(buf)
	err := bw.WriteBits(0xABC, 12)
	if err != nil {
		t.Fatalf("Failed to write initial 12 bits: %v", err)
	}
	err = bw.WriteBits(0x0123456789ABCDE, 60)
	if err != nil {
		t.Fatalf("Failed to write 60 bits: %v", err)
	}
	_, err = bw.Flush()
	if err != nil {
		t.Fatalf("Failed to flush: %v", err)
	}
	br := NewBitReader(bytes.NewReader(buf.Bytes()))
	first, err := br.ReadBits(12)
	if err != nil {
		t.Fatalf("Failed to read first 12 bits: %v", err)
	}
	second, err := br.ReadBits(60)
	if err != nil {
		t.Fatalf("Failed to read next 60 bits: %v", err)
	}
	if first != 0xABC || second != 0x0123456789ABCDE {
		t.Fatalf("Unexpected round trip values: %x %x", first, second)
	}
}
