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

// NewConfigVetCmd creates the config vet command.
func NewConfigVetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vet",
		Short: "Validate configuration",
		Long: `Validate the OPM CLI configuration file.

Checks performed:
  1. Config file exists at resolved path
  2. cue.mod/module.cue exists
  3. Config file is syntactically valid CUE
  4. Config evaluates without errors (imports resolve, constraints pass)

The config path is resolved using precedence:
  --config flag > OPM_CONFIG env > ~/.opm/config.cue

Examples:
  # Validate default configuration
  opm config vet

  # Validate custom config path
  opm config vet --config /path/to/config.cue`,
		RunE: runConfigVet,
	}

	return cmd
}

func runConfigVet(cmd *cobra.Command, args []string) error {
	// Resolve config path using precedence
	pathResult, err := config.ResolveConfigPath(config.ResolveConfigPathOptions{
		FlagValue: GetConfigPath(),
	})
	if err != nil {
		return oerrors.Wrap(oerrors.ErrNotFound, "could not resolve config path")
	}

	configPath := pathResult.ConfigPath

	output.Debug("validating config",
		"path", configPath,
		"source", pathResult.Source,
	)

	// Check 1: Config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &oerrors.DetailError{
			Type:     "not found",
			Message:  "configuration file not found",
			Location: configPath,
			Hint:     "Run 'opm config init' to create default configuration",
			Cause:    oerrors.ErrNotFound,
		}
	}

	// Check 2: cue.mod/module.cue exists
	// Determine the home directory from config path
	configDir := filepath.Dir(configPath)
	moduleFile := filepath.Join(configDir, "cue.mod", "module.cue")

	if _, err := os.Stat(moduleFile); os.IsNotExist(err) {
		return &oerrors.DetailError{
			Type:     "not found",
			Message:  "cue.mod/module.cue not found",
			Location: moduleFile,
			Hint:     "Run 'opm config init' to create configuration",
			Cause:    oerrors.ErrNotFound,
		}
	}

	// Check 3 & 4: Validate CUE syntax and evaluation
	// Use LoadOPMConfig which handles registry resolution and CUE evaluation
	_, err = config.LoadOPMConfig(config.LoaderOptions{
		RegistryFlag: GetRegistryFlag(),
		ConfigFlag:   configPath,
	})
	if err != nil {
		// The error from LoadOPMConfig already includes hints
		return err
	}

	output.Println("Configuration is valid: " + configPath)
	return nil
}
