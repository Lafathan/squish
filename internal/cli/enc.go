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
	var (
		outPath    = flagSet.String("o", "-", "output file path (or '-' for stdout)")
		outPath2   = flagSet.String("output", "", "output file path (or '-' for stdout)")
		codecPipe  = flagSet.String("codec", "", "codec pipeline, e.g. RLE|HUFFMAN")
		blockSize  = flagSet.String("blocksize", "25KiB", "block size (e.g. 256KiB, 1MiB)")
		checksum   = flagSet.String("checksum", "", "checksum mode: u|c|uc")
		listCodecs = flagSet.Bool("list-codecs", false, "list supported codecs and exit")
	)
	flagSet.Parse(args)

	var err error

	// parse and display "listCodec"
	if *listCodecs {
		codecNames := slices.Collect(maps.Keys(codec.StringToCodecIDMap))
		sort.Strings(codecNames)
		fmt.Printf("%s", strings.Join(codecNames, ", "))
		return 0
	}

	// parse codec pipeline
	if *codecPipe == "" {
		printEncHelp(flagSet)
		return 2
	}
	codecsStrings := strings.Split(*codecPipe, "|")
	var codecList []uint8
	for _, cString := range codecsStrings {
		codecID, ok := codec.StringToCodecIDMap[strings.ToUpper(cString)]
		if !ok {
			fmt.Printf("unknown codec %q\n\n", cString)
		}
		codecList = append(codecList, codecID)
	}

	// parse output and open file
	output := *outPath
	if *outPath2 != "" {
		output = *outPath2
	}
	var outFile *os.File
	if output == "-" || output == "" {
		outFile = os.Stdout
	} else {
		outFile, err = os.Create(output)
		if err != nil {
			fmt.Printf("failed to write file %q\n\n:%v", output, err)
			return 0
		}
	}
	defer outFile.Close()

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
		fmt.Printf("unknown checksum value %q\n\n", *checksum)
	}

	// parse the blocksize flags
	var blockByteSize int
	units := map[string]int{
		"B":   1,
		"KiB": 1 << 10,
		"MiB": 1 << 20,
		"KB":  1000,
		"MB":  1000000,
	}
	for suffix, magnitude := range units {
		prefix, found := strings.CutSuffix(*blockSize, suffix)
		if found {
			val, err := strconv.Atoi(prefix)
			if err != nil {
				fmt.Printf("unknown blocksize value %q\n\n", val)
				return 2
			}
			blockByteSize = val * magnitude
			break
		}
	}

	// open the input file
	var inFile *os.File
	input := args[len(args)-1]
	if input == "-" || input == "" {
		inFile = os.Stdin
	} else {
		inFile, err = os.Open(input)
		if err != nil {
			fmt.Printf("failed to read file %q\n\n", input)
			return 0
		}
	}
	defer inFile.Close()

	// call the business
	pipeline.Encode(inFile, outFile, codecList, blockByteSize, checksumFlag)
	return 0
}

func printEncHelp(fs *flag.FlagSet) {
	fmt.Println(`
squish enc - compress input into a .sqz stream

USAGE:
	squish enc -codec <pipeline> [flags] [input]

REQUIRED:
  -codec <pipeline>        Codec pipeline, e.g. RLE|HUFFMAN|RLE

FLAGS:`)
	fs.PrintDefaults()
	fmt.Println(`
PIPELINE SYNTAX:
  -codec CODEC1|CODEC2|... applies codecs in order.
  Codec names are case-insensitive.

EXAMPLES:
  squish enc ./input.txt --codec RLE|HUFFMAN -o ./output.sqz
  squish enc - -codec RLE --blocksize 128KiB -o ./out.sqz
  squish enc ./data.bin --codec RAW -o - > data.sqz`)
}
