package cli

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"slices"
	"sort"
	"squish/internal/codec"
	"squish/internal/pipeline"
	"squish/internal/sqerr"
	"strings"
)

func runEnc(args []string) sqerr.Code {
	flagSet := flag.NewFlagSet("enc", flag.ContinueOnError)
	flagSet.SetOutput(os.Stdout)

	var (
		outPath    = flagSet.String("o", "", "output file path (default stdout)")
		outPath2   = flagSet.String("output", "", "output file path (default stdout)")
		codecPipe  = flagSet.String("codec", "AUTO", "codec pipeline, e.g. RLE-HUFFMAN")
		blockSize  = flagSet.String("blocksize", "128KiB", "block size (e.g. 256KiB, 1MiB)")
		checksum   = flagSet.String("checksum", "", "checksum mode: u|c|uc")
		listCodecs = flagSet.Bool("list-codecs", false, "list supported codecs and exit")
	)

	flagSet.Usage = func() {
		fmt.Fprintf(os.Stdout, "squish enc - compress input into a .sqz stream\n")
		fmt.Fprintf(os.Stdout, "\n")
		fmt.Fprintf(os.Stdout, "USAGE:\n")
		fmt.Fprintf(os.Stdout, "  squish enc -codec <pipeline> [flags] [input]\n")
		fmt.Fprintf(os.Stdout, "\n")
		fmt.Fprintf(os.Stdout, "FLAGS:\n")
		flagSet.PrintDefaults()
		fmt.Fprintf(os.Stdout, "\n")
		fmt.Fprintf(os.Stdout, "PIPELINE SYNTAX:\n")
		fmt.Fprintf(os.Stdout, "  -codec CODEC1-CODEC2-... applies codecs in order, left-to-right.\n")
		fmt.Fprintf(os.Stdout, "  Codec names are case-insensitive.\n")
		fmt.Fprintf(os.Stdout, "\n")
		fmt.Fprintf(os.Stdout, "BLOCKSIZE SYNTAX:\n")
		fmt.Fprintf(os.Stdout, "  -blocksize <value><unit>.\n")
		fmt.Fprintf(os.Stdout, "  <value> is a non-zero integer.\n")
		fmt.Fprintf(os.Stdout, "  <unit> options are B, KB, KiB, MB, and MiB.\n")
		fmt.Fprintf(os.Stdout, "  Units are case sensitive.\n")
		fmt.Fprintf(os.Stdout, "\n")
		fmt.Fprintf(os.Stdout, "EXAMPLES:\n")
		fmt.Fprintf(os.Stdout, "  squish enc ./input.txt -codec RLE-HUFFMAN -o ./output.sqz\n")
		fmt.Fprintf(os.Stdout, "  squish enc -codec RLE -blocksize 128KiB -o ./out.sqz\n")
		fmt.Fprintf(os.Stdout, "  squish enc ./data.bin -o > data.sqz\n")
	}

	if err := flagSet.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return sqerr.Success
		}
		return sqerr.Usage
	}

	// parse and display "listCodec"
	if *listCodecs {
		codecNames := slices.Collect(maps.Keys(codec.StringToCodecIDMap))
		codecNames = append(codecNames, slices.Collect(maps.Keys(codec.CodecAliases))...)
		sort.Strings(codecNames)
		fmt.Fprintf(os.Stdout, "%s\n", strings.Join(codecNames, ", "))
		return sqerr.Success
	}

	// parse codec pipeline
	codecList, err := parseCodecPipeline(*codecPipe)
	if err != nil {
		fmt.Fprintf(os.Stderr, "enc: failed to parse codec list %q: %v", *codecPipe, err)
		return sqerr.ErrorCode(err)
	}

	// parse output file
	output := *outPath
	if *outPath2 != "" {
		output = *outPath2
	}
	outFile, closeFn, err := openOutput(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "enc: failed to write file %q: %v", output, err)
		return sqerr.IO
	}
	defer closeFn()

	// parse the checksum flags
	checksumFlag, err := parseChecksum(*checksum)
	if err != nil {
		fmt.Fprintf(os.Stderr, "enc: failed to parse checksum value %q: %v", *checksum, err)
		return sqerr.ErrorCode(err)
	}

	// parse the blocksize flags
	blockByteSize, err := parseBlockSize(*blockSize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "enc: %v", err)
		return sqerr.ErrorCode(err)
	}

	// get positional arguments
	remainingArgs := flagSet.Args()
	input := ""
	if len(remainingArgs) >= 1 {
		input = remainingArgs[0]
	}
	if len(remainingArgs) > 1 {
		fmt.Fprintf(os.Stderr, "enc: too many positional arguments (expected at most 1)")
		return sqerr.Usage
	}

	// parse input file
	inFile, closeFn, err := openInput(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "enc: failed to open input file %q: %v", input, err)
		return sqerr.IO
	}
	defer closeFn()

	// call the business
	if err := pipeline.Encode(inFile, outFile, codecList, blockByteSize, checksumFlag); err != nil {
		fmt.Fprintf(os.Stderr, "enc: encode failed: %v", err)
		return sqerr.ErrorCode(err)
	}
	return sqerr.Success
}
