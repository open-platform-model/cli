package mod

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	oerrors "github.com/opmodel/cli/pkg/errors"
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
			return runModBuild(args, cfg, &rf, outputFlag, splitFlag, outDirFlag)
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
func runModBuild(args []string, cfg *config.GlobalConfig, rf *cmdutil.RenderFlags, outputFmt string, split bool, outDir string) error {
	ctx := context.Background()

	// Validate output format
	outputFormat, err := cmdutil.ParseManifestOutputFormat(outputFmt)
	if err != nil {
		return err
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
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}

	// Render module via shared pipeline.
	// DebugValues: when no -f flag is given, use the module's debugValues field
	// as the values source (consistent with how opm mod vet works in release mode).
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
	cmdutil.ShowRenderOutput(result, cmdutil.ShowOutputOpts{Verbose: cfg.Flags.Verbose})

	// --- Build-specific output logic below ---

	// Create scoped module logger
	return cmdutil.WriteManifestOutput(result.Resources, outputFormat, split, outDir, result.Release.Name)
}
