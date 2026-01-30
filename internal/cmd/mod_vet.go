// Package cmd provides CLI command implementations.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cue"
	oerrors "github.com/opmodel/cli/internal/errors"
	"github.com/opmodel/cli/internal/output"
)

var (
	modVetConcrete bool
	modVetDir      string
)

// NewModVetCmd creates the mod vet command.
func NewModVetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vet",
		Short: "Validate module schema",
		Long: `Validate an OPM module's CUE schema.

This command delegates to the CUE binary for validation, ensuring
the module schema is correct and values satisfy constraints.

Examples:
  # Validate current directory
  opm mod vet

  # Validate with concrete values required
  opm mod vet --concrete

  # Validate a specific directory
  opm mod vet --dir ./my-module`,
		RunE: runModVet,
	}

	cmd.Flags().BoolVarP(&modVetConcrete, "concrete", "c", false,
		"Require all values to be concrete (no defaults)")
	cmd.Flags().StringVarP(&modVetDir, "dir", "d", ".",
		"Module directory to validate")

	return cmd
}

func runModVet(cmd *cobra.Command, args []string) error {
	// Verify directory exists
	if _, err := os.Stat(modVetDir); os.IsNotExist(err) {
		return oerrors.NewNotFoundError(
			fmt.Sprintf("directory does not exist: %s", modVetDir),
			modVetDir,
			"Specify a valid module directory with --dir",
		)
	}

	// Check for module.cue
	moduleCue := modVetDir + "/module.cue"
	if _, err := os.Stat(moduleCue); os.IsNotExist(err) {
		return &oerrors.ErrorDetail{
			Type:     "validation failed",
			Message:  "not an OPM module directory",
			Location: modVetDir,
			Hint:     "Module directory must contain module.cue. Run 'opm mod init' to create a module.",
			Cause:    oerrors.ErrValidation,
		}
	}

	// Get registry from global flag
	registry := GetRegistry()

	output.Debug("validating module",
		"dir", modVetDir,
		"concrete", modVetConcrete,
		"registry", registry,
	)

	// Run CUE vet
	if err := cue.Vet(modVetDir, modVetConcrete, registry); err != nil {
		return err
	}

	output.Println("Module validated successfully")
	return nil
}
