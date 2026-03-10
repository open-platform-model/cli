package modulecmd

import (
	"context"
	"fmt"

	opmexit "github.com/opmodel/cli/internal/exit"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/internal/workflow/query"
)

// NewModuleStatusCmd creates the module status command.
func NewModuleStatusCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rsf cmdutil.ReleaseSelectorFlags
	var kf cmdutil.K8sFlags

	// Status-specific flags (local to this command)
	var (
		outputFlag  string
		detailsFlag bool
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

Exit codes:
  0  All resources healthy
  1  Command error (cluster unreachable, permission denied, etc.)
  2  Resources exist but are not ready
  5  No resources found

Examples:
  # Show status by release name
  opm module status --release-name my-app -n production

  # Show status with extra columns (replicas, image)
  opm module status --release-name my-app -n production -o wide

  # Show pod-level diagnostics for unhealthy workloads
  opm module status --release-name my-app -n production --details

  # Show status in JSON format
  opm module status --release-name my-app -n production -o json`,
		RunE: func(c *cobra.Command, args []string) error {
			return runModuleStatus(args, cfg, &rsf, &kf, outputFlag, detailsFlag)
		},
	}

	rsf.AddTo(c)
	kf.AddTo(c)

	// Status-specific flags
	c.Flags().StringVarP(&outputFlag, "output", "o", "table",
		"Output format (table, wide, yaml, json)")
	c.Flags().BoolVar(&detailsFlag, "details", false,
		"Show pod-level diagnostics for unhealthy workloads")

	return c
}

// runStatus executes the status command.
func runModuleStatus(_ []string, cfg *config.GlobalConfig, rsf *cmdutil.ReleaseSelectorFlags, kf *cmdutil.K8sFlags, outputFmt string, verbose bool) error {
	ctx := context.Background()

	// Validate release selector flags
	if err := rsf.Validate(); err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
	}

	// Resolve Kubernetes configuration with local flags
	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
		NamespaceFlag:  rsf.Namespace,
	})
	if err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}
	if err := cmdutil.RequireNamespace(k8sConfig); err != nil {
		return err
	}

	namespace := k8sConfig.Namespace.Value

	// Log resolved k8s config at DEBUG level
	cmdutil.LogResolvedKubernetesConfig(namespace, k8sConfig.Kubeconfig.Value, k8sConfig.Context.Value)

	// Create scoped release logger using shared LogName helper
	logName := rsf.LogName()
	releaseLog := output.ReleaseLogger(logName)

	outputFormat, err := query.ParseStatusOutputFormat(outputFmt)
	if err != nil {
		return err
	}

	// Create Kubernetes client from pre-resolved config
	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		releaseLog.Error("connecting to cluster", "error", err)
		return err
	}

	inv, liveResources, missingEntries, err := query.ResolveInventory(ctx, k8sClient, rsf, namespace, releaseLog)
	if err != nil {
		return err
	}

	statusOpts := query.BuildStatusOptions(namespace, rsf, outputFormat, verbose, inv, liveResources, missingEntries)
	return query.PrintReleaseStatus(ctx, k8sClient, statusOpts, logName)
}
