package main

import (
	"os"

	"github.com/monthy-app/biscuit/internal/cli"
)

func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
