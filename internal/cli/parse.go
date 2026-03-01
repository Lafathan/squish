package cli

import (
	"fmt"
	"os"
	"slices"
	"squish/internal/codec"
	"squish/internal/frame"
	"squish/internal/sqerr"
	"strconv"
	"strings"
)

const (
	stdInOutKey   = "-"
	pipelineSplit = "-"
)

func openInput(path string) (*os.File, func(), error) {
	// open an input or fall back to Stdin
	if path == "" || path == stdInOutKey {
		return os.Stdin, func() {}, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, func() {}, err
	}
	closeFn := func() {
		_ = f.Close()
	}
	return f, closeFn, nil
}

func openOutput(path string) (*os.File, func(), error) {
	// open an output or fall back to Stdout
	if path == "" || path == stdInOutKey {
		return os.Stdout, func() {}, nil
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, func() {}, err
	}
	closeFn := func() {
		_ = f.Close()
	}
	return f, closeFn, nil
}

func parseCodecPipeline(pipeline string) ([]uint8, error) {
	// parse a string of codec keys seperated by "-" into a slice of codecID values
	pipeline = strings.ToUpper(pipeline)
	for alias, expandedCodecs := range codec.CodecAliases {
		pipeline = strings.ReplaceAll(pipeline, alias, expandedCodecs)
	}
	codecStrings := strings.Split(pipeline, pipelineSplit)
	codecList := make([]uint8, 0, len(codecStrings))
	for _, cString := range codecStrings {
		if cString == "" {
			return codecList, sqerr.New(sqerr.Usage, "empty codec in pipeline")
		}
		codecID, ok := codec.StringToCodecIDMap[strings.ToUpper(cString)]
		if !ok {
			return codecList, sqerr.New(sqerr.Unsupported, fmt.Sprintf("unknown codec %q", cString))
		}
		codecList = append(codecList, codecID)
	}
	if slices.Contains(codecList, codec.AUTO) {
		codecList = []uint8{codec.AUTO}
	}
	return codecList, nil
}

func parseBlockSize(size string) (int, error) {
	// parse a string of blocksize into a value of byte length
	var (
		matched       bool = false
		blockByteSize int  = 0
		units              = [5]string{"KiB", "MiB", "KB", "MB", "B"}
		mags               = [5]int{1 << 10, 1 << 20, 1000, 1000000, 1}
	)
	for i := range len(units) {
		prefix, found := strings.CutSuffix(size, units[i])
		if found {
			val, err := strconv.Atoi(prefix)
			if err != nil || val <= 0 {
				return 0, sqerr.New(sqerr.Usage, fmt.Sprintf("invalid blocksize %q", size))
			}
			blockByteSize = val * mags[i]
			matched = true
			break
		}
	}
	if !matched {
		return 0, sqerr.New(sqerr.Usage, fmt.Sprintf("invalid blocksize %q", size))
	}
	return blockByteSize, nil
}

func parseChecksum(csum string) (byte, error) {
	// parse checksum string
	var checksumFlag byte
	switch strings.ToLower(csum) {
	case "":
		checksumFlag = frame.NoChecksum
	case "u":
		checksumFlag = frame.UncompressedChecksum
	case "c":
		checksumFlag = frame.CompressedChecksum
	case "uc":
		checksumFlag = frame.UncompressedChecksum | frame.CompressedChecksum
	default:
		return 0, sqerr.New(sqerr.Usage, "unknown checksum value")
	}
	return checksumFlag, nil
}
