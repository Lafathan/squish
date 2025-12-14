package bitio

import (
	"io"
	"strings"
	"testing"
)

func TestWriteBits(t *testing.T) {
	// test for writing bits to a stream that will build a string
	// stream is input of 64 bit integers that we only want the last 6 bits of
	inputStr := "Hello World!"
	var str strings.Builder
	writer := io.Writer(&str)
	bitWriter := NewBitWriter(writer)
	ans := [16]uint64{18, 6, 21, 44, 27, 6, 60, 32, 21, 54, 61, 50, 27, 6, 16, 33}
	for i, want := range ans {
		err := bitWriter.WriteBits(want, 6)
		if err != nil {
			t.Fatalf("Unexpected error at index %d: %v", i, err)
		}
	}
	err := bitWriter.Flush()
	if err != nil {
		t.Fatalf("Flush error: %v", err)
	}
	// the built string is then compared to the known ascii value of the bit stream
	res := str.String()
	if res != inputStr {
		t.Fatalf("String '%s' does not match expected '%s'", res, inputStr)
	}
}
