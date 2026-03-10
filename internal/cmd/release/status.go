package release

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
)

// NewReleaseStatusCmd creates the release status command.
func NewReleaseStatusCmd(cfg *config.GlobalConfig) *cobra.Command {
	var kf cmdutil.K8sFlags
	var namespace string

	var (
		outputFlag  string
		detailsFlag bool
	)

	c := &cobra.Command{
		Use:   "status <file|name|uuid>",
		Short: "Show resource status for a release",
		Long: `Show status of resources deployed by an OPM release.

Arguments:
  file         Path to a release.cue file or directory containing one.
               The release name and namespace are read from the file's metadata.
               --namespace overrides the namespace found in the file.
  name         Release name (use -n / --namespace to scope by namespace).
  uuid         Release UUID.

Examples:
  # Identify by release.cue file in the current directory
  opm release status .

  # Identify by release.cue file path
  opm release status ./releases/jellyfin/release.cue

  # Identify by name
  opm release status jellyfin -n media

  # Wide output
  opm release status jellyfin -n media -o wide`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runReleaseStatus(args[0], cfg, &kf, namespace, outputFlag, detailsFlag)
		},
	}

	kf.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Target namespace (default: from config)")
	c.Flags().StringVarP(&outputFlag, "output", "o", "table", "Output format (table, wide, yaml, json)")
	c.Flags().BoolVar(&detailsFlag, "details", false, "Show pod-level diagnostics for unhealthy workloads")

	return c
}

func runReleaseStatus(identifier string, cfg *config.GlobalConfig, kf *cmdutil.K8sFlags, namespaceFlag, outputFmt string, verbose bool) error {
	ctx := context.Background()

	target, err := cmdutil.ResolveReleaseTarget(identifier, cfg, kf, namespaceFlag)
	if err != nil {
		return err
	}

	cmdutil.LogResolvedKubernetesConfig(target.Namespace, target.K8sConfig.Kubeconfig.Value, target.K8sConfig.Context.Value)

	logName := target.LogName
	releaseLog := output.ReleaseLogger(logName)

	outputFormat, err := cmdutil.ParseStatusOutputFormat(outputFmt)
	if err != nil {
		return err
	}

	k8sClient, err := cmdutil.NewK8sClient(target.K8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		releaseLog.Error("connecting to cluster", "error", err)
		return err
	}

	inv, liveResources, missingEntries, err := cmdutil.ResolveInventory(ctx, k8sClient, target.Selector, target.Namespace, releaseLog)
	if err != nil {
		return err
	}

	statusOpts := cmdutil.BuildStatusOptions(target.Namespace, target.Selector, outputFormat, verbose, inv, liveResources, missingEntries)
	return cmdutil.PrintReleaseStatus(ctx, k8sClient, statusOpts, logName)
}
