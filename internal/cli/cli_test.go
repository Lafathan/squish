package cli

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"squish/internal/codec"
	"squish/internal/frame"
)

func TestOpenInputStdin(t *testing.T) {
	tests := []string{"", "-"}
	for _, path := range tests {
		f, closeFn, err := openInput(path)
		if err != nil {
			t.Fatalf("openInput(%q) unexpected err: %v", path, err)
		}
		if f != os.Stdin {
			t.Fatalf("openInput(%q) expected os.Stdin; got %v", path, f)
		}
		closeFn()
	}
}

func TestOpenOutputStdout(t *testing.T) {
	tests := []string{"", "-"}
	for _, path := range tests {
		f, closeFn, err := openOutput(path)
		if err != nil {
			t.Fatalf("openOutput(%q) unexpected err: %v", path, err)
		}
		if f != os.Stdout {
			t.Fatalf("openOutput(%q) expected os.Stdout; got %v", path, f)
		}
		closeFn()
	}
}

func TestOpenInputFilePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.txt")
	want := []byte("Hello world!")
	if err := os.WriteFile(path, want, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	f, closeFn, err := openInput(path)
	if err != nil {
		t.Fatalf("openInput(%q) err: %v", path, err)
	}
	defer closeFn()
	got, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll err: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("read mismatch: got %q want %q", string(got), string(want))
	}
}

func TestOpenOutputFilePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	f, closeFn, err := openOutput(path)
	if err != nil {
		t.Fatalf("openOutput(%q) err: %v", path, err)
	}
	want := []byte("encoded bytes here")
	if _, err := f.Write(want); err != nil {
		t.Fatalf("Write err: %v", err)
	}
	closeFn()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile err: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("file mismatch: got %q want %q", string(got), string(want))
	}
}

func TestParseCodecPipeline(t *testing.T) {
	var names []string
	for k := range codec.StringToCodecIDMap {
		if k != "" && k != "AUTO" {
			names = append(names, k)
		}
	}
	if len(names) < 2 {
		t.Skip("not enough codecs in StringToCodecIDMap to run basic pipeline test")
	}
	pipeline := names[0]
	for i := range len(names) - 1 {
		pipeline += "-" + names[i+1]
	}
	got, err := parseCodecPipeline(pipeline)
	if err != nil {
		t.Fatalf("parseCodecPipeline(%q) err: %v", pipeline, err)
	}
	want := []uint8{}
	for _, name := range names {
		want = append(want, codec.StringToCodecIDMap[name])
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pipeline mismatch: got %v want %v", got, want)
	}
}

func TestParseCodecPipelineUnknownCodec(t *testing.T) {
	_, err := parseCodecPipeline("HELLO-WORLD")
	if err == nil {
		t.Fatalf("expected error for unknown codec")
	}
}

func TestParseCodecPipelineEmptyCodec(t *testing.T) {
	_, err := parseCodecPipeline("-RLE")
	if err == nil {
		t.Fatalf("expected error for empty codec segment")
	}
}

func TestParseCodecPipelineAlias(t *testing.T) {
	var alias, expanded string
	for a, e := range codec.CodecAliases {
		alias, expanded = a, e
		break
	}
	if alias == "" {
		t.Skip("no aliases defined in codec.CodecAliases")
	}
	got1, err := parseCodecPipeline(alias)
	if err != nil {
		t.Fatalf("parseCodecPipeline(alias=%q) err: %v", alias, err)
	}
	got2, err := parseCodecPipeline(expanded)
	if err != nil {
		t.Fatalf("parseCodecPipeline(expanded=%q) err: %v", expanded, err)
	}
	if !reflect.DeepEqual(got1, got2) {
		t.Fatalf("alias expansion mismatch: alias %q -> got %v, expanded %q -> got %v",
			alias, got1, expanded, got2)
	}
}

func TestParseCodecPipelineAuto(t *testing.T) {
	got, err := parseCodecPipeline("RLE-AUTO-HUFFMAN")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := []uint8{codec.AUTO}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AUTO collapse mismatch: got %v want %v", got, want)
	}
}

func TestParseBlockSizeValid(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"1B", 1},
		{"1KiB", 1 << 10},
		{"1MiB", 1 << 20},
		{"3KB", 3_000},
		{"4MB", 4_000_000},
	}
	for _, tc := range tests {
		got, err := parseBlockSize(tc.in)
		if err != nil {
			t.Fatalf("parseBlockSize(%q) err: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("parseBlockSize(%q) got %d want %d", tc.in, got, tc.want)
		}
	}
}

func TestParseBlockSizeInvalid(t *testing.T) {
	tests := []string{
		"",
		"10",
		"0KB",
		"-1KB",
		"1GiB",
		"1kib",
		"abcKB",
		"1 2KB",
	}

	for _, in := range tests {
		_, err := parseBlockSize(in)
		if err == nil {
			t.Fatalf("parseBlockSize(%q) expected error", in)
		}
	}
}

func TestParseChecksumValid(t *testing.T) {
	tests := []struct {
		in   string
		want byte
	}{
		{"", frame.NoChecksum},
		{"u", frame.UncompressedChecksum},
		{"c", frame.CompressedChecksum},
		{"uc", frame.UncompressedChecksum | frame.CompressedChecksum},
		{"U", frame.UncompressedChecksum},
		{"C", frame.CompressedChecksum},
		{"uc", frame.UncompressedChecksum | frame.CompressedChecksum},
	}

	for _, tc := range tests {
		got, err := parseChecksum(tc.in)
		if err != nil {
			t.Fatalf("parseChecksum(%q) err: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("parseChecksum(%q) got %d want %d", tc.in, got, tc.want)
		}
	}
}

func TestParseChecksumInvalid(t *testing.T) {
	tests := []string{"x", "uu", "ucu", "-", "cu"}

	for _, in := range tests {
		_, err := parseChecksum(in)
		if err == nil {
			t.Fatalf("parseChecksum(%q) expected error", in)
		}
	}
}
