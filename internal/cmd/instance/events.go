package instance

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/workflow/query"
)

// NewInstanceEventsCmd creates the instance events command.
func NewInstanceEventsCmd(cfg *config.GlobalConfig) *cobra.Command {
	var kf cmdutil.K8sFlags
	var namespace string

	var (
		sinceFlag  string
		typeFlag   string
		watchFlag  bool
		outputFlag string
	)

	c := &cobra.Command{
		Use:   "events <file|name|uuid>",
		Short: "Show events for an instance",
		Long: `Show Kubernetes events for all resources belonging to an OPM instance.

Arguments:
  file         Path to an instance.cue file or directory containing one.
               The instance name and namespace are read from the file's metadata.
               --namespace overrides the namespace found in the file.
  name         Instance name (use -n / --namespace to scope by namespace).
  uuid         Instance UUID.

Examples:
  # Identify by instance.cue file in the current directory
  opm instance events .

  # Identify by instance.cue file path
  opm instance events ./instances/jellyfin/instance.cue -n media

  # Identify by name
  opm instance events jellyfin -n media

  # Stream events in real-time
  opm instance events jellyfin -n media --watch`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runInstanceEvents(args[0], cfg, &kf, namespace, sinceFlag, typeFlag, watchFlag, outputFlag)
		},
	}

	kf.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Target namespace")
	c.Flags().StringVar(&sinceFlag, "since", "1h", "Time window for events (e.g., 30m, 1h, 2h30m)")
	c.Flags().StringVar(&typeFlag, "type", "", "Filter by event type: Normal, Warning")
	c.Flags().BoolVar(&watchFlag, "watch", false, "Stream new events in real-time")
	c.Flags().StringVarP(&outputFlag, "output", "o", "table", "Output format (table, json, yaml)")

	return c
}

func runInstanceEvents(identifier string, cfg *config.GlobalConfig, kf *cmdutil.K8sFlags, namespaceFlag, since, eventType string, watchMode bool, outputFmt string) error {
	ctx := context.Background()

	eventsOpts, err := query.ParseEventsOptions(since, eventType, outputFmt, watchMode)
	if err != nil {
		return err
	}

	target, err := cmdutil.ResolveInstanceTarget(identifier, cfg, kf, namespaceFlag)
	if err != nil {
		return err
	}

	logName := target.LogName
	instanceLog := output.InstanceLogger(logName)

	k8sClient, err := cmdutil.NewK8sClient(target.K8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		instanceLog.Error("connecting to cluster", "error", err)
		return err
	}

	_, liveResources, _, err := query.ResolveInventory(ctx, k8sClient, target.Selector, target.Namespace, instanceLog)
	if err != nil {
		return err
	}

	eventsOpts.Namespace = target.Namespace
	eventsOpts.InstanceName = target.Selector.InstanceName
	eventsOpts.InstanceID = target.Selector.InstanceID
	eventsOpts.InventoryLive = liveResources

	if watchMode {
		return query.WatchEvents(ctx, k8sClient, eventsOpts, logName)
	}
	return query.PrintEvents(ctx, k8sClient, eventsOpts, logName)
}
