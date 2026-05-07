// Package cuetidy embeds CUE's `mod tidy` workflow as an in-process call
// against the static-linked cuelang command tree. It exists so `opm` can
// resolve a freshly initialized configuration's dependencies without the user
// having to install the standalone `cue` binary.
package cuetidy

import (
	"context"
	"fmt"
	"io"
	"os"

	cuecmd "cuelang.org/go/cmd/cue/cmd"
)

// Run executes the equivalent of `cue mod tidy` against dir.
//
// CUE's cobra command tree resolves the module root from the process working
// directory, so this function temporarily chdirs into dir and restores the
// previous directory on return. Callers must not invoke Run concurrently from
// multiple goroutines for the same process.
//
// out receives the underlying command's output (cuelang's `Command.SetOutput`
// drives a single writer for both stdout-style messages and the error printer).
// Pass io.Discard to silence it.
func Run(ctx context.Context, dir string, out io.Writer) error {
	if out == nil {
		out = io.Discard
	}

	prevDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}
	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("entering %s: %w", dir, err)
	}
	defer func() {
		// If restoring the cwd fails the process is already in a bad state;
		// preserve the primary error rather than masking it.
		_ = os.Chdir(prevDir) //nolint:errcheck // best-effort cwd restore
	}()

	cmd, newErr := cuecmd.New([]string{"mod", "tidy"})
	if newErr != nil {
		return fmt.Errorf("constructing cue command: %w", newErr)
	}
	cmd.SetOutput(out)

	return cmd.Run(ctx)
}
