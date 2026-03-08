package release

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
	"github.com/opmodel/cli/internal/kubernetes"
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

	if eventType != "" && eventType != "Normal" && eventType != "Warning" {
		return &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("invalid --type %q (valid: Normal, Warning)", eventType),
		}
	}

	outputFormat, valid := output.ParseFormat(outputFmt)
	if !valid || outputFormat == output.FormatWide || outputFormat == output.FormatDir {
		return &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("invalid output format %q (valid: table, json, yaml)", outputFmt),
		}
	}

	sinceCutoff, err := kubernetes.ParseSince(since)
	if err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err}
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
		return runReleaseEventsWatch(ctx, k8sClient, eventsOpts, logName)
	}

	result, err := kubernetes.GetModuleEvents(ctx, k8sClient, eventsOpts)
	if err != nil {
		if kubernetes.IsNoResourcesFound(err) {
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

func runReleaseEventsWatch(ctx context.Context, client *kubernetes.Client, opts kubernetes.EventsOptions, logName string) error { //nolint:gocyclo // watch loop with multiple filter branches
	releaseLog := output.ReleaseLogger(logName)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() { <-sigCh; cancel() }()

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
				return nil
			}
			if event.Type != watch.Added && event.Type != watch.Modified {
				continue
			}
			ev, ok := event.Object.(*corev1.Event)
			if !ok {
				continue
			}
			if !uidSet[ev.InvolvedObject.UID] {
				continue
			}
			if opts.EventType != "" && ev.Type != opts.EventType {
				continue
			}
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
