package config

import (
	"fmt"
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

No platform is built or loaded. A platform/ directory left next to the
config file by an earlier release is no longer read by any command: vet
warns about it and passes. Check a platform module directory with
'opm platform check <dir>'. A legacy data-only platform.cue fails
validation: re-run 'opm config init --force' to remove it.

The config path is resolved using precedence:
  --config flag > OPM_CONFIG env > ~/.opm/config.cue

Examples:
  # Validate default configuration
  opm config vet

  # Validate custom config path
  opm config vet --config /path/to/config.cue`,
		RunE: func(c *cobra.Command, args []string) error {
			return runConfigVet(c, args, cfg)
		},
		Annotations: map[string]string{
			cmdutil.SkipConfigLoadAnnotation: "true",
		},
	}

	return c
}

func runConfigVet(_ *cobra.Command, _ []string, cfg *config.GlobalConfig) error {
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

	// A leftover pre-0019 data file fails loudly; a leftover platform
	// module directory only warns: no command reads either.
	legacy := config.LegacyPlatformFilePath(configPath)
	switch _, err := os.Stat(legacy); {
	case err == nil:
		return &opmexit.ExitError{
			Code: opmexit.ExitValidationError,
			Err: &oerrors.DetailError{
				Type:     "validation failed",
				Message:  "legacy data-only platform file found; no command reads it",
				Location: legacy,
				Hint:     "Run 'opm config init --force' to remove it",
				Cause:    oerrors.ErrValidation,
			},
		}
	case !os.IsNotExist(err):
		// Anything but "absent" is reported as what it is, never mistaken
		// for the legacy file being present.
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  fmt.Errorf("checking for a legacy platform file at %s: %w", legacy, err),
		}
	}
	if platformDir := config.LegacyPlatformDirPath(configPath); dirExists(platformDir) {
		output.Warn(platformDir + " is no longer read by any command; pass it with --platform <dir> or delete it")
	}

	return nil
}

// dirExists reports whether path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
