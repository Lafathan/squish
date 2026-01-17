package cli

import (
	"flag"
	"fmt"
	"os"
	"squish/internal/sqerr"
	"squish/internal/version"
)

func Run(args []string) sqerr.Code {
	flagSet := flag.NewFlagSet("cli", flag.ContinueOnError)
	flagSet.SetOutput(os.Stdout)

	var (
		vFlag       = flagSet.Bool("v", false, "show version and exit")
		versionFlag = flagSet.Bool("version", false, "show version and exit")
	)

	flagSet.Usage = func() {
		fmt.Fprintf(os.Stdout, "squish - a simple compression and decompression utility\n")
		fmt.Fprintf(os.Stdout, "\n")
		fmt.Fprintf(os.Stdout, "USAGE:\n")
		fmt.Fprintf(os.Stdout, "squish <command> [flags] [input]\n")
		fmt.Fprintf(os.Stdout, "\n")
		fmt.Fprintf(os.Stdout, "COMMANDS:\n")
		fmt.Fprintf(os.Stdout, "enc     Compress input into a .sqz stream\n")
		fmt.Fprintf(os.Stdout, "dec     Decompress a .sqz stream into original bytes\n")
		fmt.Fprintf(os.Stdout, "\n")
		fmt.Fprintf(os.Stdout, "INPUT/OUTPUT:\n")
		fmt.Fprintf(os.Stdout, "input defaults to '-' (stdin) if omitted\n")
		fmt.Fprintf(os.Stdout, "output defauls to '-' (stdout) if -o/--output is not provided\n")
		fmt.Fprintf(os.Stdout, "use '-' to explicitly mean stdin/stdout\n")
		fmt.Fprintf(os.Stdout, "\n")
		fmt.Fprintf(os.Stdout, "FLAGS:\n")
		flagSet.PrintDefaults()
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

	if err := flagSet.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return sqerr.Success
		}
		flagSet.Usage()
		return sqerr.Usage
	}

	if len(args) == 0 {
		flagSet.Usage()
		return sqerr.Success
	}
	if *vFlag || *versionFlag {
		fmt.Fprintf(os.Stdout, "squish %s (commit %s, built %s)\n", version.Version, version.Commit, version.Date)
		return sqerr.Success
	}
	switch args[0] {
	case "enc":
		return runEnc(args[1:])
	case "dec":
		return runDec(args[1:])
	default:
		fmt.Printf("unknown command: %q\n\n", args[0])
		flagSet.Usage()
		return sqerr.Usage
	}
}
