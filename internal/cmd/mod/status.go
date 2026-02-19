package mod

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdtypes"
	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
)

// NewModStatusCmd creates the mod status command.
func NewModStatusCmd(cfg *cmdtypes.GlobalConfig) *cobra.Command {
	var rsf cmdutil.ReleaseSelectorFlags
	var kf cmdutil.K8sFlags

	// Status-specific flags (local to this command)
	var (
		outputFlag         string
		watchFlag          bool
		ignoreNotFoundFlag bool
	)

	c := &cobra.Command{
		Use:   "status",
		Short: "Show resource status",
		Long: `Show status of resources deployed by an OPM release.

Resources are discovered via the release inventory Secret, so the original
module source is not required. Health is evaluated per resource category:

  - Workloads (Deployment, StatefulSet, DaemonSet): Ready condition
  - Jobs: Complete condition
  - CronJobs: always healthy (scheduled)
  - Passive (ConfigMap, Secret, Service, PVC): healthy on creation
  - Custom (CRDs): Ready condition if present, else passive

Exactly one of --release-name or --release-id is required to identify the release.
The --namespace flag defaults to the value configured in ~/.opm/config.cue.

Examples:
  # Show status by release name
  opm mod status --release-name my-app -n production

  # Show status by release ID
  opm mod status --release-id a1b2c3d4-e5f6-7890-abcd-ef1234567890 -n production

  # Show status in JSON format
  opm mod status --release-name my-app -n production -o json

  # Watch status continuously
  opm mod status --release-name my-app -n production --watch`,
		RunE: func(c *cobra.Command, args []string) error {
			return runStatus(args, cfg, &rsf, &kf, outputFlag, watchFlag, ignoreNotFoundFlag)
		},
	}

	rsf.AddTo(c)
	kf.AddTo(c)

	// Status-specific flags
	c.Flags().StringVarP(&outputFlag, "output", "o", "table",
		"Output format (table, yaml, json)")
	c.Flags().BoolVar(&watchFlag, "watch", false,
		"Watch status continuously (poll every 2s)")
	c.Flags().BoolVar(&ignoreNotFoundFlag, "ignore-not-found", false,
		"Exit 0 when no resources match the selector")

	return c
}

// runStatus executes the status command.
func runStatus(_ []string, cfg *cmdtypes.GlobalConfig, rsf *cmdutil.ReleaseSelectorFlags, kf *cmdutil.K8sFlags, outputFmt string, watch, ignoreNotFound bool) error {
	ctx := context.Background()

	// Validate release selector flags
	if err := rsf.Validate(); err != nil {
		return &cmdtypes.ExitError{Code: cmdtypes.ExitGeneralError, Err: err}
	}

	// Resolve Kubernetes configuration with local flags
	k8sConfig, err := cmdutil.ResolveKubernetes(
		cfg.OPMConfig,
		kf.Kubeconfig,
		kf.Context,
		rsf.Namespace,
		"", // no provider flag for status
	)
	if err != nil {
		return &cmdtypes.ExitError{Code: cmdtypes.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}

	namespace := k8sConfig.Namespace.Value

	// Log resolved k8s config at DEBUG level
	output.Debug("resolved kubernetes config",
		"kubeconfig", k8sConfig.Kubeconfig.Value,
		"context", k8sConfig.Context.Value,
		"namespace", namespace,
	)

	// Create scoped release logger using shared LogName helper
	logName := rsf.LogName()
	releaseLog := output.ReleaseLogger(logName)

	// Validate output format
	outputFormat, valid := output.ParseFormat(outputFmt)
	if !valid || outputFormat == output.FormatDir {
		return &cmdtypes.ExitError{
			Code: cmdtypes.ExitGeneralError,
			Err:  fmt.Errorf("invalid output format %q (valid: table, yaml, json)", outputFmt),
		}
	}

	// Create Kubernetes client from pre-resolved config
	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.OPMConfig.Config.Log.Kubernetes.APIWarnings)
	if err != nil {
		releaseLog.Error("connecting to cluster", "error", err)
		return err
	}

	// Resolve the inventory Secret for this release.
	// --release-id: direct GET by name (opm.<name>.<uuid>), with UUID label fallback.
	// --release-name: label scan on inventory Secrets only (FindInventoryByReleaseName).
	var inv *inventory.InventorySecret
	var invErr error
	switch {
	case rsf.ReleaseID != "":
		relName := rsf.ReleaseName
		if relName == "" {
			relName = rsf.ReleaseID
		}
		inv, invErr = inventory.GetInventory(ctx, k8sClient, relName, namespace, rsf.ReleaseID)
	case rsf.ReleaseName != "":
		inv, invErr = inventory.FindInventoryByReleaseName(ctx, k8sClient, rsf.ReleaseName, namespace)
	}
	if invErr != nil {
		releaseLog.Error("reading inventory", "error", invErr)
		return &cmdtypes.ExitError{Code: cmdtypes.ExitGeneralError, Err: fmt.Errorf("reading inventory: %w", invErr)}
	}
	if inv == nil {
		name := rsf.ReleaseName
		if name == "" {
			name = rsf.ReleaseID
		}
		notFound := &kubernetes.ReleaseNotFoundError{Name: name, Namespace: namespace}
		if ignoreNotFound {
			releaseLog.Info("release not found (ignored)")
			return nil
		}
		releaseLog.Error("release not found", "name", name, "namespace", namespace)
		return &cmdtypes.ExitError{Code: cmdtypes.ExitNotFound, Err: notFound, Printed: true}
	}

	liveResources, missingEntries, discoverErr := inventory.DiscoverResourcesFromInventory(ctx, k8sClient, inv)
	if discoverErr != nil {
		releaseLog.Error("discovering resources from inventory", "error", discoverErr)
		return &cmdtypes.ExitError{Code: cmdtypes.ExitGeneralError, Err: fmt.Errorf("discovering resources: %w", discoverErr)}
	}

	statusOpts := kubernetes.StatusOptions{
		Namespace:     namespace,
		ReleaseName:   rsf.ReleaseName,
		ReleaseID:     rsf.ReleaseID,
		OutputFormat:  outputFormat,
		InventoryLive: liveResources,
	}
	for _, m := range missingEntries {
		statusOpts.MissingResources = append(statusOpts.MissingResources, kubernetes.MissingResource{
			Kind:      m.Kind,
			Namespace: m.Namespace,
			Name:      m.Name,
		})
	}

	// If watch mode, run in loop
	if watch {
		return runStatusWatch(ctx, k8sClient, statusOpts, logName, ignoreNotFound)
	}

	// Single run
	return fetchAndPrintStatus(ctx, k8sClient, statusOpts, logName, ignoreNotFound, false)
}

// fetchAndPrintStatus fetches and displays the current status.
// forWatch controls whether table format is forced (true in watch mode).
func fetchAndPrintStatus(ctx context.Context, client *kubernetes.Client, opts kubernetes.StatusOptions, logName string, ignoreNotFound, forWatch bool) error {
	releaseLog := output.ReleaseLogger(logName)

	result, err := kubernetes.GetReleaseStatus(ctx, client, opts)
	if err != nil {
		if ignoreNotFound && kubernetes.IsNoResourcesFound(err) {
			releaseLog.Info("no resources found (ignored)")
			return nil
		}
		releaseLog.Error("getting status", "error", err)
		return &cmdtypes.ExitError{Code: cmdtypes.ExitCodeFromK8sError(err), Err: err, Printed: true}
	}

	var formatted string
	if forWatch {
		// In watch mode, always use table format
		formatted = kubernetes.FormatStatusTable(result)
	} else {
		formatted, err = kubernetes.FormatStatus(result, opts.OutputFormat)
		if err != nil {
			releaseLog.Error("formatting status", "error", err)
			return &cmdtypes.ExitError{Code: cmdtypes.ExitGeneralError, Err: err, Printed: true}
		}
	}

	output.Println(formatted)
	return nil
}

// runStatusWatch runs status in continuous watch mode, polling every 2 seconds.
func runStatusWatch(ctx context.Context, client *kubernetes.Client, opts kubernetes.StatusOptions, logName string, ignoreNotFound bool) error {
	// Set up signal handling for clean exit
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Initial display
	if err := fetchAndPrintStatus(ctx, client, opts, logName, ignoreNotFound, true); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// Clear screen
			output.ClearScreen()
			if err := fetchAndPrintStatus(ctx, client, opts, logName, ignoreNotFound, true); err != nil {
				return err
			}
		}
	}
}
