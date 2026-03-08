package release

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
	oerrors "github.com/opmodel/cli/pkg/errors"
)

// NewReleaseStatusCmd creates the release status command.
func NewReleaseStatusCmd(cfg *config.GlobalConfig) *cobra.Command {
	var kf cmdutil.K8sFlags
	var namespace string

	var (
		outputFlag  string
		watchFlag   bool
		verboseFlag bool
	)

	c := &cobra.Command{
		Use:   "status <name|uuid>",
		Short: "Show resource status for a release",
		Long: `Show status of resources deployed by an OPM release.

Arguments:
  name|uuid    Release name or UUID (required)

Examples:
  # Show status by release name
  opm release status jellyfin -n media

  # Watch status continuously
  opm release status jellyfin -n media --watch

  # Wide output
  opm release status jellyfin -n media -o wide`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runReleaseStatus(args[0], cfg, &kf, namespace, outputFlag, watchFlag, verboseFlag)
		},
	}

	kf.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Target namespace (default: from config)")
	c.Flags().StringVarP(&outputFlag, "output", "o", "table", "Output format (table, wide, yaml, json)")
	c.Flags().BoolVar(&watchFlag, "watch", false, "Watch status continuously (poll every 2s)")
	c.Flags().BoolVar(&verboseFlag, "verbose", false, "Show pod-level diagnostics for unhealthy workloads")

	return c
}

func runReleaseStatus(identifier string, cfg *config.GlobalConfig, kf *cmdutil.K8sFlags, namespaceFlag, outputFmt string, watch, verbose bool) error {
	ctx := context.Background()

	name, uuid := cmdutil.ResolveReleaseIdentifier(identifier)
	rsf := &cmdutil.ReleaseSelectorFlags{
		ReleaseName: name,
		ReleaseID:   uuid,
		Namespace:   namespaceFlag,
	}

	if err := rsf.Validate(); err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err}
	}

	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
		NamespaceFlag:  namespaceFlag,
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

	outputFormat, valid := output.ParseFormat(outputFmt)
	if !valid || outputFormat == output.FormatDir {
		return &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("invalid output format %q (valid: table, wide, yaml, json)", outputFmt),
		}
	}

	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		releaseLog.Error("connecting to cluster", "error", err)
		return err
	}

	inv, liveResources, missingEntries, err := cmdutil.ResolveInventory(ctx, k8sClient, rsf, namespace, releaseLog)
	if err != nil {
		return err
	}

	componentMap := make(map[string]string)
	if len(inv.Index) > 0 {
		if change, ok := inv.Changes[inv.Index[0]]; ok {
			for _, entry := range change.Inventory.Entries {
				key := entry.Kind + "/" + entry.Namespace + "/" + entry.Name
				componentMap[key] = entry.Component
			}
		}
	}

	var version string
	if len(inv.Index) > 0 {
		if change, ok := inv.Changes[inv.Index[0]]; ok {
			version = change.Source.Version
		}
	}

	wideMode := outputFormat == output.FormatWide
	statusOpts := kubernetes.StatusOptions{
		Namespace:     namespace,
		ReleaseName:   rsf.ReleaseName,
		ReleaseID:     rsf.ReleaseID,
		Version:       version,
		ComponentMap:  componentMap,
		OutputFormat:  outputFormat,
		InventoryLive: liveResources,
		Wide:          wideMode,
		Verbose:       verbose,
	}
	for _, m := range missingEntries {
		statusOpts.MissingResources = append(statusOpts.MissingResources, kubernetes.MissingResource{
			Kind:      m.Kind,
			Namespace: m.Namespace,
			Name:      m.Name,
		})
	}

	if watch {
		return runReleaseStatusWatch(ctx, k8sClient, statusOpts, logName)
	}
	return fetchAndPrintReleaseStatus(ctx, k8sClient, statusOpts, logName, false)
}

func fetchAndPrintReleaseStatus(ctx context.Context, client *kubernetes.Client, opts kubernetes.StatusOptions, logName string, forWatch bool) error {
	releaseLog := output.ReleaseLogger(logName)

	result, err := kubernetes.GetReleaseStatus(ctx, client, opts)
	if err != nil {
		if kubernetes.IsNoResourcesFound(err) {
			releaseLog.Error("getting status", "error", err)
			return &oerrors.ExitError{Code: oerrors.ExitNotFound, Err: err, Printed: true}
		}
		releaseLog.Error("getting status", "error", err)
		return &oerrors.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: err, Printed: true}
	}

	var formatted string
	if forWatch {
		formatted = kubernetes.FormatStatusTable(result)
	} else {
		formatted, err = kubernetes.FormatStatus(result, opts.OutputFormat)
		if err != nil {
			releaseLog.Error("formatting status", "error", err)
			return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err, Printed: true}
		}
	}
	output.Println(formatted)

	if result.AggregateStatus != "Ready" && result.AggregateStatus != "Complete" {
		return &oerrors.ExitError{Code: oerrors.ExitValidationError, Err: fmt.Errorf("release %q: %d resource(s) not ready", opts.ReleaseName, result.Summary.NotReady), Printed: true}
	}
	return nil
}

func runReleaseStatusWatch(ctx context.Context, client *kubernetes.Client, opts kubernetes.StatusOptions, logName string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() { <-sigCh; cancel() }()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	if err := fetchAndPrintReleaseStatus(ctx, client, opts, logName, true); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			output.ClearScreen()
			if err := fetchAndPrintReleaseStatus(ctx, client, opts, logName, true); err != nil {
				return err
			}
		}
	}
}
