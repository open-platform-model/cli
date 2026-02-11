package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/build"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
)

// Apply command flags.
var (
	applyValuesFlags    []string
	applyNamespaceFlag  string
	applyNameFlag       string
	applyProviderFlag   string
	applyDryRunFlag     bool
	applyWaitFlag       bool
	applyTimeoutFlag    time.Duration
	applyKubeconfigFlag string
	applyContextFlag    string
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
	cmd.Flags().StringVar(&applyNameFlag, "name", "",
		"Release name (default: module name)")
	cmd.Flags().StringVar(&applyProviderFlag, "provider", "",
		"Provider to use (default: from config)")
	cmd.Flags().BoolVar(&applyDryRunFlag, "dry-run", false,
		"Server-side dry run (no changes made)")
	cmd.Flags().BoolVar(&applyWaitFlag, "wait", false,
		"Wait for resources to be ready")
	cmd.Flags().DurationVar(&applyTimeoutFlag, "timeout", 5*time.Minute,
		"Wait timeout")
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

	// Resolve flags with global fallback
	kubeconfig := resolveFlag(applyKubeconfigFlag, GetKubeconfig())
	kubeContext := resolveFlag(applyContextFlag, GetContext())
	namespace := resolveFlag(applyNamespaceFlag, GetNamespace())
	provider := resolveFlag(applyProviderFlag, GetProvider())

	// Get pre-loaded configuration
	opmConfig := GetOPMConfig()
	if opmConfig == nil {
		return &ExitError{Code: ExitGeneralError, Err: fmt.Errorf("configuration not loaded")}
	}

	// Build render options
	renderOpts := build.RenderOptions{
		ModulePath: modulePath,
		Values:     applyValuesFlags,
		Name:       applyNameFlag,
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
		output.Error("render failed", "error", err)
		return &ExitError{Code: ExitValidationError, Err: err}
	}

	// Check for render errors before touching the cluster
	if result.HasErrors() {
		printRenderErrors(result.Errors)
		return &ExitError{
			Code: ExitValidationError,
			Err:  fmt.Errorf("%d render error(s)", len(result.Errors)),
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
		return &ExitError{Code: ExitConnectivityError, Err: err}
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
		return &ExitError{Code: ExitGeneralError, Err: err}
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
		modLog.Info(fmt.Sprintf("applied %d resources successfully", applyResult.Applied))
	}

	if len(applyResult.Errors) == 0 && !applyDryRunFlag {
		output.Println(output.FormatCheckmark("Module applied"))
	}

	if len(applyResult.Errors) > 0 {
		return &ExitError{
			Code: ExitGeneralError,
			Err:  fmt.Errorf("%d resource(s) failed to apply", len(applyResult.Errors)),
		}
	}

	return nil
}

// resolveFlag returns the local flag if set, otherwise falls back to resolved global value.
func resolveFlag(localFlag string, resolvedGlobal string) string {
	if localFlag != "" {
		return localFlag
	}
	return resolvedGlobal
}
