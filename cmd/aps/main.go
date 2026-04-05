package main

import (
	"os"

	"hop.top/aps/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
