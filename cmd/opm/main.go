// Package main is the entry point for the OPM CLI.
package main

import (
	"os"

	"github.com/open-platform-model/cli/internal/cmd"
)

func main() {
	os.Exit(cmd.Run(os.Args[1:], os.Stderr))
}
