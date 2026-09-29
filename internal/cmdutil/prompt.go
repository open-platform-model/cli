package cmdutil

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/publish"
)

// StdinReader returns the command's input and whether it may be prompted:
// an injected reader (tests) always may; the process stdin only when it is a
// terminal.
func StdinReader(c *cobra.Command) (io.Reader, bool) {
	in := c.InOrStdin()
	if f, ok := in.(*os.File); ok && f == os.Stdin {
		//nolint:gosec // G115: stdin's fd is a small non-negative number on every supported platform
		return in, term.IsTerminal(int(os.Stdin.Fd()))
	}
	return in, true
}

// PromptLine prints prompt on standard error and reads one trimmed line
// from r. what names the answer in a read error ("module path").
//
// It reads byte by byte and never past the newline, so consecutive prompts
// on one reader each get their own line.
func PromptLine(r io.Reader, prompt, what string) (string, error) {
	output.Prompt(prompt)
	var line strings.Builder
	buf := make([]byte, 1)
	for {
		n, err := r.Read(buf)
		if n == 1 {
			if buf[0] == '\n' {
				return strings.TrimSpace(line.String()), nil
			}
			line.WriteByte(buf[0])
		}
		if err != nil {
			return "", fmt.Errorf("reading %s: %w", what, err)
		}
	}
}

// Refuse prints one refusal through the house funnel and exits 2.
func Refuse(r publish.Refusal) error {
	PrintRefusals([]publish.Refusal{r})
	return &opmexit.ExitError{
		Code:    opmexit.ExitValidationError,
		Err:     errors.New(r.Headline),
		Printed: true,
	}
}

// ValidationError is a plain exit-2 error with a hint.
func ValidationError(msg, hint string) error {
	return &opmexit.ExitError{
		Code: opmexit.ExitValidationError,
		Err:  fmt.Errorf("%s\n  %s", msg, hint),
	}
}
