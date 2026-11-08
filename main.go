package main

import (
	"os"

	"github.com/nyxlumen/nyx-browser/v2/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
