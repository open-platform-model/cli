// Package cmd provides CLI command implementations.
package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/config"
	oerrors "github.com/opmodel/cli/internal/errors"
	"github.com/opmodel/cli/internal/output"
)

var configInitForce bool

// NewConfigInitCmd creates the config init command.
func NewConfigInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize default configuration",
		Long: `Initialize the OPM CLI configuration.

Creates the following files in ~/.opm/:
  config.cue           Main configuration file
  cue.mod/module.cue   CUE module metadata

The configuration includes:
  - Default registry for CUE module resolution
  - Kubernetes provider configuration
  - Cache directory settings

Examples:
  # Initialize configuration
  opm config init

  # Overwrite existing configuration
  opm config init --force`,
		RunE: runConfigInit,
	}

	cmd.Flags().BoolVarP(&configInitForce, "force", "f", false,
		"Overwrite existing configuration")

	return cmd
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	// Get paths
	paths, err := config.DefaultPaths()
	if err != nil {
		return oerrors.Wrap(oerrors.ErrNotFound, "could not determine home directory")
	}

	// Check if config exists
	if _, err := os.Stat(paths.ConfigFile); err == nil && !configInitForce {
		return &oerrors.DetailError{
			Type:     "validation failed",
			Message:  "configuration already exists",
			Location: paths.ConfigFile,
			Hint:     "Use --force to overwrite existing configuration.",
			Cause:    oerrors.ErrValidation,
		}
	}

	// Create directories with secure permissions (0700)
	if err := os.MkdirAll(paths.HomeDir, 0o700); err != nil {
		return oerrors.Wrap(oerrors.ErrPermission, "could not create ~/.opm directory")
	}

	cueModDir := filepath.Join(paths.HomeDir, "cue.mod")
	if err := os.MkdirAll(cueModDir, 0o700); err != nil {
		return oerrors.Wrap(oerrors.ErrPermission, "could not create ~/.opm/cue.mod directory")
	}

	// Write config.cue with secure permissions (0600)
	if err := os.WriteFile(paths.ConfigFile, []byte(config.DefaultConfigTemplate), 0o600); err != nil {
		return oerrors.Wrap(oerrors.ErrPermission, "could not write config.cue")
	}

	// Write cue.mod/module.cue with secure permissions (0600)
	moduleFile := filepath.Join(cueModDir, "module.cue")
	if err := os.WriteFile(moduleFile, []byte(config.DefaultModuleTemplate), 0o600); err != nil {
		return oerrors.Wrap(oerrors.ErrPermission, "could not write cue.mod/module.cue")
	}

	output.Println("Configuration initialized at " + paths.HomeDir)
	output.Println("")
	output.Println("Created files:")
	output.Println("  " + paths.ConfigFile)
	output.Println("  " + moduleFile)
	output.Println("")
	output.Println("Next: run 'cue mod tidy' in " + paths.HomeDir + " to resolve dependencies")
	output.Println("Validate with: opm config vet")

	return nil
}
