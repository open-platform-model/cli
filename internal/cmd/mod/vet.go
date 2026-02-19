package mod

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdtypes"
	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/output"
)

// NewModVetCmd creates the mod vet command.
func NewModVetCmd(cfg *cmdtypes.GlobalConfig) *cobra.Command {
	var rf cmdutil.RenderFlags

	c := &cobra.Command{
		Use:   "vet [path]",
		Short: "Validate module without generating manifests",
		Long: `Validate an OPM module via the render pipeline.

This command loads a module, matches components to transformers, and validates
the module can be rendered successfully. No manifests are output — purely a
pass/fail validation tool with per-resource feedback.

Arguments:
  path    Path to module directory (default: current directory)

Examples:
  # Validate module in current directory
  opm mod vet

  # Validate with custom values
  opm mod vet ./my-module -f prod-values.cue -n production

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

// runVet executes the vet command.
func runVet(args []string, cfg *cmdtypes.GlobalConfig, rf *cmdutil.RenderFlags) error {
	ctx := context.Background()

	// Resolve Kubernetes configuration (namespace, provider) for the render pipeline.
	// vet does not connect to a cluster, but namespace and provider still need to flow
	// through the same resolver (flag > env > config > default).
	k8sConfig, err := cmdutil.ResolveKubernetes(cfg.OPMConfig, "", "", rf.Namespace, rf.Provider)
	if err != nil {
		return &cmdtypes.ExitError{Code: cmdtypes.ExitGeneralError, Err: fmt.Errorf("resolving config: %w", err)}
	}

	// Render module via shared pipeline
	result, err := cmdutil.RenderRelease(ctx, cmdutil.RenderReleaseOpts{
		Args:        args,
		Values:      rf.Values,
		ReleaseName: rf.ReleaseName,
		K8sConfig:   k8sConfig,
		OPMConfig:   cfg.OPMConfig,
		Registry:    cfg.Registry,
	})
	if err != nil {
		return err
	}

	// Post-render: check errors, show matches, log warnings
	if err := cmdutil.ShowRenderOutput(result, cmdutil.ShowOutputOpts{
		Verbose: cfg.Verbose,
	}); err != nil {
		return err
	}

	// --- Vet-specific logic below ---

	// Create scoped module logger for vet output
	releaseLog := output.ReleaseLogger(result.Release.Name)

	// Print values validation check line
	var valuesDetail string
	if len(rf.Values) > 0 {
		basenames := make([]string, len(rf.Values))
		for i, vf := range rf.Values {
			basenames[i] = filepath.Base(vf)
		}
		valuesDetail = strings.Join(basenames, ", ")
	} else {
		valuesDetail = "values.cue"
	}
	releaseLog.Info(output.FormatVetCheck("Values satisfy #config", valuesDetail))

	// Print per-resource validation lines (skip when --verbose already showed them)
	if !cfg.Verbose {
		for _, res := range result.Resources {
			line := output.FormatResourceLine(res.Kind(), res.Namespace(), res.Name(), output.StatusValid)
			releaseLog.Info(line)
		}
	}

	// Print final summary
	summary := fmt.Sprintf("Module valid (%d resources)", result.ResourceCount())
	releaseLog.Info(output.FormatCheckmark(summary))

	return nil
}
