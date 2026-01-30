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

var modTidyDir string

// NewModTidyCmd creates the mod tidy command.
func NewModTidyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tidy",
		Short: "Update module dependencies",
		Long: `Update an OPM module's CUE dependencies.

This command delegates to the CUE binary to update the module's
dependency graph and cue.mod/module.cue file.

Examples:
  # Tidy current directory
  opm mod tidy

  # Tidy a specific directory
  opm mod tidy --dir ./my-module`,
		RunE: runModTidy,
	}

	cmd.Flags().StringVarP(&modTidyDir, "dir", "d", ".",
		"Module directory to tidy")

	return cmd
}

func runModTidy(cmd *cobra.Command, args []string) error {
	// Verify directory exists
	if _, err := os.Stat(modTidyDir); os.IsNotExist(err) {
		return oerrors.NewNotFoundError(
			fmt.Sprintf("directory does not exist: %s", modTidyDir),
			modTidyDir,
			"Specify a valid module directory with --dir",
		)
	}

	// Check for cue.mod/module.cue
	cueModFile := modTidyDir + "/cue.mod/module.cue"
	if _, err := os.Stat(cueModFile); os.IsNotExist(err) {
		return &oerrors.ErrorDetail{
			Type:     "validation failed",
			Message:  "not a CUE module directory",
			Location: modTidyDir,
			Hint:     "Directory must contain cue.mod/module.cue. Run 'opm mod init' to create a module.",
			Cause:    oerrors.ErrValidation,
		}
	}

	// Get registry from global flag
	registry := GetRegistry()

	output.Debug("tidying module dependencies",
		"dir", modTidyDir,
		"registry", registry,
	)

	// Run CUE mod tidy
	if err := cue.Tidy(modTidyDir, registry); err != nil {
		return err
	}

	output.Println("Dependencies updated successfully")
	return nil
}
