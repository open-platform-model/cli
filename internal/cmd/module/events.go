package modulecmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/internal/workflow/query"
	oerrors "github.com/opmodel/cli/pkg/errors"
)

// NewModuleEventsCmd creates the module events command.
func NewModuleEventsCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rsf cmdutil.ReleaseSelectorFlags
	var kf cmdutil.K8sFlags

	var (
		sinceFlag  string
		typeFlag   string
		watchFlag  bool
		outputFlag string
	)

	c := &cobra.Command{
		Use:   "events",
		Short: "Show events for a release",
		Long: `Show Kubernetes events for all resources belonging to an OPM release.

Events are collected from OPM-managed resources and their Kubernetes-owned
children (Pods, ReplicaSets). This surfaces diagnostic information like
OOMKilled, ImagePullBackOff, and scheduling failures that live on child
resources not directly tracked by OPM.

Exactly one of --release-name or --release-id is required to identify the release.
The --namespace flag defaults to the value configured in ~/.opm/config.cue.

Note: Kubernetes garbage-collects events after ~1 hour by default. The --since
flag defaults to 1h to match this retention window.

Exit codes:
  0  Success (events displayed, or no events found)
  1  Command error (cluster unreachable, permission denied, etc.)
  5  No resources found

Examples:
  # Show events from the last hour (default)
  opm module events --release-name jellyfin -n media

  # Show events from the last 30 minutes
  opm module events --release-name jellyfin -n media --since 30m

  # Show only warnings
  opm module events --release-name jellyfin -n media --type Warning

  # Stream events in real-time
  opm module events --release-name jellyfin -n media --watch

  # JSON output for tooling
  opm module events --release-name jellyfin -n media -o json`,
		RunE: func(c *cobra.Command, args []string) error {
			return runModuleEvents(args, cfg, &rsf, &kf, sinceFlag, typeFlag, watchFlag, outputFlag)
		},
	}

	rsf.AddTo(c)
	kf.AddTo(c)

	c.Flags().StringVar(&sinceFlag, "since", "1h",
		"Time window for events (e.g., 30m, 1h, 2h30m, 1d, 7d)")
	c.Flags().StringVar(&typeFlag, "type", "",
		"Filter by event type: Normal, Warning")
	c.Flags().BoolVar(&watchFlag, "watch", false,
		"Stream new events in real-time")
	c.Flags().StringVarP(&outputFlag, "output", "o", "table",
		"Output format (table, json, yaml)")
	return c
}

// runEvents executes the events command.
//
//nolint:gocyclo // linear validation + dispatch; each branch is distinct
func runModuleEvents(_ []string, cfg *config.GlobalConfig, rsf *cmdutil.ReleaseSelectorFlags, kf *cmdutil.K8sFlags, since, eventType string, watchMode bool, outputFmt string) error {
	ctx := context.Background()

	// Validate release selector flags.
	if err := rsf.Validate(); err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err}
	}

	eventsOpts, err := query.ParseEventsOptions(since, eventType, outputFmt, watchMode)
	if err != nil {
		return err
	}

	// Resolve Kubernetes configuration.
	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
		NamespaceFlag:  rsf.Namespace,
	})
	if err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}
	if err := cmdutil.RequireNamespace(k8sConfig); err != nil {
		return err
	}

	namespace := k8sConfig.Namespace.Value

	output.Debug("resolved kubernetes config",
		"kubeconfig", k8sConfig.Kubeconfig.Value,
		"context", k8sConfig.Context.Value,
		"namespace", namespace,
	)

	logName := rsf.LogName()
	releaseLog := output.ReleaseLogger(logName)

	// Create Kubernetes client.
	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		releaseLog.Error("connecting to cluster", "error", err)
		return err
	}

	// Resolve inventory and live resources.
	_, liveResources, _, err := query.ResolveInventory(ctx, k8sClient, rsf, namespace, releaseLog)
	if err != nil {
		return err
	}

	eventsOpts.Namespace = namespace
	eventsOpts.ReleaseName = rsf.ReleaseName
	eventsOpts.ReleaseID = rsf.ReleaseID
	eventsOpts.InventoryLive = liveResources

	if watchMode {
		return query.WatchEvents(ctx, k8sClient, eventsOpts, logName)
	}
	return query.PrintEvents(ctx, k8sClient, eventsOpts, logName)
}
