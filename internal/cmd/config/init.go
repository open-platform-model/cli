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

Creates ~/.opm/config.cue: the CLI configuration (registry, kubernetes, log),
plain data. Init is offline; nothing is resolved.

Init writes no platform. A render resolves its platform from --platform
<dir>, else the cluster's Platform, else a platform generated from the
render's own dependency pins. An existing ~/.opm/platform/ from an earlier
release is left untouched and no longer read; pass it with
--platform ~/.opm/platform to keep rendering against it.

A legacy data-only ~/.opm/platform.cue from an earlier release is removed.

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

	// A pre-0019 data-only platform.cue is a stale artifact of an earlier
	// release; remove it and say so.
	removedLegacy, err := removeLegacyPlatformFile(config.LegacyPlatformFilePath(paths.ConfigFile))
	if err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitPermissionDenied,
			Err:  oerrors.Wrap(oerrors.ErrPermission, "could not remove the legacy platform file: "+err.Error()),
		}
	}

	output.Println(output.FormatCheckmark("Configuration initialized at " + paths.HomeDir))
	output.Println("")
	output.Println("Created files:")
	output.Println("  " + paths.ConfigFile)
	if removedLegacy != "" {
		output.Println("")
		output.Println(output.FormatNotice("Removed legacy platform file " + removedLegacy + " (no command reads it)"))
	}
	output.Println("")
	output.Println("Validate with: opm config vet")

	return nil
}

// removeLegacyPlatformFile deletes the pre-0019 data-only platform file at
// path when it exists and returns the path removed ("" when there was none).
func removeLegacyPlatformFile(path string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if err := os.Remove(path); err != nil {
		return "", err
	}
	return path, nil
}
