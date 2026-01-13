package cli

import (
	"fmt"
)

func Run(args []string) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		printMainHelp()
		return 0
	}
	switch args[0] {
	case "enc":
		return runEnc(args[1:])
	case "dec":
		return runDec(args[1:])
	//case "analyze":
	//	return runAnalyze(args[1:])
	default:
		fmt.Printf("unknown command: %q\n\n", args[0])
		printMainHelp()
		return 2
	}
}

func printMainHelp() {
	fmt.Printf(`
squish - a simple compression and decompression utility

USAGE:
	squish <command> [flags] [input]

COMMANDS:
	enc     Compress input into a .sqz stream
	dec     Decompress a .sqz stream into original bytes
	analyze Show information about a .sqz stream (codecs, blocks, ratios, etc.)

INPUT/OUTPUT:
 	input defaults to '-' (stdin) if omitted
	output defauls to '-' (stdout) if -o/--output is not provided
	use '-' to explicitly mean stdin/stdout

FLAGS:
  -o, -output <path|->
        output file (defaults to '-')
  -v, -version
        display current version of squish
  -h, -help
        show help and exit 

Examples:
	squish enc -codec RLE-HUFFMAN -o ./output.sqz ./input.txt 
	squish enc -codec RLE --blocksize 256KiB -o ./out.sqz -
	squish dec -o ./input.txt ./output.sqz 
	squish dec -o - ./compressed.sqz 
	squish analyze ./output.sqz

Run 'squish <command> -h' for command specific help.
`)
}
