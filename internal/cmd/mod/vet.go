package mod

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
	oerrors "github.com/opmodel/cli/pkg/errors"
	"github.com/opmodel/cli/pkg/loader"
)

// NewModVetCmd creates the mod vet command.
func NewModVetCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rf cmdutil.RenderFlags

	c := &cobra.Command{
		Use:   "vet [path]",
		Short: "Validate module without generating manifests",
		Long: `Validate an OPM module or release via schema and render checks.

When pointed at a module directory (no release.cue present), vet runs in
module-only mode: it validates the CUE schema and checks that values satisfy
#config. No release wrapper or cluster connection is required.

When a release.cue is present (or -f is provided), vet runs the full render
pipeline and validates the complete release → transformer match → resource
output chain.

Arguments:
  path    Path to module directory (default: current directory)

Examples:
  # Validate module schema in current directory (module-only mode)
  opm mod vet

  # Validate module against explicit values (module-only mode)
  opm mod vet ./my-module -f prod-values.cue

  # Validate a complete release (release mode)
  opm mod vet ./my-release-dir

  # Validate with verbose output (show matching decisions)
  opm mod vet ./my-module --verbose`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runVet(args, cfg, &rf)
		},
	}

	rf.AddTo(c)

	return c
}

// runVet executes the vet command. It selects between two modes:
//
//   - Module-only mode: the target directory has no release.cue. Validates
//     the module CUE schema and checks that values (debugValues or -f file)
//     satisfy #config. No render pipeline or cluster connection required.
//
//   - Release mode: the target directory has a release.cue, or -f is given
//     alongside a release.cue. Runs the full shared render pipeline.
func runVet(args []string, cfg *config.GlobalConfig, rf *cmdutil.RenderFlags) error {
	ctx := context.Background()

	modulePath := cmdutil.ResolveModulePath(args)

	// Module-only mode: no release.cue in the target directory.
	if isModuleOnlyDir(modulePath) {
		return runVetModuleOnly(ctx, modulePath, cfg, rf)
	}

	// Release mode: resolve Kubernetes config and run the full render pipeline.
	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:        cfg,
		NamespaceFlag: rf.Namespace,
		ProviderFlag:  rf.Provider,
	})
	if err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}

	result, err := cmdutil.RenderRelease(ctx, cmdutil.RenderReleaseOpts{
		Args:        args,
		Values:      rf.Values,
		ReleaseName: rf.ReleaseName,
		K8sConfig:   k8sConfig,
		Config:      cfg,
		DebugValues: len(rf.Values) == 0,
	})
	if err != nil {
		return err
	}

	// Post-render: check errors, show matches, log warnings
	if err := cmdutil.ShowRenderOutput(result, cmdutil.ShowOutputOpts{
		Verbose: cfg.Flags.Verbose,
	}); err != nil {
		return err
	}

	// --- Vet-specific output ---

	releaseLog := output.ReleaseLogger(result.Release.Name)

	var valuesDetail string
	if len(rf.Values) > 0 {
		basenames := make([]string, len(rf.Values))
		for i, vf := range rf.Values {
			basenames[i] = filepath.Base(vf)
		}
		valuesDetail = strings.Join(basenames, ", ")
	} else {
		valuesDetail = "debugValues"
	}
	releaseLog.Info(output.FormatVetCheck("Values satisfy #config", valuesDetail))

	if !cfg.Flags.Verbose {
		for _, res := range result.Resources {
			line := output.FormatResourceLine(res.GetKind(), res.GetNamespace(), res.GetName(), output.StatusValid)
			releaseLog.Info(line)
		}
	}

	summary := fmt.Sprintf("Module valid (%d resources)", result.ResourceCount())
	releaseLog.Info(output.FormatCheckmark(summary))

	return nil
}

// isModuleOnlyDir returns true when modulePath is a directory that does not
// contain a release.cue file. In that case vet runs in module-only mode.
func isModuleOnlyDir(modulePath string) bool {
	_, err := os.Stat(filepath.Join(modulePath, "release.cue"))
	return os.IsNotExist(err)
}

// runVetModuleOnly validates a module directory without a release.cue.
// It loads the module CUE package, validates the schema, and checks that the
// values (from -f flag or debugValues field) satisfy #config.
// No release wrapper, engine render, or cluster connection is required.
func runVetModuleOnly(ctx context.Context, modulePath string, cfg *config.GlobalConfig, rf *cmdutil.RenderFlags) error {
	_ = ctx // reserved for future use

	cueCtx := cfg.CueContext

	// Load and structurally validate the module CUE package.
	modVal, err := loader.LoadModulePackage(cueCtx, modulePath)
	if err != nil {
		return &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("loading module: %w", err),
		}
	}

	// Derive a display name for log output.
	modName := filepath.Base(modulePath)
	if nameVal := modVal.LookupPath(cue.ParsePath("metadata.name")); nameVal.Exists() {
		if name, nameErr := nameVal.String(); nameErr == nil && name != "" {
			modName = name
		}
	}
	moduleLog := output.ReleaseLogger(modName)

	// Resolve the values to validate against #config.
	var valuesVal cue.Value
	var valuesDetail string

	if len(rf.Values) > 0 {
		// Load the first -f values file.
		valuesFile := rf.Values[0]
		var loadErr error
		valuesVal, loadErr = loader.LoadValuesFile(cueCtx, valuesFile)
		if loadErr != nil {
			return &oerrors.ExitError{
				Code: oerrors.ExitGeneralError,
				Err:  fmt.Errorf("loading values file: %w", loadErr),
			}
		}
		valuesDetail = filepath.Base(valuesFile)
	} else {
		// Fall back to the module's debugValues field.
		debugVal := modVal.LookupPath(cue.ParsePath("debugValues"))
		if !debugVal.Exists() {
			moduleLog.Warn("no debugValues defined and no -f flag — schema-only validation")
			moduleLog.Info(output.FormatCheckmark("Module schema valid"))
			return nil
		}
		valuesVal = debugVal
		valuesDetail = "debugValues"
	}

	// Validate that the values are concrete (no open constraints remaining).
	if err := valuesVal.Validate(cue.Concrete(true)); err != nil {
		cmdutil.PrintValidationError(valuesDetail+" not concrete", err)
		return &oerrors.ExitError{
			Code:    oerrors.ExitValidationError,
			Err:     fmt.Errorf("%s values are not fully concrete", valuesDetail),
			Printed: true,
		}
	}

	// Validate values against the module's #config schema.
	// Delegates to loader.ValidateConfig — the same gate used by all release
	// loading paths — so errors are always surfaced as *ConfigError with
	// structured positions and the grouped display format.
	configVal := modVal.LookupPath(cue.ParsePath("#config"))
	if configVal.Exists() {
		if cfgErr := loader.ValidateConfig(configVal, valuesVal, "module", modName); cfgErr != nil {
			cmdutil.PrintValidationError("values do not satisfy #config", cfgErr)
			return &oerrors.ExitError{
				Code:    oerrors.ExitValidationError,
				Err:     cfgErr,
				Printed: true,
			}
		}
	}

	moduleLog.Info(output.FormatVetCheck("Values satisfy #config", valuesDetail))
	moduleLog.Info(output.FormatCheckmark("Module schema valid"))

	return nil
}
