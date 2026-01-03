package main

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args
	fmt.Printf("Found %d args: %v\n", len(args), args)
}
