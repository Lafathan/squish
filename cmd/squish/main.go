package main

import (
	"os"
	"squish/internal/cli"
)

func main() {
	args := os.Args
	os.Exit(int(cli.Run(args[1:])))
}
