package instance

import (
	"context"
	"fmt"
	"os"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/workflow/render"
)

// NewInstanceBuildCmd creates the instance build command.
func NewInstanceBuildCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rff cmdutil.InstanceFileFlags
	var namespace string
	var nameFlag string

	var (
		outputFlag string
		splitFlag  bool
		outDirFlag string
	)

	c := &cobra.Command{
		Use:   "build <instance.cue|module-dir>",
		Short: "Render an instance file or module directory to manifests",
		Long: `Render an OPM instance file or a module package directory to Kubernetes manifests.

When the argument is an instance .cue file, it is loaded and rendered as-is.
When the argument is a directory containing a module CUE package, the CLI
synthesizes a #ModuleInstance around the module using the module's debugValues
(or values from -f) and renders it.

Arguments:
  instance.cue     Path to an instance .cue file
  module-dir      Path to a module package directory (synthesizes an instance)

Examples:
  # Build an instance file
  opm instance build ./jellyfin_instance.cue

  # Build with split output
  opm instance build ./jellyfin_instance.cue --split --out-dir ./manifests

  # Build as JSON
  opm instance build ./jellyfin_instance.cue -o json

  # Synthesize and build a module without writing an instance.cue
  opm instance build ./my-module --name my-debug`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runInstanceBuild(args[0], cfg, &rff, namespace, nameFlag, outputFlag, splitFlag, outDirFlag)
		},
	}

	rff.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Target namespace")
	c.Flags().StringVar(&nameFlag, "name", "", "Override synthetic instance name (module-directory mode only)")
	c.Flags().StringVarP(&outputFlag, "output", "o", "yaml", "Output format: yaml, json")
	c.Flags().BoolVar(&splitFlag, "split", false, "Write separate files per resource")
	c.Flags().StringVar(&outDirFlag, "out-dir", "./manifests", "Directory for split output")

	return c
}

// runInstanceBuild executes the instance build command.
func runInstanceBuild(buildArg string, cfg *config.GlobalConfig, rff *cmdutil.InstanceFileFlags, namespaceFlag, nameFlag, outputFmt string, split bool, outDir string) error {
	ctx := context.Background()

	outputFormat, err := render.ParseManifestOutputFormat(outputFmt)
	if err != nil {
		return err
	}

	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:        cfg,
		NamespaceFlag: namespaceFlag,
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
			ModulePath:   buildArg,
			ValuesFiles:  rff.Values,
			Name:         nameFlag,
			PlatformFlag: rff.Platform, // offline: no cluster read (0006 D21)
			K8sConfig:    k8sConfig,
			Config:       cfg,
		})
	default:
		if nameFlag != "" {
			output.Warn("--name is ignored for instance-file builds; it only applies to module-directory builds")
		}
		result, err = render.FromInstanceFile(ctx, render.InstanceFileOpts{
			PlatformFlag:     rff.Platform, // offline: no cluster read (0006 D21)
			InstanceFilePath: buildArg,
			ValuesFiles:      rff.Values,
			K8sConfig:        k8sConfig,
			Config:           cfg,
		})
	}
	if err != nil {
		return err
	}

	render.ShowOutput(result, render.ShowOutputOpts{Verbose: cfg.Flags.Verbose})

	return render.WriteManifestOutput(result.Resources, outputFormat, split, outDir, result.Instance.Name)
}
