// Package main is the entry point for the OPM CLI.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/opmodel/cli/internal/cmd"
	oerrors "github.com/opmodel/cli/internal/errors"
)

func main() {
	rootCmd := cmd.NewRootCmd()

	if err := rootCmd.Execute(); err != nil {
		// Check if the error contains an ExitError with a specific code
		var exitErr *oerrors.ExitError
		if errors.As(err, &exitErr) {
			// Only print if the command layer hasn't already printed it
			if !exitErr.Printed {
				fmt.Fprintln(os.Stderr, err)
			}
			os.Exit(exitErr.Code)
		}
		// Non-ExitError: unexpected, print it
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
