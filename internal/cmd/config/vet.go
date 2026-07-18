package config

import (
	"os"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/output"
	oerrors "github.com/open-platform-model/cli/pkg/errors"
)

// NewConfigVetCmd creates the config vet command.
func NewConfigVetCmd(cfg *config.GlobalConfig) *cobra.Command {
	c := &cobra.Command{
		Use:   "vet",
		Short: "Validate configuration",
		Long: `Validate the OPM CLI configuration files.

Checks performed:
  1. Config file exists at resolved path
  2. Config file is valid CUE and matches the embedded schema
  3. Platform file (when present) is valid, data-only, and matches
     the platform schema

A missing platform.cue is noted but does not fail validation — it is
only required when a render needs the local default platform.

The config path is resolved using precedence:
  --config flag > OPM_CONFIG env > ~/.opm/config.cue
The platform file is resolved as platform.cue next to the config file.

Examples:
  # Validate default configuration
  opm config vet

  # Validate custom config path
  opm config vet --config /path/to/config.cue`,
		RunE: func(c *cobra.Command, args []string) error {
			return runConfigVet(args, cfg)
		},
		Annotations: map[string]string{
			cmdutil.SkipConfigLoadAnnotation: "true",
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
		return &opmexit.ExitError{
			Code: opmexit.ExitNotFound,
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
		return &opmexit.ExitError{
			Code: opmexit.ExitNotFound,
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

	// Check 2: Validate config syntax + schema via a single-pass load into a
	// throwaway GlobalConfig.
	var temp config.GlobalConfig
	err = config.Load(&temp, config.LoaderOptions{
		RegistryFlag: cfg.Flags.Registry,
		ConfigFlag:   configPath,
	})
	if err != nil {
		// The error from Load already includes hints
		return &opmexit.ExitError{
			Code: opmexit.ExitValidationError,
			Err:  err,
		}
	}
	output.Println(output.FormatVetCheck("Config schema validation passed", ""))

	// Check 3: Platform file (sibling of the config file). Missing is a
	// note, not a failure.
	platformPath := config.PlatformFilePath(configPath)
	if _, err := os.Stat(platformPath); os.IsNotExist(err) {
		output.Println(output.FormatNotice("No local default platform configured (" + platformPath + " not found) — run 'opm config init' to seed one"))
		return nil
	}
	if err := config.ValidatePlatformFile(platformPath); err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitValidationError,
			Err:  err,
		}
	}
	output.Println(output.FormatVetCheck("Platform file validation passed", platformPath))

	return nil
}
