package config

import (
	"os"
	"path/filepath"

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

Creates the following in ~/.opm/:
  config.cue                  CLI configuration (registry, kubernetes, log)
  platform/cue.mod/module.cue Local default platform: pinned core + catalogs
  platform/platform.cue       Local default platform: one entry per catalog

config.cue is plain data. platform/ is a real CUE module that subscribes
to the official OPM catalogs by import; it is used whenever no --platform
flag is given and no cluster Platform is readable. Init writes the pins
without resolving anything (offline); 'opm config vet' builds the module
and proves the pins resolve.

Maintenance: catalog builds are pinned in platform/cue.mod/module.cue.
To move to a newer catalog release, edit the pin there (or run
'cue mod get <path>@<version>' in that directory) and run 'opm config vet'.

A legacy data-only ~/.opm/platform.cue from an earlier release is removed
when the module is written.

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

	// Write the platform module (0700 dirs, 0600 files); offline, nothing
	// is resolved (0019 D5: the module's cue.mod pins are the platform).
	if err := config.WritePlatformModule(paths.PlatformDir); err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitPermissionDenied,
			Err:  oerrors.Wrap(oerrors.ErrPermission, "could not write the platform module: "+err.Error()),
		}
	}

	// A pre-0019 data-only platform.cue beside the module would be a silent
	// second answer; remove it and say so.
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
	output.Println("  " + filepath.Join(paths.PlatformDir, filepath.FromSlash(config.PlatformModuleFileName)))
	output.Println("  " + filepath.Join(paths.PlatformDir, config.PlatformCUEFileName))
	if removedLegacy != "" {
		output.Println("")
		output.Println(output.FormatNotice("Removed legacy platform file " + removedLegacy + " (the local default platform is now the module at " + paths.PlatformDir + ")"))
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
