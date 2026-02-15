package cli

import (
	"flag"
	"fmt"
	"os"
	"squish/internal/pipeline"
	"squish/internal/sqerr"
)

func runDec(args []string) sqerr.Code {
	flagSet := flag.NewFlagSet("dec", flag.ContinueOnError)
	flagSet.SetOutput(os.Stdout)
	var (
		outPath  = flagSet.String("o", "", "output file path (default stdout)")
		outPath2 = flagSet.String("output", "", "output file path (default stdout)")
	)

	flagSet.Usage = func() {
		fmt.Fprintf(os.Stdout, "squish dec - deccompress a .sqz stream into original bytes\n")
		fmt.Fprintf(os.Stdout, "\n")
		fmt.Fprintf(os.Stdout, "USAGE:\n")
		fmt.Fprintf(os.Stdout, "  squish dec [flags] [input]\n")
		fmt.Fprintf(os.Stdout, "\n")
		fmt.Fprintf(os.Stdout, "FLAGS:\n")
		flagSet.PrintDefaults()
		fmt.Fprintf(os.Stdout, "\n")
		fmt.Fprintf(os.Stdout, "EXAMPLES:\n")
		fmt.Fprintf(os.Stdout, "  squish dec -o ./file ./file.sqz\n")
		fmt.Fprintf(os.Stdout, "  squish dec ./file.sqz \n")
		fmt.Fprintf(os.Stdout, "  squish enc -codec RAW ./data.bin > data.sqz\n")
	}

	if err := flagSet.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return sqerr.Success
		}
		return sqerr.Usage
	}

	// parse output file
	output := *outPath
	if *outPath2 != "" {
		output = *outPath2
	}
	outFile, closeFn, err := openOutput(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dec: failed to write file %q: %v", output, err)
		return sqerr.IO
	}
	defer closeFn()

	// get positional arguments
	remainingArgs := flagSet.Args()
	input := ""
	if len(remainingArgs) >= 1 {
		input = remainingArgs[0]
	}
	if len(remainingArgs) > 1 {
		fmt.Fprintf(os.Stderr, "dec: too many positional arguments (expected at most 1)")
		return sqerr.IO
	}

	// parse input file
	inFile, closeFn, err := openInput(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dec: failed to open input file %q: %v", input, err)
		return sqerr.IO
	}
	defer closeFn()

	// call the business
	if err := pipeline.Decode(inFile, outFile); err != nil {
		fmt.Fprintf(os.Stderr, "dec: decode failed %v", err)
		return sqerr.ErrorCode(err)
	}
	return sqerr.Success
}
