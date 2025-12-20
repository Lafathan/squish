package bitio

import (
	"io"
	"strings"
	"testing"
)

func TestReadBits(t *testing.T) {
	// test for reading in bits from a stream
	// stream is the input 'Hello World!'
	// it is compared to the known bit values of the stream 6 bits at a time
	reader := strings.NewReader("Hello World!")
	bitReader := NewBitReader(reader)
	ans := [15]uint64{18, 6, 21, 44, 27, 6, 60, 32, 21, 54, 61, 50, 27, 6, 16}
	for i, want := range ans {
		got, err := bitReader.ReadBits(6)
		if err != nil {
			t.Fatalf("Unexpected error at index %d: %v", i, err)
		}
		if got != want {
			t.Errorf("Mismatch at index %d: got %d, want %d", i, got, want)
		}
	}
	// 8 bits is requested over the last 6 to verify no EOS reading
	_, err := bitReader.ReadBits(8)
	if err == nil {
		t.Fatalf("Read past EOS")
	}
}

func TestShortBuffer(t *testing.T) {
	// test that the BitReader returns as error when trying
	// to read too many bytes at once
	reader := strings.NewReader("Hello World!")
	bitReader := NewBitReader(reader)
	_, err := bitReader.ReadBits(65)
	if err != io.ErrShortBuffer {
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
