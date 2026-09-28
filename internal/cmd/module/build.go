package modulecmd

import (
	"context"
	"fmt"
	"os"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/workflow/render"
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
		Short: "Render a module to manifests via synthetic instance",
		Long: `Render an OPM module package to Kubernetes manifests by synthesizing
a #ModuleInstance around it. Values come from the module's debugValues (default)
or from -f/--values files.

The render answers whether the module renders with the catalogs it declares:
by default it runs against a platform generated from the module's own
cue.mod/module.cue, one registry entry per catalog the module pins, at the
pinned version. Neither the cluster nor ~/.opm/platform/ is read. Pass
--platform <dir> to render against a platform module instead, for example one
pulled with 'opm platform pull'.

Arguments:
  path    Path to a module package directory (default: current directory)

Examples:
  # Build the current module against its own deps using debugValues
  opm module build

  # Build a specific module with custom values
  opm module build ./my-module -f overrides.cue

  # Build against a platform module instead of the module's deps
  opm module build ./my-module --platform ./pulled-platform

  # Build with a custom synthetic instance name
  opm module build ./my-module --name my-debug`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runModuleBuild(args, cfg, &rf, nameFlag, outputFlag, splitFlag, outDirFlag)
		},
	}

	rf.AddTo(c)
	useModuleDepsPlatformHelp(c)
	c.Flags().StringVar(&nameFlag, "name", "", "Override synthetic instance name")
	c.Flags().StringVarP(&outputFlag, "output", "o", "yaml", "Output format: yaml, json")
	c.Flags().BoolVar(&splitFlag, "split", false, "Write separate files per resource")
	c.Flags().StringVar(&outDirFlag, "out-dir", "./manifests", "Directory for split output")

	return c
}

// moduleDepsPlatformHelp is the --platform help text of the commands that
// render for a module's author (module build, module vet): without the flag
// they render against the module's own deps, not a configured platform.
const moduleDepsPlatformHelp = "Render against this platform module directory instead of the module's own deps"

// useModuleDepsPlatformHelp overwrites the shared --platform usage text that
// cmdutil.RenderFlags registered on c.
func useModuleDepsPlatformHelp(c *cobra.Command) {
	c.Flags().Lookup("platform").Usage = moduleDepsPlatformHelp
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
			Err:  fmt.Errorf("module build expects a directory; CUE packages span all files in a dir. Use 'opm instance build %s' for a instance file", modulePath),
		}
	}

	outputFormat, err := render.ParseManifestOutputFormat(outputFmt)
	if err != nil {
		return err
	}

	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:        cfg,
		NamespaceFlag: rf.Namespace,
	})
	if err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}

	result, err := render.FromModule(ctx, render.ModuleOpts{
		ModulePath:       modulePath,
		ValuesFiles:      rf.Values,
		Name:             nameFlag,
		PlatformFlag:     rf.Platform, // offline: no cluster read (0006:D21)
		PlatformFromDeps: true,
		K8sConfig:        k8sConfig,
		Config:           cfg,
	})
	if err != nil {
		return err
	}

	render.ShowOutput(result, render.ShowOutputOpts{Verbose: cfg.Flags.Verbose})

	return render.WriteManifestOutput(result.Resources, outputFormat, split, outDir, result.Instance.Name)
}
