package main

import (
	"os"

	"hop.top/aps/internal/cli"
	"hop.top/aps/internal/cli/exit"
)

func main() {
	if err := cli.Execute(); err != nil {
		// exit.Code maps wrapped domain errors to canonical exit codes
		// per convention §8.1 (3=not-found, 4=conflict, 5=auth, …).
		os.Exit(exit.Code(err))
	}
}
