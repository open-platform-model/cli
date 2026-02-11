package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
)

// Status command flags.
var (
	statusNamespaceFlag  string
	statusNameFlag       string
	statusReleaseIDFlag  string
	statusOutputFlag     string
	statusWatchFlag      bool
	statusKubeconfigFlag string
	statusContextFlag    string
)

// NewModStatusCmd creates the mod status command.
func NewModStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show resource status",
		Long: `Show status of resources deployed by an OPM module.

Resources are discovered via OPM labels, so the original module source is
not required. Health is evaluated per resource category:

  - Workloads (Deployment, StatefulSet, DaemonSet): Ready condition
  - Jobs: Complete condition
  - CronJobs: always healthy (scheduled)
  - Passive (ConfigMap, Secret, Service, PVC): healthy on creation
  - Custom (CRDs): Ready condition if present, else passive

Exactly one of --name or --release-id is required to identify the module deployment.
The --namespace flag is always required.

Examples:
  # Show status by module name
  opm mod status --name my-app -n production

  # Show status by release ID
  opm mod status --release-id a1b2c3d4-e5f6-7890-abcd-ef1234567890 -n production

  # Show status in JSON format
  opm mod status --name my-app -n production -o json

  # Watch status continuously
  opm mod status --name my-app -n production --watch`,
		RunE: runStatus,
	}

	// Add flags
	cmd.Flags().StringVarP(&statusNamespaceFlag, "namespace", "n", "",
		"Target namespace (required)")
	cmd.Flags().StringVar(&statusNameFlag, "name", "",
		"Module name (mutually exclusive with --release-id)")
	cmd.Flags().StringVar(&statusReleaseIDFlag, "release-id", "",
		"Release identity UUID (mutually exclusive with --name)")
	cmd.Flags().StringVarP(&statusOutputFlag, "output", "o", "table",
		"Output format (table, yaml, json)")
	cmd.Flags().BoolVar(&statusWatchFlag, "watch", false,
		"Watch status continuously (poll every 2s)")
	cmd.Flags().StringVar(&statusKubeconfigFlag, "kubeconfig", "",
		"Path to kubeconfig file")
	cmd.Flags().StringVar(&statusContextFlag, "context", "",
		"Kubernetes context to use")

	// Namespace is always required
	_ = cmd.MarkFlagRequired("namespace")

	return cmd
}

// runStatus executes the status command.
func runStatus(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Manual validation: require exactly one of --name or --release-id (mutually exclusive)
	if statusNameFlag != "" && statusReleaseIDFlag != "" {
		return &ExitError{
			Code: ExitGeneralError,
			Err:  fmt.Errorf("--name and --release-id are mutually exclusive"),
		}
	}
	if statusNameFlag == "" && statusReleaseIDFlag == "" {
		return &ExitError{
			Code: ExitGeneralError,
			Err:  fmt.Errorf("either --name or --release-id is required"),
		}
	}

	// Resolve flags with global fallback
	kubeconfig := resolveFlag(statusKubeconfigFlag, GetKubeconfig())
	kubeContext := resolveFlag(statusContextFlag, GetContext())

	// Create scoped module logger - prefer name, fall back to release-id
	logName := statusNameFlag
	if logName == "" {
		logName = fmt.Sprintf("release:%s", statusReleaseIDFlag[:8])
	}
	modLog := output.ModuleLogger(logName)

	// Validate output format
	switch statusOutputFlag {
	case "table", "yaml", "json":
		// valid
	default:
		return &ExitError{
			Code: ExitGeneralError,
			Err:  fmt.Errorf("invalid output format %q: must be table, yaml, or json", statusOutputFlag),
		}
	}

	// Create Kubernetes client
	k8sClient, err := kubernetes.NewClient(kubernetes.ClientOptions{
		Kubeconfig:  kubeconfig,
		Context:     kubeContext,
		APIWarnings: opmConfig.Config.Log.Kubernetes.APIWarnings,
	})
	if err != nil {
		modLog.Error("connecting to cluster", "error", err)
		return &ExitError{Code: ExitConnectivityError, Err: err}
	}

	statusOpts := kubernetes.StatusOptions{
		Namespace:    statusNamespaceFlag,
		Name:         statusNameFlag,
		ReleaseID:    statusReleaseIDFlag,
		OutputFormat: statusOutputFlag,
		Watch:        statusWatchFlag,
	}

	// If watch mode, run in loop
	if statusWatchFlag {
		return runStatusWatch(ctx, k8sClient, statusOpts)
	}

	// Single run
	return runStatusOnce(ctx, k8sClient, statusOpts)
}

// runStatusOnce executes a single status check.
func runStatusOnce(ctx context.Context, client *kubernetes.Client, opts kubernetes.StatusOptions) error {
	logName := opts.Name
	if logName == "" {
		logName = fmt.Sprintf("release:%s", opts.ReleaseID[:8])
	}
	modLog := output.ModuleLogger(logName)

	result, err := kubernetes.GetModuleStatus(ctx, client, opts)
	if err != nil {
		modLog.Error("getting status", "error", err)
		return &ExitError{Code: ExitGeneralError, Err: err}
	}

	formatted, err := kubernetes.FormatStatus(result, opts.OutputFormat)
	if err != nil {
		modLog.Error("formatting status", "error", err)
		return &ExitError{Code: ExitGeneralError, Err: err}
	}

	output.Println(formatted)
	return nil
}

// runStatusWatch runs status in continuous watch mode, polling every 2 seconds.
func runStatusWatch(ctx context.Context, client *kubernetes.Client, opts kubernetes.StatusOptions) error {
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
	if err := displayStatus(ctx, client, opts); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// Clear screen
			fmt.Fprint(os.Stdout, "\033[2J\033[H")
			if err := displayStatus(ctx, client, opts); err != nil {
				return err
			}
		}
	}
}

// displayStatus fetches and displays the current status.
func displayStatus(ctx context.Context, client *kubernetes.Client, opts kubernetes.StatusOptions) error {
	logName := opts.Name
	if logName == "" {
		logName = fmt.Sprintf("release:%s", opts.ReleaseID[:8])
	}
	modLog := output.ModuleLogger(logName)

	result, err := kubernetes.GetModuleStatus(ctx, client, opts)
	if err != nil {
		modLog.Error("getting status", "error", err)
		return &ExitError{Code: ExitGeneralError, Err: err}
	}

	// In watch mode, always use table format
	formatted := kubernetes.FormatStatusTable(result)
	output.Println(formatted)
	return nil
}
