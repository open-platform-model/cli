package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/config"
	oerrors "github.com/opmodel/cli/internal/errors"
	"github.com/opmodel/cli/internal/output"
)

// NewConfigVetCmd creates the config vet command.
func NewConfigVetCmd(cfg *config.GlobalConfig) *cobra.Command {
	c := &cobra.Command{
		Use:   "vet",
		Short: "Validate configuration",
		Long: `Validate the OPM CLI configuration file.

Checks performed:
  1. Config file exists at resolved path
  2. cue.mod/module.cue exists
  3. Config file is syntactically valid CUE
  4. Config evaluates without errors (imports resolve, constraints pass)
  5. Config matches embedded schema (no unknown fields, types correct)

The config path is resolved using precedence:
  --config flag > OPM_CONFIG env > ~/.opm/config.cue

Examples:
  # Validate default configuration
  opm config vet

  # Validate custom config path
  opm config vet --config /path/to/config.cue`,
		RunE: func(c *cobra.Command, args []string) error {
			return runConfigVet(args, cfg)
		},
	}

	return c
}

func runConfigVet(_ []string, cfg *config.GlobalConfig) error {
	// Resolve config path using precedence: cfg.Flags.Config > env > default
	pathResult, err := config.ResolveConfigPath(config.ResolveConfigPathOptions{
		FlagValue: cfg.Flags.Config,
	})
	if err != nil {
		return &oerrors.ExitError{
			Code: oerrors.ExitNotFound,
			Err:  oerrors.Wrap(oerrors.ErrNotFound, "could not resolve config path"),
		}
	}

	configPath := pathResult.ConfigPath

	output.Debug("validating config",
		"path", configPath,
		"source", pathResult.Source,
	)

	// Check 1: Config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &oerrors.ExitError{
			Code: oerrors.ExitNotFound,
			Err: &oerrors.DetailError{
				Type:     "not found",
				Message:  "configuration file not found",
				Location: configPath,
				Hint:     "Run 'opm config init' to create default configuration",
				Cause:    oerrors.ErrNotFound,
			},
		}
	}
	output.Println(output.FormatVetCheck("Config file found", configPath))

	// Check 2: cue.mod/module.cue exists
	// Determine the home directory from config path
	configDir := filepath.Dir(configPath)
	moduleFile := filepath.Join(configDir, "cue.mod", "module.cue")

	if _, err := os.Stat(moduleFile); os.IsNotExist(err) {
		return &oerrors.ExitError{
			Code: oerrors.ExitNotFound,
			Err: &oerrors.DetailError{
				Type:     "not found",
				Message:  "cue.mod/module.cue not found",
				Location: moduleFile,
				Hint:     "Run 'opm config init' to create configuration",
				Cause:    oerrors.ErrNotFound,
			},
		}
	}
	output.Println(output.FormatVetCheck("Module metadata found", moduleFile))

	// Check 3, 4, 5: Validate CUE syntax, evaluation, and schema
	// Use Load into a throwaway GlobalConfig to validate the config file.
	var temp config.GlobalConfig
	err = config.Load(&temp, config.LoaderOptions{
		RegistryFlag: cfg.Flags.Registry,
		ConfigFlag:   configPath,
	})
	if err != nil {
		// The error from Load already includes hints
		return &oerrors.ExitError{
			Code: oerrors.ExitValidationError,
			Err:  err,
		}
	}

	output.Println(output.FormatVetCheck("CUE syntax and evaluation passed", ""))
	output.Println(output.FormatVetCheck("Schema validation passed", ""))
	return nil
}
