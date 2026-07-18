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

// NewConfigInitCmd creates the config init command.
func NewConfigInitCmd(_ *config.GlobalConfig) *cobra.Command {
	// Init-specific flags (local to this command)
	var forceFlag bool

	c := &cobra.Command{
		Use:   "init",
		Short: "Initialize default configuration",
		Long: `Initialize the OPM CLI configuration.

Creates the following files in ~/.opm/:
  config.cue     CLI configuration (registry, kubernetes, log)
  platform.cue   Local default platform (catalog subscriptions)

Both files are plain data — no CUE module, no imports, nothing to fetch.
The platform file subscribes to the official OPM catalogs and is used
whenever no --platform flag is given and no cluster Platform is readable.

Examples:
  # Initialize configuration
  opm config init

  # Overwrite existing configuration
  opm config init --force`,
		RunE: func(c *cobra.Command, args []string) error {
			return runConfigInit(args, forceFlag)
		},
		Annotations: map[string]string{
			cmdutil.SkipConfigLoadAnnotation: "true",
		},
	}

	c.Flags().BoolVarP(&forceFlag, "force", "f", false,
		"Overwrite existing configuration")

	return c
}

func runConfigInit(_ []string, force bool) error {
	// Get paths
	paths, err := config.DefaultPaths()
	if err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitNotFound,
			Err:  oerrors.Wrap(oerrors.ErrNotFound, "could not determine home directory"),
		}
	}

	// Check if config exists
	if _, err := os.Stat(paths.ConfigFile); err == nil && !force {
		return &opmexit.ExitError{
			Code: opmexit.ExitValidationError,
			Err: &oerrors.DetailError{
				Type:     "validation failed",
				Message:  "configuration already exists",
				Location: paths.ConfigFile,
				Hint:     "Use --force to overwrite existing configuration.",
				Cause:    oerrors.ErrValidation,
			},
		}
	}

	// Create directory with secure permissions (0700)
	if err := os.MkdirAll(paths.HomeDir, 0o700); err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitPermissionDenied,
			Err:  oerrors.Wrap(oerrors.ErrPermission, "could not create ~/.opm directory"),
		}
	}

	// Write config.cue with secure permissions (0600)
	if err := os.WriteFile(paths.ConfigFile, []byte(config.DefaultConfigTemplate), 0o600); err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitPermissionDenied,
			Err:  oerrors.Wrap(oerrors.ErrPermission, "could not write config.cue"),
		}
	}

	// Write platform.cue with secure permissions (0600)
	if err := os.WriteFile(paths.PlatformFile, []byte(config.DefaultPlatformTemplate), 0o600); err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitPermissionDenied,
			Err:  oerrors.Wrap(oerrors.ErrPermission, "could not write platform.cue"),
		}
	}

	output.Println(output.FormatCheckmark("Configuration initialized at " + paths.HomeDir))
	output.Println("")
	output.Println("Created files:")
	output.Println("  " + paths.ConfigFile)
	output.Println("  " + paths.PlatformFile)
	output.Println("")
	output.Println("Validate with: opm config vet")

	return nil
}
