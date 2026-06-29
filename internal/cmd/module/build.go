package modulecmd

import (
	"context"
	"fmt"
	"os"

	opmexit "github.com/opmodel/cli/internal/exit"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/workflow/render"
)

// NewModuleBuildCmd creates the module build command.
func NewModuleBuildCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rf cmdutil.RenderFlags
	var nameFlag string

	var (
		outputFlag string
		splitFlag  bool
		outDirFlag string
	)

	c := &cobra.Command{
		Use:   "build [path]",
		Short: "Render a module to manifests via synthetic release",
		Long: `Render an OPM module package to Kubernetes manifests by synthesizing
a #ModuleRelease around it. Values come from the module's debugValues (default)
or from -f/--values files.

Arguments:
  path    Path to a module package directory (default: current directory)

Examples:
  # Build the current module using debugValues
  opm module build

  # Build a specific module with custom values
  opm module build ./my-module -f overrides.cue

  # Build with a custom synthetic release name
  opm module build ./my-module --name my-debug`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runModuleBuild(args, cfg, &rf, nameFlag, outputFlag, splitFlag, outDirFlag)
		},
	}

	rf.AddTo(c)
	c.Flags().StringVar(&nameFlag, "name", "", "Override synthetic release name")
	c.Flags().StringVarP(&outputFlag, "output", "o", "yaml", "Output format: yaml, json")
	c.Flags().BoolVar(&splitFlag, "split", false, "Write separate files per resource")
	c.Flags().StringVar(&outDirFlag, "out-dir", "./manifests", "Directory for split output")

	return c
}

func runModuleBuild(args []string, cfg *config.GlobalConfig, rf *cmdutil.RenderFlags, nameFlag, outputFmt string, split bool, outDir string) error {
	ctx := context.Background()

	modulePath := cmdutil.ResolveModulePath(args)

	info, statErr := os.Stat(modulePath)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("module path %q not found", modulePath)}
		}
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("stat %q: %w", modulePath, statErr)}
	}
	if !info.IsDir() {
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  fmt.Errorf("module build expects a directory; CUE packages span all files in a dir. Use 'opm instance build %s' for a release file", modulePath),
		}
	}

	outputFormat, err := render.ParseManifestOutputFormat(outputFmt)
	if err != nil {
		return err
	}

	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:        cfg,
		NamespaceFlag: rf.Namespace,
		ProviderFlag:  rf.Provider,
	})
	if err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}

	result, err := render.FromModule(ctx, render.ModuleOpts{
		ModulePath:  modulePath,
		ValuesFiles: rf.Values,
		Name:        nameFlag,
		K8sConfig:   k8sConfig,
		Config:      cfg,
	})
	if err != nil {
		return err
	}

	render.ShowOutput(result, render.ShowOutputOpts{Verbose: cfg.Flags.Verbose})

	return render.WriteManifestOutput(result.Resources, outputFormat, split, outDir, result.Release.Name)
}
