package mod

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdtypes"
	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
)

// NewModBuildCmd creates the mod build command.
func NewModBuildCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rf cmdutil.RenderFlags

	// Build-specific flags (local to this command)
	var (
		outputFlag string
		splitFlag  bool
		outDirFlag string
	)

	c := &cobra.Command{
		Use:   "build [path]",
		Short: "Render module to manifests",
		Long: `Render an OPM module to Kubernetes manifests.

This command loads a module, matches components to transformers from the
configured provider, and outputs platform-specific resources.

Arguments:
  path    Path to module directory (default: current directory)

Examples:
  # Build module in current directory
  opm mod build

  # Build with custom values
  opm mod build ./my-module -f prod-values.cue -n production

  # Build with split output
  opm mod build ./my-module --split --out-dir ./manifests

  # Build with verbose output
  opm mod build ./my-module --verbose

  # Build as JSON
  opm mod build ./my-module -o json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runBuild(args, cfg, &rf, outputFlag, splitFlag, outDirFlag)
		},
	}

	rf.AddTo(c)

	// Build-specific flags
	c.Flags().StringVarP(&outputFlag, "output", "o", "yaml",
		"Output format: yaml, json")
	c.Flags().BoolVar(&splitFlag, "split", false,
		"Write separate files per resource")
	c.Flags().StringVar(&outDirFlag, "out-dir", "./manifests",
		"Directory for split output")
	return c
}

// runBuild executes the build command.
func runBuild(args []string, cfg *config.GlobalConfig, rf *cmdutil.RenderFlags, outputFmt string, split bool, outDir string) error {
	ctx := context.Background()

	// Validate output format
	outputFormat, valid := output.ParseFormat(outputFmt)
	if !valid {
		return &cmdtypes.ExitError{
			Code: cmdtypes.ExitGeneralError,
			Err:  fmt.Errorf("invalid output format %q (valid: yaml, json)", outputFmt),
		}
	}

	// Resolve Kubernetes configuration (namespace, provider) for the render pipeline.
	// build does not connect to a cluster, but namespace and provider still need to flow
	// through the same resolver (flag > env > config > default).
	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:        cfg,
		NamespaceFlag: rf.Namespace,
		ProviderFlag:  rf.Provider,
	})
	if err != nil {
		return &cmdtypes.ExitError{Code: cmdtypes.ExitGeneralError, Err: fmt.Errorf("resolving config: %w", err)}
	}

	// Render module via shared pipeline
	result, err := cmdutil.RenderRelease(ctx, cmdutil.RenderReleaseOpts{
		Args:        args,
		Values:      rf.Values,
		ReleaseName: rf.ReleaseName,
		K8sConfig:   k8sConfig,
		Config:      cfg,
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

	// --- Build-specific output logic below ---

	// Convert resources to ResourceInfo interface
	resourceInfos := make([]output.ResourceInfo, len(result.Resources))
	for i, r := range result.Resources {
		resourceInfos[i] = r
	}

	// Create scoped module logger
	releaseLog := output.ReleaseLogger(result.Release.Name)

	// Output results
	if split {
		// Split output to files
		splitOpts := output.SplitOptions{
			OutDir: outDir,
			Format: outputFormat,
		}
		if err := output.WriteSplitManifests(resourceInfos, splitOpts); err != nil {
			return &cmdtypes.ExitError{Code: cmdtypes.ExitGeneralError, Err: fmt.Errorf("writing split manifests: %w", err)}
		}
		releaseLog.Info(fmt.Sprintf("wrote %d resources to %s", len(result.Resources), outDir))
	} else {
		// Output to stdout
		manifestOpts := output.ManifestOptions{
			Format: outputFormat,
			Writer: os.Stdout,
		}
		if err := output.WriteManifests(resourceInfos, manifestOpts); err != nil {
			return &cmdtypes.ExitError{Code: cmdtypes.ExitGeneralError, Err: fmt.Errorf("writing manifests: %w", err)}
		}
	}

	return nil
}
