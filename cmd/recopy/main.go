package main

import (
	"os"

	"recopy/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
