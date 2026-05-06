package release

import (
	"context"
	"fmt"
	"os"

	opmexit "github.com/opmodel/cli/internal/exit"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/internal/workflow/render"
)

// NewReleaseBuildCmd creates the release build command.
func NewReleaseBuildCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rff cmdutil.ReleaseFileFlags
	var namespace string
	var nameFlag string

	var (
		outputFlag string
		splitFlag  bool
		outDirFlag string
	)

	c := &cobra.Command{
		Use:   "build <release.cue|module-dir>",
		Short: "Render a release file or module directory to manifests",
		Long: `Render an OPM release file or a module package directory to Kubernetes manifests.

When the argument is a release .cue file, it is loaded and rendered as-is.
When the argument is a directory containing a module CUE package, the CLI
synthesizes a #ModuleRelease around the module using the module's debugValues
(or values from -f) and renders it.

Arguments:
  release.cue     Path to a release .cue file
  module-dir      Path to a module package directory (synthesizes a release)

Examples:
  # Build a release file
  opm release build ./jellyfin_release.cue

  # Build with split output
  opm release build ./jellyfin_release.cue --split --out-dir ./manifests

  # Build as JSON
  opm release build ./jellyfin_release.cue -o json

  # Synthesize and build a module without writing a release.cue
  opm release build ./my-module --name my-debug`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runReleaseBuild(args[0], cfg, &rff, namespace, nameFlag, outputFlag, splitFlag, outDirFlag)
		},
	}

	rff.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Target namespace")
	c.Flags().StringVar(&nameFlag, "name", "", "Override synthetic release name (module-directory mode only)")
	c.Flags().StringVarP(&outputFlag, "output", "o", "yaml", "Output format: yaml, json")
	c.Flags().BoolVar(&splitFlag, "split", false, "Write separate files per resource")
	c.Flags().StringVar(&outDirFlag, "out-dir", "./manifests", "Directory for split output")

	return c
}

// runReleaseBuild executes the release build command.
func runReleaseBuild(buildArg string, cfg *config.GlobalConfig, rff *cmdutil.ReleaseFileFlags, namespaceFlag, nameFlag, outputFmt string, split bool, outDir string) error {
	ctx := context.Background()

	outputFormat, err := render.ParseManifestOutputFormat(outputFmt)
	if err != nil {
		return err
	}

	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:        cfg,
		NamespaceFlag: namespaceFlag,
		ProviderFlag:  rff.Provider,
	})
	if err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}

	info, statErr := os.Stat(buildArg)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("path %q not found", buildArg)}
		}
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("stat %q: %w", buildArg, statErr)}
	}

	var result *render.Result
	switch {
	case info.IsDir():
		result, err = render.FromModule(ctx, render.ModuleOpts{
			ModulePath:  buildArg,
			ValuesFiles: rff.Values,
			Name:        nameFlag,
			K8sConfig:   k8sConfig,
			Config:      cfg,
		})
	default:
		if nameFlag != "" {
			output.Warn("--name is ignored for release-file builds; it only applies to module-directory builds")
		}
		result, err = render.FromReleaseFile(ctx, render.ReleaseFileOpts{
			ReleaseFilePath: buildArg,
			ValuesFiles:     rff.Values,
			K8sConfig:       k8sConfig,
			Config:          cfg,
		})
	}
	if err != nil {
		return err
	}

	render.ShowOutput(result, render.ShowOutputOpts{Verbose: cfg.Flags.Verbose})

	return render.WriteManifestOutput(result.Resources, outputFormat, split, outDir, result.Release.Name)
}
