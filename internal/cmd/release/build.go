package release

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	oerrors "github.com/opmodel/cli/pkg/errors"
)

// NewReleaseBuildCmd creates the release build command.
func NewReleaseBuildCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rff cmdutil.ReleaseFileFlags
	var namespace string

	var (
		outputFlag string
		splitFlag  bool
		outDirFlag string
	)

	c := &cobra.Command{
		Use:   "build <release.cue>",
		Short: "Render release file to manifests",
		Long: `Render an OPM release file to Kubernetes manifests.

This command loads a release file, optionally injects a local module, and
outputs platform-specific resources.

Arguments:
  release.cue    Path to the release .cue file (required)

Examples:
  # Build a release file
  opm release build ./jellyfin_release.cue

  # Build with a local module
  opm release build ./jellyfin_release.cue --module ./my-module

  # Build with split output
  opm release build ./jellyfin_release.cue --split --out-dir ./manifests

  # Build as JSON
  opm release build ./jellyfin_release.cue -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runReleaseBuild(args[0], cfg, &rff, namespace, outputFlag, splitFlag, outDirFlag)
		},
	}

	rff.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Target namespace")
	c.Flags().StringVarP(&outputFlag, "output", "o", "yaml", "Output format: yaml, json")
	c.Flags().BoolVar(&splitFlag, "split", false, "Write separate files per resource")
	c.Flags().StringVar(&outDirFlag, "out-dir", "./manifests", "Directory for split output")

	return c
}

// runReleaseBuild executes the release build command.
func runReleaseBuild(releaseFile string, cfg *config.GlobalConfig, rff *cmdutil.ReleaseFileFlags, namespaceFlag, outputFmt string, split bool, outDir string) error {
	ctx := context.Background()

	outputFormat, err := cmdutil.ParseManifestOutputFormat(outputFmt)
	if err != nil {
		return err
	}

	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:        cfg,
		NamespaceFlag: namespaceFlag,
		ProviderFlag:  rff.Provider,
	})
	if err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}

	result, err := cmdutil.RenderFromReleaseFile(ctx, cmdutil.RenderFromReleaseFileOpts{
		ReleaseFilePath: releaseFile,
		ValuesFiles:     rff.Values,
		ModulePath:      rff.Module,
		K8sConfig:       k8sConfig,
		Config:          cfg,
	})
	if err != nil {
		return err
	}

	cmdutil.ShowRenderOutput(result, cmdutil.ShowOutputOpts{Verbose: cfg.Flags.Verbose})

	return cmdutil.WriteManifestOutput(result.Resources, outputFormat, split, outDir, result.Release.Name)
}
