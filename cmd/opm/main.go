// Package main is the entry point for the OPM CLI.
package main

import (
	"fmt"
	"os"

	"github.com/opmodel/cli/internal/cmd"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		// Handle exit errors with specific exit codes
		if exitErr, ok := err.(*cmd.ExitError); ok {
			fmt.Fprintln(os.Stderr, exitErr.Error())
			os.Exit(exitErr.Code)
		}

		// Default to general error
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(cmd.ExitGeneralError)
	}
}
