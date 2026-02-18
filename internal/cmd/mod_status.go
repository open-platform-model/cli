package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
)

// NewModStatusCmd creates the mod status command.
func NewModStatusCmd() *cobra.Command {
	var rsf cmdutil.ReleaseSelectorFlags
	var kf cmdutil.K8sFlags

	// Status-specific flags (local to this command)
	var (
		outputFlag         string
		watchFlag          bool
		ignoreNotFoundFlag bool
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show resource status",
		Long: `Show status of resources deployed by an OPM release.

Resources are discovered via OPM labels, so the original module source is
not required. Health is evaluated per resource category:

  - Workloads (Deployment, StatefulSet, DaemonSet): Ready condition
  - Jobs: Complete condition
  - CronJobs: always healthy (scheduled)
  - Passive (ConfigMap, Secret, Service, PVC): healthy on creation
  - Custom (CRDs): Ready condition if present, else passive

Exactly one of --release-name or --release-id is required to identify the release deployment.
The --namespace flag is always required.

Examples:
  # Show status by release name
  opm mod status --release-name my-app -n production

  # Show status by release ID
  opm mod status --release-id a1b2c3d4-e5f6-7890-abcd-ef1234567890 -n production

  # Show status in JSON format
  opm mod status --release-name my-app -n production -o json

  # Watch status continuously
  opm mod status --release-name my-app -n production --watch`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cmd, args, &rsf, &kf, outputFlag, watchFlag, ignoreNotFoundFlag)
		},
	}

	rsf.AddTo(cmd)
	kf.AddTo(cmd)

	// Status-specific flags
	cmd.Flags().StringVarP(&outputFlag, "output", "o", "table",
		"Output format (table, yaml, json)")
	cmd.Flags().BoolVar(&watchFlag, "watch", false,
		"Watch status continuously (poll every 2s)")
	cmd.Flags().BoolVar(&ignoreNotFoundFlag, "ignore-not-found", false,
		"Exit 0 when no resources match the selector")

	return cmd
}

// runStatus executes the status command.
func runStatus(_ *cobra.Command, _ []string, rsf *cmdutil.ReleaseSelectorFlags, kf *cmdutil.K8sFlags, outputFmt string, watch, ignoreNotFound bool) error {
	ctx := context.Background()

	// Validate release selector flags
	if err := rsf.Validate(); err != nil {
		return &ExitError{Code: ExitGeneralError, Err: err}
	}

	// Resolve Kubernetes configuration with local flags
	k8sConfig, err := cmdutil.ResolveKubernetes(
		GetOPMConfig(),
		kf.Kubeconfig,
		kf.Context,
		rsf.Namespace,
		"", // no provider flag for status
	)
	if err != nil {
		return &ExitError{Code: ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}

	namespace := k8sConfig.Namespace.Value

	// Log resolved k8s config at DEBUG level
	output.Debug("resolved kubernetes config",
		"kubeconfig", k8sConfig.Kubeconfig.Value,
		"context", k8sConfig.Context.Value,
		"namespace", namespace,
	)

	// Create scoped module logger using shared LogName helper
	logName := rsf.LogName()
	modLog := output.ModuleLogger(logName)

	// Validate output format
	outputFormat, valid := output.ParseFormat(outputFmt)
	if !valid || outputFormat == output.FormatDir {
		return &ExitError{
			Code: ExitGeneralError,
			Err:  fmt.Errorf("invalid output format %q (valid: table, yaml, json)", outputFmt),
		}
	}

	opmConfig := GetOPMConfig()

	// Create Kubernetes client via shared factory
	k8sClient, err := cmdutil.NewK8sClient(kubernetes.ClientOptions{
		Kubeconfig:  kf.Kubeconfig,
		Context:     kf.Context,
		APIWarnings: opmConfig.Config.Log.Kubernetes.APIWarnings,
	})
	if err != nil {
		modLog.Error("connecting to cluster", "error", err)
		return err
	}

	statusOpts := kubernetes.StatusOptions{
		Namespace:    namespace,
		ReleaseName:  rsf.ReleaseName,
		ReleaseID:    rsf.ReleaseID,
		OutputFormat: outputFormat,
	}

	// Attempt inventory-first discovery when a release-id is provided.
	// Falls back to label-scan when no inventory exists (backward compatibility).
	if rsf.ReleaseID != "" {
		relName := rsf.ReleaseName
		if relName == "" {
			relName = rsf.ReleaseID
		}
		inv, invErr := inventory.GetInventory(ctx, k8sClient, relName, namespace, rsf.ReleaseID)
		if invErr != nil {
			output.Debug("could not read inventory for status, using label-scan", "error", invErr)
		} else if inv != nil {
			liveResources, missingEntries, invDiscoverErr := inventory.DiscoverResourcesFromInventory(ctx, k8sClient, inv)
			if invDiscoverErr != nil {
				output.Debug("inventory discovery failed, falling back to label-scan", "error", invDiscoverErr)
			} else {
				statusOpts.InventoryLive = liveResources
				for _, m := range missingEntries {
					statusOpts.MissingResources = append(statusOpts.MissingResources, kubernetes.MissingResource{
						Kind:      m.Kind,
						Namespace: m.Namespace,
						Name:      m.Name,
					})
				}
			}
		}
	}

	// If watch mode, run in loop
	if watch {
		return runStatusWatch(ctx, k8sClient, statusOpts, logName, ignoreNotFound)
	}

	// Single run
	return runStatusOnce(ctx, k8sClient, statusOpts, logName, ignoreNotFound)
}

// runStatusOnce executes a single status check.
func runStatusOnce(ctx context.Context, client *kubernetes.Client, opts kubernetes.StatusOptions, logName string, ignoreNotFound bool) error {
	modLog := output.ModuleLogger(logName)

	result, err := kubernetes.GetModuleStatus(ctx, client, opts)
	if err != nil {
		if ignoreNotFound && kubernetes.IsNoResourcesFound(err) {
			modLog.Info("no resources found (ignored)")
			return nil
		}
		modLog.Error("getting status", "error", err)
		return &ExitError{Code: ExitGeneralError, Err: err, Printed: true}
	}

	formatted, err := kubernetes.FormatStatus(result, opts.OutputFormat)
	if err != nil {
		modLog.Error("formatting status", "error", err)
		return &ExitError{Code: ExitGeneralError, Err: err, Printed: true}
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
	if err := displayStatus(ctx, client, opts, logName, ignoreNotFound); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// Clear screen
			output.ClearScreen()
			if err := displayStatus(ctx, client, opts, logName, ignoreNotFound); err != nil {
				return err
			}
		}
	}
}

// displayStatus fetches and displays the current status.
func displayStatus(ctx context.Context, client *kubernetes.Client, opts kubernetes.StatusOptions, logName string, ignoreNotFound bool) error {
	modLog := output.ModuleLogger(logName)

	result, err := kubernetes.GetModuleStatus(ctx, client, opts)
	if err != nil {
		if ignoreNotFound && kubernetes.IsNoResourcesFound(err) {
			modLog.Info("no resources found (ignored)")
			return nil
		}
		modLog.Error("getting status", "error", err)
		return &ExitError{Code: exitCodeFromK8sError(err), Err: err, Printed: true}
	}

	// In watch mode, always use table format
	formatted := kubernetes.FormatStatusTable(result)
	output.Println(formatted)
	return nil
}
