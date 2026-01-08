package cli

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"slices"
	"sort"
	"squish/internal/codec"
	"squish/internal/frame"
	"squish/internal/pipeline"
	"strconv"
	"strings"
)

func runEnc(args []string) int {
	flagSet := flag.NewFlagSet("enc", flag.ContinueOnError)
	flagSet.SetOutput(os.Stdout)
	var (
		outPath    = flagSet.String("o", "-", "output file path (or '-' for stdout)")
		outPath2   = flagSet.String("output", "", "output file path (or '-' for stdout)")
		codecPipe  = flagSet.String("codec", "", "codec pipeline, e.g. RLE-HUFFMAN")
		blockSize  = flagSet.String("blocksize", "25KiB", "block size (e.g. 256KiB, 1MiB)")
		checksum   = flagSet.String("checksum", "", "checksum mode: u|c|uc")
		listCodecs = flagSet.Bool("list-codecs", false, "list supported codecs and exit")
	)

	if err := flagSet.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}

	// parse and display "listCodec"
	if *listCodecs {
		codecNames := slices.Collect(maps.Keys(codec.StringToCodecIDMap))
		sort.Strings(codecNames)
		fmt.Fprintf(os.Stdout, "%s", strings.Join(codecNames, ", "))
		return 0
	}

	// parse codec pipeline
	if *codecPipe == "" {
		fmt.Fprintf(os.Stdout, "enc: missing required -codec")
		printEncHelp(flagSet)
		return 2
	}
	codecStrings := strings.Split(*codecPipe, "-")
	codecList := make([]uint8, 0, len(codecStrings))
	for _, cString := range codecStrings {
		if cString == "" {
			fmt.Fprintf(os.Stderr, "enc: empty codec in pipeline")
		}
		codecID, ok := codec.StringToCodecIDMap[strings.ToUpper(cString)]
		if !ok {
			fmt.Fprintf(os.Stderr, "enc: unknown codec %q (try: squish enc --list-codecs)", cString)
		}
		codecList = append(codecList, codecID)
	}

	// parse output file
	output := *outPath
	if *outPath2 != "" {
		output = *outPath2
	}
	var outFile *os.File
	var closeFile bool
	if output == "-" || output == "" {
		outFile = os.Stdout
	} else {
		f, err := os.Create(output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "enc: failed to write file %q\n\n:%v", output, err)
			return 0
		}
		outFile = f
		closeFile = true
	}
	if closeFile {
		defer outFile.Close()
	}

	// parse the checksum flags
	var checksumFlag byte
	switch *checksum {
	case "":
		checksumFlag = frame.NoChecksum
	case "u":
		checksumFlag = frame.UncompressedChecksum
	case "c":
		checksumFlag = frame.CompressedChecksum
	case "uc":
		checksumFlag = frame.UncompressedChecksum | frame.CompressedChecksum
	default:
		fmt.Fprintf(os.Stderr, "enc: unknown checksum value %q\n\n", *checksum)
	}

	// parse the blocksize flags
	var blockByteSize int
	var matched bool = false
	bs := strings.TrimSpace(*blockSize)
	units := [5]string{"KiB", "MiB", "KB", "MB", "B"}
	mags := [5]int{1 << 10, 1 << 20, 1000, 1000000, 1}
	for i := range 5 {
		prefix, found := strings.CutSuffix(bs, units[i])
		prefix = strings.TrimSpace(prefix)
		if found {
			val, err := strconv.Atoi(prefix)
			if err != nil || val <= 0 {
				fmt.Printf("enc: invalid blocksize %q (expected e.g. 256KiB, 1MiB)\n\n", bs)
				return 2
			}
			blockByteSize = val * mags[i]
			matched = true
			break
		}
	}
	if !matched {
		fmt.Printf("enc: invalid blocksize %q (expected e.g. 256KiB, 1MiB)\n\n", bs)
	}

	// get positional arguments
	remainingArgs := flagSet.Args()
	input := "-"
	if len(remainingArgs) >= 1 {
		input = remainingArgs[0]
	}
	if len(remainingArgs) > 1 {
		fmt.Fprintf(os.Stderr, "enc: too many positional arguments (expected at most 1)")
	}

	// open the input file
	var inFile *os.File
	closeFile = false
	if input == "-" || input == "" {
		inFile = os.Stdin
	} else {
		f, err := os.Open(input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "enc: failed to open input file %q\n\n", input)
			return 1
		}
		inFile = f
		closeFile = true
	}
	if closeFile {
		defer inFile.Close()
	}

	// call the business
	if err := pipeline.Encode(inFile, outFile, codecList, blockByteSize, checksumFlag); err != nil {
		fmt.Fprintf(os.Stderr, "enc: encode failed: %v", err)
	}
	return 0
}

func printEncHelp(fs *flag.FlagSet) {
	fmt.Fprintln(os.Stdout, `
squish enc - compress input into a .sqz stream

USAGE:
	squish enc -codec <pipeline> [flags] [input]

REQUIRED:
  -codec <pipeline>        Codec pipeline, e.g. RLE|HUFFMAN|RLE

FLAGS:`)
	fs.PrintDefaults()
	fmt.Fprintln(os.Stdout, `
PIPELINE SYNTAX:
  -codec CODEC1|CODEC2|... applies codecs in order.
  Codec names are case-insensitive.

EXAMPLES:
  squish enc ./input.txt --codec RLE|HUFFMAN -o ./output.sqz
  squish enc - -codec RLE --blocksize 128KiB -o ./out.sqz
  squish enc ./data.bin --codec RAW -o - > data.sqz`)
}
