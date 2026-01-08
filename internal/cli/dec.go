package cli

import (
	"flag"
	"fmt"
	"os"
	"squish/internal/pipeline"
)

func runDec(args []string) int {
	flagSet := flag.NewFlagSet("dec", flag.ContinueOnError)
	flagSet.SetOutput(os.Stdout)
	var (
		outPath  = flagSet.String("o", "-", "output file path (or '-' for stdout)")
		outPath2 = flagSet.String("output", "", "output file path (or '-' for stdout)")
	)

	if err := flagSet.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
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
			fmt.Fprintf(os.Stderr, "dec: failed to write file %q\n\n:%v", output, err)
			return 0
		}
		outFile = f
		closeFile = true
	}
	if closeFile {
		defer outFile.Close()
	}

	// get positional arguments
	remainingArgs := flagSet.Args()
	input := "-"
	if len(remainingArgs) >= 1 {
		input = remainingArgs[0]
	}
	if len(remainingArgs) > 1 {
		fmt.Fprintf(os.Stderr, "dec: too many positional arguments (expected at most 1)")
	}

	// open the input file
	var inFile *os.File
	closeFile = false
	if input == "-" || input == "" {
		inFile = os.Stdin
	} else {
		f, err := os.Open(input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dec: failed to open input file %q\n\n", input)
			return 1
		}
		inFile = f
		closeFile = true
	}
	if closeFile {
		defer inFile.Close()
	}

	// call the business
	if err := pipeline.Decode(inFile, outFile); err != nil {
		fmt.Fprintf(os.Stderr, "dec: decode failed %v", err)
	}
	return 0
}

func printDecHelp(fs *flag.FlagSet) {
	fmt.Fprintln(os.Stdout, `
squish dec - decompress a .sqz stream into original bytes

USAGE:
  squish dec [flags] [input]

FLAGS:`)
	fs.PrintDefaults()
	fmt.Fprintln(os.Stdout, `
  -o, --output <path|->    Output file (default: '-')

EXAMPLES:
  squish dec ./file.sqz -o ./file
  squish dec ./file.sqz -o -
  cat file.sqz | squish dec > file\n`)
}
