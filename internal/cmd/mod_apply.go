package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/build"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
)

// Apply command flags.
var (
	applyValuesFlags     []string
	applyNamespaceFlag   string
	applyReleaseNameFlag string
	applyProviderFlag    string
	applyDryRunFlag      bool
	applyWaitFlag        bool
	applyTimeoutFlag     time.Duration
	applyCreateNSFlag    bool
	applyKubeconfigFlag  string
	applyContextFlag     string
)

// NewModApplyCmd creates the mod apply command.
func NewModApplyCmd() *cobra.Command {
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
  opm mod apply --wait --timeout 10m`,
		Args: cobra.MaximumNArgs(1),
		RunE: runApply,
	}

	// Add flags
	cmd.Flags().StringArrayVarP(&applyValuesFlags, "values", "f", nil,
		"Additional values files (can be repeated)")
	cmd.Flags().StringVarP(&applyNamespaceFlag, "namespace", "n", "",
		"Target namespace")
	cmd.Flags().StringVar(&applyReleaseNameFlag, "release-name", "",
		"Release name (default: module name)")
	cmd.Flags().StringVar(&applyProviderFlag, "provider", "",
		"Provider to use (default: from config)")
	cmd.Flags().BoolVar(&applyDryRunFlag, "dry-run", false,
		"Server-side dry run (no changes made)")
	cmd.Flags().BoolVar(&applyWaitFlag, "wait", false,
		"Wait for resources to be ready")
	cmd.Flags().DurationVar(&applyTimeoutFlag, "timeout", 5*time.Minute,
		"Wait timeout")
	cmd.Flags().BoolVar(&applyCreateNSFlag, "create-namespace", false,
		"Create target namespace if it does not exist")
	cmd.Flags().StringVar(&applyKubeconfigFlag, "kubeconfig", "",
		"Path to kubeconfig file")
	cmd.Flags().StringVar(&applyContextFlag, "context", "",
		"Kubernetes context to use")

	return cmd
}

// runApply executes the apply command.
func runApply(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Determine module path
	modulePath := "."
	if len(args) > 0 {
		modulePath = args[0]
	}

	// Resolve Kubernetes configuration with local flags
	k8sConfig, err := resolveCommandKubernetes(
		applyKubeconfigFlag,
		applyContextFlag,
		applyNamespaceFlag,
		applyProviderFlag,
	)
	if err != nil {
		return &ExitError{Code: ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}

	kubeconfig := k8sConfig.Kubeconfig.Value
	kubeContext := k8sConfig.Context.Value
	namespace := k8sConfig.Namespace.Value
	provider := k8sConfig.Provider.Value

	// Log resolved k8s config at DEBUG level
	output.Debug("resolved kubernetes config",
		"kubeconfig", kubeconfig,
		"context", kubeContext,
		"namespace", namespace,
		"provider", provider,
	)

	// Get pre-loaded configuration
	opmConfig := GetOPMConfig()
	if opmConfig == nil {
		return &ExitError{Code: ExitGeneralError, Err: fmt.Errorf("configuration not loaded")}
	}

	// Build render options
	renderOpts := build.RenderOptions{
		ModulePath: modulePath,
		Values:     applyValuesFlags,
		Name:       applyReleaseNameFlag,
		Namespace:  namespace,
		Provider:   provider,
		Registry:   GetRegistry(),
	}

	if err := renderOpts.Validate(); err != nil {
		return &ExitError{Code: ExitGeneralError, Err: err}
	}

	// Create and execute pipeline
	pipeline := build.NewPipeline(opmConfig)

	output.Debug("rendering module",
		"module", modulePath,
		"namespace", namespace,
		"provider", provider,
	)

	result, err := pipeline.Render(ctx, renderOpts)
	if err != nil {
		printValidationError("render failed", err)
		return &ExitError{Code: ExitValidationError, Err: err, Printed: true}
	}

	// Check for render errors before touching the cluster
	if result.HasErrors() {
		printRenderErrors(result.Errors)
		return &ExitError{
			Code:    ExitValidationError,
			Err:     fmt.Errorf("%d render error(s)", len(result.Errors)),
			Printed: true,
		}
	}

	// Create scoped module logger
	modLog := output.ModuleLogger(result.Module.Name)

	// Print warnings
	if result.HasWarnings() {
		for _, w := range result.Warnings {
			modLog.Warn(w)
		}
	}

	if len(result.Resources) == 0 {
		modLog.Info("no resources to apply")
		return nil
	}

	// Create Kubernetes client
	k8sClient, err := kubernetes.NewClient(kubernetes.ClientOptions{
		Kubeconfig:  kubeconfig,
		Context:     kubeContext,
		APIWarnings: opmConfig.Config.Log.Kubernetes.APIWarnings,
	})
	if err != nil {
		modLog.Error("connecting to cluster", "error", err)
		return &ExitError{Code: ExitConnectivityError, Err: err, Printed: true}
	}

	// Create namespace if requested
	if applyCreateNSFlag && namespace != "" {
		created, nsErr := k8sClient.EnsureNamespace(ctx, namespace, applyDryRunFlag)
		if nsErr != nil {
			modLog.Error("ensuring namespace", "error", nsErr)
			return &ExitError{Code: exitCodeFromK8sError(nsErr), Err: nsErr, Printed: true}
		}
		if created {
			if applyDryRunFlag {
				modLog.Info(fmt.Sprintf("namespace %q would be created", namespace))
			} else {
				modLog.Info(fmt.Sprintf("namespace %q created", namespace))
			}
		}
	}

	// Apply resources
	if applyDryRunFlag {
		modLog.Info("dry run - no changes will be made")
	}
	modLog.Info(fmt.Sprintf("applying %d resources", len(result.Resources)))

	applyResult, err := kubernetes.Apply(ctx, k8sClient, result.Resources, result.Module, kubernetes.ApplyOptions{
		DryRun:  applyDryRunFlag,
		Wait:    applyWaitFlag,
		Timeout: applyTimeoutFlag,
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

	if applyDryRunFlag {
		modLog.Info(fmt.Sprintf("dry run complete: %d resources would be applied", applyResult.Applied))
	} else {
		modLog.Info(formatApplySummary(applyResult))
	}

	if len(applyResult.Errors) == 0 && !applyDryRunFlag {
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
