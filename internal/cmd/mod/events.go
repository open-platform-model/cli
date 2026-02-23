package mod

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	oerrors "github.com/opmodel/cli/internal/errors"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
)

// NewModEventsCmd creates the mod events command.
func NewModEventsCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rsf cmdutil.ReleaseSelectorFlags
	var kf cmdutil.K8sFlags

	var (
		sinceFlag          string
		typeFlag           string
		watchFlag          bool
		outputFlag         string
		ignoreNotFoundFlag bool
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
  5  No resources found (override with --ignore-not-found to exit 0)

Examples:
  # Show events from the last hour (default)
  opm mod events --release-name jellyfin -n media

  # Show events from the last 30 minutes
  opm mod events --release-name jellyfin -n media --since 30m

  # Show only warnings
  opm mod events --release-name jellyfin -n media --type Warning

  # Stream events in real-time
  opm mod events --release-name jellyfin -n media --watch

  # JSON output for tooling
  opm mod events --release-name jellyfin -n media -o json`,
		RunE: func(c *cobra.Command, args []string) error {
			return runEvents(args, cfg, &rsf, &kf, sinceFlag, typeFlag, watchFlag, outputFlag, ignoreNotFoundFlag)
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
	c.Flags().BoolVar(&ignoreNotFoundFlag, "ignore-not-found", false,
		"Exit 0 when no resources match the selector")

	return c
}

// runEvents executes the events command.
//
//nolint:gocyclo // linear validation + dispatch; each branch is distinct
func runEvents(_ []string, cfg *config.GlobalConfig, rsf *cmdutil.ReleaseSelectorFlags, kf *cmdutil.K8sFlags, since, eventType string, watchMode bool, outputFmt string, ignoreNotFound bool) error {
	ctx := context.Background()

	// Validate release selector flags.
	if err := rsf.Validate(); err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err}
	}

	// Validate --type flag.
	if eventType != "" && eventType != "Normal" && eventType != "Warning" {
		return &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("invalid --type %q (valid: Normal, Warning)", eventType),
		}
	}

	// Validate --output flag.
	outputFormat, valid := output.ParseFormat(outputFmt)
	if !valid || outputFormat == output.FormatWide || outputFormat == output.FormatDir {
		return &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("invalid output format %q (valid: table, json, yaml)", outputFmt),
		}
	}

	// Parse --since flag.
	sinceCutoff, err := kubernetes.ParseSince(since)
	if err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err}
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
	inv, liveResources, _, err := cmdutil.ResolveInventory(ctx, k8sClient, rsf, namespace, ignoreNotFound, releaseLog)
	if err != nil {
		return err
	}
	if inv == nil {
		// ignoreNotFound was true and release was not found.
		return nil
	}

	eventsOpts := kubernetes.EventsOptions{
		Namespace:     namespace,
		ReleaseName:   rsf.ReleaseName,
		ReleaseID:     rsf.ReleaseID,
		Since:         sinceCutoff,
		EventType:     eventType,
		OutputFormat:  outputFormat,
		InventoryLive: liveResources,
	}

	if watchMode {
		return runEventsWatch(ctx, k8sClient, eventsOpts, logName)
	}

	// One-shot mode.
	result, err := kubernetes.GetModuleEvents(ctx, k8sClient, eventsOpts)
	if err != nil {
		if kubernetes.IsNoResourcesFound(err) {
			if ignoreNotFound {
				releaseLog.Info("no resources found (ignored)")
				return nil
			}
			releaseLog.Error("getting events", "error", err)
			return &oerrors.ExitError{Code: oerrors.ExitNotFound, Err: err, Printed: true}
		}
		releaseLog.Error("getting events", "error", err)
		return &oerrors.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: err, Printed: true}
	}

	formatted, err := kubernetes.FormatEvents(result, outputFormat)
	if err != nil {
		releaseLog.Error("formatting events", "error", err)
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err, Printed: true}
	}

	output.Println(formatted)
	return nil
}

// runEventsWatch streams events in real-time using the Kubernetes Watch API.
//
//nolint:gocyclo // watch loop with multiple filter branches
func runEventsWatch(ctx context.Context, client *kubernetes.Client, opts kubernetes.EventsOptions, logName string) error {
	releaseLog := output.ReleaseLogger(logName)

	// Set up signal handling for clean exit.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Discover children and collect UID set.
	children, err := kubernetes.DiscoverChildren(ctx, client, opts.InventoryLive, opts.Namespace)
	if err != nil {
		releaseLog.Error("discovering children", "error", err)
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("discovering children: %w", err)}
	}

	uidSet := make(map[types.UID]bool)
	for _, res := range opts.InventoryLive {
		uidSet[res.GetUID()] = true
	}
	for _, child := range children {
		uidSet[child.GetUID()] = true
	}

	// Start watch on events in namespace.
	watcher, err := client.Clientset.CoreV1().Events(opts.Namespace).Watch(ctx, metav1.ListOptions{})
	if err != nil {
		releaseLog.Error("starting event watch", "error", err)
		return &oerrors.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: fmt.Errorf("starting event watch: %w", err)}
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return nil // channel closed
			}
			if event.Type != watch.Added && event.Type != watch.Modified {
				continue
			}

			ev, ok := event.Object.(*corev1.Event)
			if !ok {
				continue
			}

			// Filter by UID set.
			if !uidSet[ev.InvolvedObject.UID] {
				continue
			}

			// Filter by event type.
			if opts.EventType != "" && ev.Type != opts.EventType {
				continue
			}

			// Filter by since.
			if !opts.Since.IsZero() {
				evTime := ev.LastTimestamp.Time
				if evTime.IsZero() {
					evTime = ev.CreationTimestamp.Time
				}
				if evTime.Before(opts.Since) {
					continue
				}
			}

			output.Println(kubernetes.FormatSingleEventLine(ev))
		}
	}
}
