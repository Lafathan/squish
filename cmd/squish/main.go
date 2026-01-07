package main

import (
	"os"
	"squish/internal/cli"
)

func main() {
	args := os.Args
	cli.Run(args[1:])
}
