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

// vetOptions holds the flags for the vet command.
type vetOptions struct {
	dir      string
	concrete bool
}

// NewVetCmd creates the mod vet command.
func NewVetCmd() *cobra.Command {
	opts := &vetOptions{}

	c := &cobra.Command{
		Use:   "vet",
		Short: "Validate module CUE files",
		Long:  `Validates the module's CUE files using the cue vet command.`,
		RunE: func(c *cobra.Command, args []string) error {
			return runVet(c.Context(), opts)
		},
	}

	c.Flags().StringVar(&opts.dir, "dir", ".", "Module directory")
	c.Flags().BoolVar(&opts.concrete, "concrete", false, "Require all values to be concrete")

	return c
}

// runVet validates the module.
func runVet(ctx context.Context, opts *vetOptions) error {
	// Check directory exists
	if _, err := os.Stat(opts.dir); os.IsNotExist(err) {
		return fmt.Errorf("%w: directory %s", cmd.ErrNotFound, opts.dir)
	}

	binary := opmcue.NewBinary()

	err := binary.Vet(ctx, opts.dir, opts.concrete)
	if err != nil {
		if errors.Is(err, opmcue.ErrCUENotFound) {
			return fmt.Errorf("%w: cue binary not found in PATH", cmd.ErrNotFound)
		}
		if errors.Is(err, opmcue.ErrCUEVersionMismatch) {
			return fmt.Errorf("%w: %v", cmd.ErrVersion, err)
		}
		return fmt.Errorf("%w: %v", cmd.ErrValidation, err)
	}

	fmt.Println("Module validated successfully")
	return nil
}
