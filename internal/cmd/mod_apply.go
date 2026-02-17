package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
)

// NewModApplyCmd creates the mod apply command.
func NewModApplyCmd() *cobra.Command {
	var rf cmdutil.RenderFlags
	var kf cmdutil.K8sFlags

	// Apply-specific flags (local to this command)
	var (
		dryRunFlag   bool
		waitFlag     bool
		timeoutFlag  time.Duration
		createNSFlag bool
	)

	cmd := &cobra.Command{
		Use:   "apply [path]",
		Short: "Deploy module to cluster",
		Long: `Deploy an OPM module to a Kubernetes cluster using server-side apply.

This command renders the module and applies the resulting resources to the
cluster. Resources are applied in weight order (CRDs first, webhooks last).

All resources are labeled with OPM metadata for later discovery by
'opm mod delete' and 'opm mod status'.

Arguments:
  path    Path to module directory (default: current directory)

Examples:
  # Apply module in current directory
  opm mod apply

  # Apply with custom values and namespace
  opm mod apply ./my-module -f prod-values.cue -n production

  # Preview what would be applied
  opm mod apply --dry-run

  # Apply and wait for resources to be ready
  opm mod apply --wait --timeout 10m

  # Apply with verbose output showing transformer matches
  opm mod apply --verbose`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runApply(cmd, args, &rf, &kf, dryRunFlag, waitFlag, timeoutFlag, createNSFlag)
		},
	}

	rf.AddTo(cmd)
	kf.AddTo(cmd)

	// Apply-specific flags
	cmd.Flags().BoolVar(&dryRunFlag, "dry-run", false,
		"Server-side dry run (no changes made)")
	cmd.Flags().BoolVar(&waitFlag, "wait", false,
		"Wait for resources to be ready")
	cmd.Flags().DurationVar(&timeoutFlag, "timeout", 5*time.Minute,
		"Wait timeout")
	cmd.Flags().BoolVar(&createNSFlag, "create-namespace", false,
		"Create target namespace if it does not exist")

	return cmd
}

// runApply executes the apply command.
func runApply(_ *cobra.Command, args []string, rf *cmdutil.RenderFlags, kf *cmdutil.K8sFlags, dryRun, wait bool, timeout time.Duration, createNS bool) error {
	ctx := context.Background()

	opmConfig := GetOPMConfig()

	// Render module via shared pipeline
	result, err := cmdutil.RenderModule(ctx, cmdutil.RenderModuleOpts{
		Args:      args,
		Render:    rf,
		K8s:       kf,
		OPMConfig: opmConfig,
		Registry:  GetRegistry(),
	})
	if err != nil {
		return err
	}

	// Post-render: check errors, show matches, log warnings
	if err := cmdutil.ShowRenderOutput(result, cmdutil.ShowOutputOpts{
		Verbose: verboseFlag,
	}); err != nil {
		return err
	}

	// --- Apply-specific logic below ---

	// Create scoped module logger
	modLog := output.ModuleLogger(result.Module.Name)

	if len(result.Resources) == 0 {
		modLog.Info("no resources to apply")
		return nil
	}

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

	// Use the resolved namespace from the render result
	namespace := result.Module.Namespace

	// Create namespace if requested
	if createNS && namespace != "" {
		created, nsErr := k8sClient.EnsureNamespace(ctx, namespace, dryRun)
		if nsErr != nil {
			modLog.Error("ensuring namespace", "error", nsErr)
			return &ExitError{Code: exitCodeFromK8sError(nsErr), Err: nsErr, Printed: true}
		}
		if created {
			if dryRun {
				modLog.Info(fmt.Sprintf("namespace %q would be created", namespace))
			} else {
				modLog.Info(fmt.Sprintf("namespace %q created", namespace))
			}
		}
	}

	// Apply resources
	if dryRun {
		modLog.Info("dry run - no changes will be made")
	}
	modLog.Info(fmt.Sprintf("applying %d resources", len(result.Resources)))

	applyResult, err := kubernetes.Apply(ctx, k8sClient, result.Resources, result.Module, kubernetes.ApplyOptions{
		DryRun: dryRun,
	})
	if err != nil {
		modLog.Error("apply failed", "error", err)
		return &ExitError{Code: exitCodeFromK8sError(err), Err: err, Printed: true}
	}

	// Report results
	if len(applyResult.Errors) > 0 {
		modLog.Warn(fmt.Sprintf("%d resource(s) had errors", len(applyResult.Errors)))
		for _, e := range applyResult.Errors {
			modLog.Error(e.Error())
		}
	}

	if dryRun {
		modLog.Info(fmt.Sprintf("dry run complete: %d resources would be applied", applyResult.Applied))
	} else {
		modLog.Info(formatApplySummary(applyResult))
	}

	if len(applyResult.Errors) == 0 && !dryRun {
		if applyResult.Unchanged == applyResult.Applied {
			output.Println(output.FormatCheckmark("Module up to date"))
		} else {
			output.Println(output.FormatCheckmark("Module applied"))
		}
	}

	if len(applyResult.Errors) > 0 {
		return &ExitError{
			Code:    ExitGeneralError,
			Err:     fmt.Errorf("%d resource(s) failed to apply", len(applyResult.Errors)),
			Printed: true,
		}
	}

	return nil
}

// formatApplySummary builds a human-readable summary of apply results with
// per-status breakdown (e.g., "applied 5 resources (2 created, 1 configured, 2 unchanged)").
func formatApplySummary(r *kubernetes.ApplyResult) string {
	var parts []string
	if r.Created > 0 {
		parts = append(parts, fmt.Sprintf("%d created", r.Created))
	}
	if r.Configured > 0 {
		parts = append(parts, fmt.Sprintf("%d configured", r.Configured))
	}
	if r.Unchanged > 0 {
		parts = append(parts, fmt.Sprintf("%d unchanged", r.Unchanged))
	}

	summary := fmt.Sprintf("applied %d resources successfully", r.Applied)
	if len(parts) > 0 {
		summary += fmt.Sprintf(" (%s)", strings.Join(parts, ", "))
	}
	return summary
}
