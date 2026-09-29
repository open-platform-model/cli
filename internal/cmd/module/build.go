package modulecmd

import (
	"context"
	"fmt"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/workflow/render"
)

// NewModuleBuildCmd creates the module build command.
func NewModuleBuildCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rf cmdutil.RenderFlags
	var nameFlag, versionFlag string

	var (
		outputFlag string
		splitFlag  bool
		outDirFlag string
	)

	c := &cobra.Command{
		Use:   "build [path | module-path]",
		Short: "Render a module to manifests via synthetic instance",
		Long: `Render an OPM module to Kubernetes manifests by synthesizing a
#ModuleInstance around it. The module is a package directory on disk or a
published module named by its module path. Values come from the module's
debugValues (default) or from -f/--values files.

The render answers whether the module renders with the catalogs it declares:
by default it runs against a platform generated from the module's own
cue.mod/module.cue, one registry entry per catalog the module pins, at the
pinned version. The cluster is not read. Pass --platform <dir> to render
against a platform module instead, for example one pulled with
'opm platform pull'.

A published module is fetched from the registry; nothing is written to disk
except CUE's module cache. --version v1 takes the newest release of major 1,
--version 1.0.4 pins that release, and no --version takes the newest release
of the highest major built on this CLI's core. The chosen version is reported
on standard error.

Arguments:
  path          Module package directory (default: current directory).
                "." or a ./, ../ or absolute path is always a directory.
  module-path   Published module path without a major, e.g.
                opmodel.dev/modules/web_app

Examples:
  # Build the current module against its own deps using debugValues
  opm module build

  # Build a specific module with custom values
  opm module build ./my-module -f overrides.cue

  # Build against a platform module instead of the module's deps
  opm module build ./my-module --platform ./pulled-platform

  # Build with a custom synthetic instance name
  opm module build ./my-module --name my-debug

  # Build the newest compatible release of a published module
  opm module build opmodel.dev/modules/web_app

  # Build a pinned release of a published module with custom values
  opm module build opmodel.dev/modules/web_app --version 1.0.4 -f values.cue`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runModuleBuild(args, cfg, &rf, nameFlag, versionFlag, outputFlag, splitFlag, outDirFlag)
		},
	}

	rf.AddTo(c)
	useModuleDepsPlatformHelp(c)
	c.Flags().StringVar(&nameFlag, "name", "", "Override synthetic instance name")
	c.Flags().StringVar(&versionFlag, "version", "", versionFlagHelp)
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

// versionFlagHelp is the --version usage text of module build and apply.
const versionFlagHelp = "Version of a published module: vN takes the newest in major N, X.Y.Z pins (default: highest major on this CLI's core)"

func runModuleBuild(args []string, cfg *config.GlobalConfig, rf *cmdutil.RenderFlags, nameFlag, versionFlag, outputFmt string, split bool, outDir string) error {
	ctx := context.Background()

	moduleArg, err := cmdutil.ResolveModuleArg(ctx, cfg, args, versionFlag, "build")
	if err != nil {
		return err
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
		ModulePath:   moduleArg.Dir,
		Published:    moduleArg.Published,
		ValuesFiles:  rf.Values,
		Name:         nameFlag,
		PlatformFlag: rf.Platform, // else the module's own deps; never the cluster
		K8sConfig:    k8sConfig,
		Config:       cfg,
	})
	if err != nil {
		return err
	}

	render.ShowOutput(result, render.ShowOutputOpts{Verbose: cfg.Flags.Verbose})

	return render.WriteManifestOutput(result.Resources, outputFormat, split, outDir, result.Instance.Name)
}
