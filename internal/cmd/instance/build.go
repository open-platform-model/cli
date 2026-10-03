package instance

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

// NewInstanceBuildCmd creates the instance build command.
func NewInstanceBuildCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rff cmdutil.InstanceFileFlags
	var kf cmdutil.K8sFlags
	var namespace string
	var offline bool

	var (
		outputFlag string
		splitFlag  bool
		outDirFlag string
	)

	c := &cobra.Command{
		Use:   "build <instance.cue | instance-dir>",
		Short: "Render an instance to manifests",
		Long: `Render an OPM instance to Kubernetes manifests.

The argument names an instance package: a .cue file (the instance is the
package in the file's directory) or a directory holding the package. The
directory may be a CUE module of its own or a package inside another module,
such as an instance directory within a module tree. What the package is
decides, not its file names: a module package is refused with the command
that builds it, 'opm module build <dir>'.

The platform is --platform <dir>, else the cluster's Platform when the
kubeconfig context reaches one, else a platform generated from the instance
package's own dependency pins. The cluster is never required: an absent
Platform or an unreachable cluster warns and falls back to the deps, and
--offline skips the cluster entirely.

Arguments:
  instance.cue    Path to an instance .cue file
  instance-dir    Path to an instance package directory

Examples:
  # Build an instance file
  opm instance build ./jellyfin_instance.cue

  # Build an instance package directory
  opm instance build ./instances/jellyfin

  # Build with split output
  opm instance build ./jellyfin_instance.cue --split --out-dir ./manifests

  # Build as JSON
  opm instance build ./jellyfin_instance.cue -o json

  # Build without contacting any cluster
  opm instance build ./jellyfin_instance.cue --offline`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runInstanceBuild(args[0], cfg, &rff, clusterLookup{k8s: kf, offline: offline}, namespace, outputFlag, splitFlag, outDirFlag)
		},
	}

	rff.AddTo(c)
	kf.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", renderNamespaceFlagHelp)
	c.Flags().BoolVar(&offline, "offline", false, offlineFlagHelp)
	c.Flags().StringVarP(&outputFlag, "output", "o", "yaml", "Output format: yaml, json")
	c.Flags().BoolVar(&splitFlag, "split", false, "Write separate files per resource")
	c.Flags().StringVar(&outDirFlag, "out-dir", "./manifests", "Directory for split output")

	return c
}

// runInstanceBuild executes the instance build command.
func runInstanceBuild(buildArg string, cfg *config.GlobalConfig, rff *cmdutil.InstanceFileFlags, lookup clusterLookup, namespaceFlag, outputFmt string, split bool, outDir string) error {
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

	if _, statErr := os.Stat(buildArg); statErr != nil {
		if os.IsNotExist(statErr) {
			return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("path %q not found", buildArg)}
		}
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("stat %q: %w", buildArg, statErr)}
	}

	result, err := render.FromInstanceFile(ctx, render.InstanceFileOpts{
		PlatformFlag:     rff.Platform,
		ClusterPlatform:  optionalClusterGetter(cfg, lookup.k8s, rff.Platform, lookup.offline),
		ClusterOptional:  true,
		InstanceFilePath: buildArg,
		ModuleCommand:    "opm module build",
		ValuesFiles:      rff.Values,
		SkipUnprovided:   rff.SkipUnprovided,
		K8sConfig:        k8sConfig,
		Config:           cfg,
	})
	if err != nil {
		return err
	}

	render.ShowOutput(result, render.ShowOutputOpts{Verbose: cfg.Flags.Verbose})

	return render.WriteManifestOutput(result.Resources, outputFormat, split, outDir, result.Instance.Name)
}
