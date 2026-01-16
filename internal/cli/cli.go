package cli

import (
	"fmt"
	"os"
	"squish/internal/sqerr"
)

func Run(args []string) sqerr.Code {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		printMainHelp()
		return sqerr.Success
	}
	switch args[0] {
	case "enc":
		return runEnc(args[1:])
	case "dec":
		return runDec(args[1:])
	default:
		fmt.Printf("unknown command: %q\n\n", args[0])
		printMainHelp()
		return sqerr.Usage
	}
}

func printMainHelp() {
	fmt.Fprintf(os.Stdout, "squish - a simple compression and decompression utility\n")
	fmt.Fprintf(os.Stdout, "\n")
	fmt.Fprintf(os.Stdout, "USAGE:\n")
	fmt.Fprintf(os.Stdout, "squish <command> [flags] [input]\n")
	fmt.Fprintf(os.Stdout, "\n")
	fmt.Fprintf(os.Stdout, "COMMANDS:\n")
	fmt.Fprintf(os.Stdout, "enc     Compress input into a .sqz stream\n")
	fmt.Fprintf(os.Stdout, "dec     Decompress a .sqz stream into original bytes\n")
	fmt.Fprintf(os.Stdout, "analyze Show information about a .sqz stream (codecs, blocks, ratios, etc.)\n")
	fmt.Fprintf(os.Stdout, "\n")
	fmt.Fprintf(os.Stdout, "INPUT/OUTPUT:\n")
	fmt.Fprintf(os.Stdout, "input defaults to '-' (stdin) if omitted\n")
	fmt.Fprintf(os.Stdout, "output defauls to '-' (stdout) if -o/--output is not provided\n")
	fmt.Fprintf(os.Stdout, "use '-' to explicitly mean stdin/stdout\n")
	fmt.Fprintf(os.Stdout, "\n")
	fmt.Fprintf(os.Stdout, "FLAGS:\n")
	fmt.Fprintf(os.Stdout, "-o, -output <path|->\n")
	fmt.Fprintf(os.Stdout, "output file (defaults to '-')\n")
	fmt.Fprintf(os.Stdout, "-v, -version\n")
	fmt.Fprintf(os.Stdout, "display current version of squish\n")
	fmt.Fprintf(os.Stdout, "-h, -help\n")
	fmt.Fprintf(os.Stdout, "show help and exit \n")
	fmt.Fprintf(os.Stdout, "\n")
	fmt.Fprintf(os.Stdout, "Examples:\n")
	fmt.Fprintf(os.Stdout, "squish enc -codec RLE-HUFFMAN -o ./output.sqz ./input.txt \n")
	fmt.Fprintf(os.Stdout, "squish enc -codec RLE --blocksize 256KiB -o ./out.sqz -\n")
	fmt.Fprintf(os.Stdout, "squish dec -o ./input.txt ./output.sqz \n")
	fmt.Fprintf(os.Stdout, "squish dec -o - ./compressed.sqz \n")
	fmt.Fprintf(os.Stdout, "squish analyze ./output.sqz\n")
	fmt.Fprintf(os.Stdout, "\n")
	fmt.Fprintf(os.Stdout, "Run 'squish <command> -h' for command specific help.\n")
}
