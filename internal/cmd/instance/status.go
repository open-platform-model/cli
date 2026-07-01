package instance

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/workflow/query"
)

// NewInstanceStatusCmd creates the instance status command.
func NewInstanceStatusCmd(cfg *config.GlobalConfig) *cobra.Command {
	var kf cmdutil.K8sFlags
	var namespace string

	var (
		outputFlag  string
		detailsFlag bool
	)

	c := &cobra.Command{
		Use:   "status <file|name|uuid>",
		Short: "Show resource status for an instance",
		Long: `Show status of resources deployed by an OPM instance.

Arguments:
  file         Path to an instance.cue file or directory containing one.
               The instance name and namespace are read from the file's metadata.
               --namespace overrides the namespace found in the file.
  name         Instance name (use -n / --namespace to scope by namespace).
  uuid         Instance UUID.

Examples:
  # Identify by instance.cue file in the current directory
  opm instance status .

  # Identify by instance.cue file path
  opm instance status ./instances/jellyfin/instance.cue

  # Identify by name
  opm instance status jellyfin -n media

  # Wide output
  opm instance status jellyfin -n media -o wide`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runInstanceStatus(args[0], cfg, &kf, namespace, outputFlag, detailsFlag)
		},
	}

	kf.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Target namespace (default: from config)")
	c.Flags().StringVarP(&outputFlag, "output", "o", "table", "Output format (table, wide, yaml, json)")
	c.Flags().BoolVar(&detailsFlag, "details", false, "Show pod-level diagnostics for unhealthy workloads")

	return c
}

func runInstanceStatus(identifier string, cfg *config.GlobalConfig, kf *cmdutil.K8sFlags, namespaceFlag, outputFmt string, verbose bool) error {
	ctx := context.Background()

	target, err := cmdutil.ResolveInstanceTarget(identifier, cfg, kf, namespaceFlag)
	if err != nil {
		return err
	}

	cmdutil.LogResolvedKubernetesConfig(target.Namespace, target.K8sConfig.Kubeconfig.Value, target.K8sConfig.Context.Value)

	logName := target.LogName
	instanceLog := output.InstanceLogger(logName)

	outputFormat, err := query.ParseStatusOutputFormat(outputFmt)
	if err != nil {
		return err
	}

	k8sClient, err := cmdutil.NewK8sClient(target.K8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		instanceLog.Error("connecting to cluster", "error", err)
		return err
	}

	inv, liveResources, missingEntries, err := query.ResolveInventory(ctx, k8sClient, target.Selector, target.Namespace, instanceLog)
	if err != nil {
		return err
	}

	statusOpts := query.BuildStatusOptions(target.Namespace, target.Selector, outputFormat, verbose, inv, liveResources, missingEntries)
	return query.PrintInstanceStatus(ctx, k8sClient, statusOpts, logName)
}
