package cmd

import (
	"errors"
	"fmt"
	"io"

	opmexit "github.com/open-platform-model/cli/internal/exit"
)

// Run executes the root command with args and returns the process exit code.
// An error that carries an exit code decides it. Any other error exits 1:
// that covers cobra's usage errors (unknown command, unknown flag, wrong
// number of arguments), which never carry a code. An error is printed on
// stderr unless the command layer already printed it.
func Run(args []string, stderr io.Writer) int {
	rootCmd := NewRootCmd()
	rootCmd.SetArgs(args)

	return exitCode(rootCmd.Execute(), stderr)
}

// exitCode maps the error of a command run to the process exit code and
// prints it on stderr when nothing printed it yet.
func exitCode(err error, stderr io.Writer) int {
	if err == nil {
		return opmexit.ExitSuccess
	}

	var exitErr *opmexit.ExitError
	if errors.As(err, &exitErr) {
		if !exitErr.Printed {
			fmt.Fprintln(stderr, err)
		}
		return exitErr.Code
	}
	fmt.Fprintln(stderr, err)
	return opmexit.ExitGeneralError
}
