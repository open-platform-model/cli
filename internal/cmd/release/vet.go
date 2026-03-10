package release

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
	oerrors "github.com/opmodel/cli/pkg/errors"
)

// NewReleaseVetCmd creates the release vet command.
func NewReleaseVetCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rff cmdutil.ReleaseFileFlags
	var namespace string

	c := &cobra.Command{
		Use:   "vet <release.cue>",
		Short: "Validate release file without generating manifests",
		Long: `Validate an OPM release file via the render pipeline.

This command loads a release file, optionally injects a local module, matches
components to transformers, and validates the release can be rendered successfully.
No manifests are output — purely a pass/fail validation tool.

Arguments:
  release.cue    Path to the release .cue file (required)

Examples:
  # Validate a release file
  opm release vet ./jellyfin_release.cue

  # Validate with a local module (for development)
  opm release vet ./jellyfin_release.cue --module ./my-module

  # Validate with a specific namespace
  opm release vet ./jellyfin_release.cue -n production`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runReleaseVet(args[0], cfg, &rff, namespace)
		},
	}

	rff.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Target namespace")

	return c
}

// runReleaseVet executes the release vet command.
func runReleaseVet(releaseFile string, cfg *config.GlobalConfig, rff *cmdutil.ReleaseFileFlags, namespaceFlag string) error {
	ctx := context.Background()

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

	if err := cmdutil.ShowRenderOutput(result, cmdutil.ShowOutputOpts{
		Verbose: cfg.Flags.Verbose,
	}); err != nil {
		return err
	}

	releaseLog := output.ReleaseLogger(result.Release.Name)

	// Print per-resource validation lines (skip when --verbose already showed them)
	if !cfg.Flags.Verbose {
		for _, res := range result.Resources {
			line := output.FormatResourceLine(res.GetKind(), res.GetNamespace(), res.GetName(), output.StatusValid)
			releaseLog.Info(line)
		}
	}

	summary := fmt.Sprintf("Release valid (%d resources)", result.ResourceCount())
	releaseLog.Info(output.FormatCheckmark(summary))

	return nil
}
