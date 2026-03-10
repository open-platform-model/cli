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

// NewReleaseEventsCmd creates the release events command.
func NewReleaseEventsCmd(cfg *config.GlobalConfig) *cobra.Command {
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
		Short: "Show events for a release",
		Long: `Show Kubernetes events for all resources belonging to an OPM release.

Arguments:
  file         Path to a release.cue file or directory containing one.
               The release name and namespace are read from the file's metadata.
               --namespace overrides the namespace found in the file.
  name         Release name (use -n / --namespace to scope by namespace).
  uuid         Release UUID.

Examples:
  # Identify by release.cue file in the current directory
  opm release events .

  # Identify by release.cue file path
  opm release events ./releases/jellyfin/release.cue -n media

  # Identify by name
  opm release events jellyfin -n media

  # Stream events in real-time
  opm release events jellyfin -n media --watch`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runReleaseEvents(args[0], cfg, &kf, namespace, sinceFlag, typeFlag, watchFlag, outputFlag)
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

func runReleaseEvents(identifier string, cfg *config.GlobalConfig, kf *cmdutil.K8sFlags, namespaceFlag, since, eventType string, watchMode bool, outputFmt string) error { //nolint:gocyclo // linear validation + dispatch
	ctx := context.Background()

	eventsOpts, err := cmdutil.ParseEventsOptions(since, eventType, outputFmt, watchMode)
	if err != nil {
		return err
	}

	ra, err := cmdutil.ResolveReleaseArg(identifier, cfg)
	if err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err}
	}
	rsf := ra.ToSelectorFlags(namespaceFlag)

	if err := rsf.Validate(); err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err}
	}

	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
		NamespaceFlag:  ra.EffectiveNamespace(namespaceFlag),
	})
	if err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}
	if err := cmdutil.RequireNamespace(k8sConfig); err != nil {
		return err
	}

	namespace := k8sConfig.Namespace.Value
	logName := rsf.LogName()
	releaseLog := output.ReleaseLogger(logName)

	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		releaseLog.Error("connecting to cluster", "error", err)
		return err
	}

	_, liveResources, _, err := cmdutil.ResolveInventory(ctx, k8sClient, rsf, namespace, releaseLog)
	if err != nil {
		return err
	}

	eventsOpts.Namespace = namespace
	eventsOpts.ReleaseName = rsf.ReleaseName
	eventsOpts.ReleaseID = rsf.ReleaseID
	eventsOpts.InventoryLive = liveResources

	if watchMode {
		return cmdutil.WatchEvents(ctx, k8sClient, eventsOpts, logName)
	}
	return cmdutil.PrintEvents(ctx, k8sClient, eventsOpts, logName)
}
