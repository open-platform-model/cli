// Package cmd provides CLI command implementations.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/cue"
	oerrors "github.com/opmodel/cli/internal/errors"
	"github.com/opmodel/cli/internal/output"
)

// NewConfigVetCmd creates the config vet command.
func NewConfigVetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "vet",
		Short: "Validate configuration file",
		Long: `Validate the OPM CLI configuration file.

Checks that:
  - The config file exists at ~/.opm/config.cue (or OPM_CONFIG)
  - The config file is valid CUE
  - All required fields are present

Examples:
  # Validate configuration
  opm config vet`,
		RunE: runConfigVet,
	}
}

func runConfigVet(cmd *cobra.Command, args []string) error {
	// Get paths
	paths, err := config.PathsFromEnv()
	if err != nil {
		return oerrors.Wrap(oerrors.ErrNotFound, "could not determine config path")
	}

	// Check config file override from flag
	if configPath := GetConfigPath(); configPath != "" {
		paths.ConfigFile = configPath
	}

	// Check if config exists
	if _, err := os.Stat(paths.ConfigFile); os.IsNotExist(err) {
		return oerrors.NewNotFoundError(
			"configuration file does not exist",
			paths.ConfigFile,
			"Run 'opm config init' to create default configuration",
		)
	}

	// Check if cue.mod/module.cue exists
	cueModFile := paths.HomeDir + "/cue.mod/module.cue"
	if _, err := os.Stat(cueModFile); os.IsNotExist(err) {
		return oerrors.NewNotFoundError(
			"cue.mod/module.cue does not exist",
			cueModFile,
			"Run 'opm config init' to create configuration",
		)
	}

	// Get registry from global flag
	registry := GetRegistry()

	output.Debug("validating config",
		"config", paths.ConfigFile,
		"home", paths.HomeDir,
		"registry", registry,
	)

	// Use CUE vet to validate the config
	if err := cue.Vet(paths.HomeDir, false, registry); err != nil {
		// Enhance error message with config context
		if detail, ok := err.(*oerrors.ErrorDetail); ok {
			detail.Context = map[string]string{
				"Config": paths.ConfigFile,
			}
			return detail
		}
		return fmt.Errorf("validating config: %w", err)
	}

	output.Println("Configuration is valid: " + paths.ConfigFile)
	return nil
}
