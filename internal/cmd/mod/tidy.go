package mod

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/opmodel/cli/internal/cmd"
	opmcue "github.com/opmodel/cli/internal/cue"
	"github.com/spf13/cobra"
)

// tidyOptions holds the flags for the tidy command.
type tidyOptions struct {
	dir string
}

// NewTidyCmd creates the mod tidy command.
func NewTidyCmd() *cobra.Command {
	opts := &tidyOptions{}

	c := &cobra.Command{
		Use:   "tidy",
		Short: "Tidy module dependencies",
		Long:  `Runs cue mod tidy to update module dependencies.`,
		RunE: func(c *cobra.Command, args []string) error {
			return runTidy(c.Context(), opts)
		},
	}

	c.Flags().StringVar(&opts.dir, "dir", ".", "Module directory")

	return c
}

// runTidy runs cue mod tidy.
func runTidy(ctx context.Context, opts *tidyOptions) error {
	// Check directory exists
	if _, err := os.Stat(opts.dir); os.IsNotExist(err) {
		return fmt.Errorf("%w: directory %s", cmd.ErrNotFound, opts.dir)
	}

	binary := opmcue.NewBinary()

	err := binary.Tidy(ctx, opts.dir)
	if err != nil {
		if errors.Is(err, opmcue.ErrCUENotFound) {
			return fmt.Errorf("%w: cue binary not found in PATH", cmd.ErrNotFound)
		}
		if errors.Is(err, opmcue.ErrCUEVersionMismatch) {
			return fmt.Errorf("%w: %v", cmd.ErrVersion, err)
		}
		return fmt.Errorf("tidy failed: %w", err)
	}

	fmt.Println("Module dependencies tidied")
	return nil
}
