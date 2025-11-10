package main

import (
	"os"

	"recopy/internal/cli"
)

// main is the program entry point; it invokes cli.Run with the command-line
// arguments (excluding the program name) and exits with the returned status code.
func main() {
	os.Exit(cli.Run(os.Args[1:]))
}