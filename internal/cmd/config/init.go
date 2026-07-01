package config

import (
	"context"
	"os"
	"path/filepath"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/cuetidy"
	"github.com/open-platform-model/cli/internal/output"
	oerrors "github.com/open-platform-model/cli/pkg/errors"
)

// NewConfigInitCmd creates the config init command.
func NewConfigInitCmd(_ *config.GlobalConfig) *cobra.Command {
	// Init-specific flags (local to this command)
	var (
		forceFlag  bool
		noTidyFlag bool
	)

	c := &cobra.Command{
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

After writing the files, opm runs the equivalent of 'cue mod tidy' in
~/.opm to fetch the providers referenced by the default config so that
'opm config vet' works without further setup. Disable with --no-tidy
when working offline or against a registry that does not yet have the
providers available.

Examples:
  # Initialize configuration (resolves dependencies automatically)
  opm config init

  # Overwrite existing configuration
  opm config init --force

  # Skip dependency resolution (offline / air-gapped use)
  opm config init --no-tidy`,
		RunE: func(c *cobra.Command, args []string) error {
			return runConfigInit(c.Context(), args, forceFlag, noTidyFlag)
		},
		Annotations: map[string]string{
			cmdutil.SkipConfigLoadAnnotation: "true",
		},
	}

	c.Flags().BoolVarP(&forceFlag, "force", "f", false,
		"Overwrite existing configuration")
	c.Flags().BoolVar(&noTidyFlag, "no-tidy", false,
		"Skip running 'cue mod tidy' after writing the config")

	return c
}

func runConfigInit(ctx context.Context, _ []string, force, noTidy bool) error {
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

	// Create directories with secure permissions (0700)
	if err := os.MkdirAll(paths.HomeDir, 0o700); err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitPermissionDenied,
			Err:  oerrors.Wrap(oerrors.ErrPermission, "could not create ~/.opm directory"),
		}
	}

	cueModDir := filepath.Join(paths.HomeDir, "cue.mod")
	if err := os.MkdirAll(cueModDir, 0o700); err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitPermissionDenied,
			Err:  oerrors.Wrap(oerrors.ErrPermission, "could not create ~/.opm/cue.mod directory"),
		}
	}

	// Write config.cue with secure permissions (0600)
	if err := os.WriteFile(paths.ConfigFile, []byte(config.DefaultConfigTemplate), 0o600); err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitPermissionDenied,
			Err:  oerrors.Wrap(oerrors.ErrPermission, "could not write config.cue"),
		}
	}

	// Write cue.mod/module.cue with secure permissions (0600)
	moduleFile := filepath.Join(cueModDir, "module.cue")
	if err := os.WriteFile(moduleFile, []byte(config.DefaultModuleTemplate), 0o600); err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitPermissionDenied,
			Err:  oerrors.Wrap(oerrors.ErrPermission, "could not write cue.mod/module.cue"),
		}
	}

	output.Println(output.FormatCheckmark("Configuration initialized at " + paths.HomeDir))
	output.Println("")
	output.Println("Created files:")
	output.Println("  " + paths.ConfigFile)
	output.Println("  " + moduleFile)
	output.Println("")

	if noTidy {
		output.Println(output.FormatNotice("Run 'cue mod tidy' in " + paths.HomeDir + " to resolve dependencies (skipped: --no-tidy)"))
		output.Println("Validate with: opm config vet")
		return nil
	}

	if err := tidyConfigDir(ctx, paths.HomeDir); err != nil {
		// Tidy failure is not fatal: the files are written. Surface the cause
		// and fall back to the manual instructions so the user can recover.
		output.Println(output.FormatNotice("Could not resolve dependencies automatically: " + err.Error()))
		output.Println(output.FormatNotice("Run 'cue mod tidy' in " + paths.HomeDir + " once the issue is resolved"))
		output.Println("Validate with: opm config vet")
		return nil //nolint:nilerr // best-effort: files exist; tidy failure should not fail init
	}

	output.Println(output.FormatCheckmark("Dependencies resolved (cue.mod/module.cue updated)"))
	output.Println("Validate with: opm config vet")

	return nil
}

// tidyConfigDir runs the equivalent of `cue mod tidy` against dir. If the user
// has not set CUE_REGISTRY, we provide the same default registry baked into the
// config template so the providers referenced there can be resolved on a fresh
// machine. The original env value is restored on return.
func tidyConfigDir(ctx context.Context, dir string) error {
	const cueRegistryEnv = "CUE_REGISTRY"

	prev, hadPrev := os.LookupEnv(cueRegistryEnv)
	if !hadPrev || prev == "" {
		if err := os.Setenv(cueRegistryEnv, config.DefaultRegistry); err != nil {
			return err
		}
		defer func() {
			if hadPrev {
				_ = os.Setenv(cueRegistryEnv, prev)
			} else {
				_ = os.Unsetenv(cueRegistryEnv)
			}
		}()
	}

	return cuetidy.Run(ctx, dir, os.Stderr)
}
