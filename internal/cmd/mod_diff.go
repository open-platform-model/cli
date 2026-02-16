package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/build"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
)

// Diff command flags.
var (
	diffValuesFlags     []string
	diffNamespaceFlag   string
	diffReleaseNameFlag string
	diffKubeconfigFlag  string
	diffContextFlag     string
)

// NewModDiffCmd creates the mod diff command.
func NewModDiffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff [path]",
		Short: "Show differences with cluster",
		Long: `Show differences between local module and live cluster state.

This command renders the module locally and compares each resource against
its live state on the cluster using semantic YAML diff (via dyff).

Resources are categorized as:
  - modified: exists locally and on cluster with differences
  - added:    exists locally but not on cluster
  - orphaned: exists on cluster (by OPM labels) but not in local render

Arguments:
  path    Path to module directory (default: current directory)

Examples:
  # Diff module in current directory
  opm mod diff

  # Diff with custom values
  opm mod diff -f prod-values.cue

  # Diff using specific kubeconfig
  opm mod diff --kubeconfig ~/.kube/staging --context staging-cluster`,
		Args: cobra.MaximumNArgs(1),
		RunE: runDiff,
	}

	// Add flags
	cmd.Flags().StringArrayVarP(&diffValuesFlags, "values", "f", nil,
		"Additional values files (can be repeated)")
	cmd.Flags().StringVarP(&diffNamespaceFlag, "namespace", "n", "",
		"Target namespace")
	cmd.Flags().StringVar(&diffReleaseNameFlag, "release-name", "",
		"Release name (default: module name)")
	cmd.Flags().StringVar(&diffKubeconfigFlag, "kubeconfig", "",
		"Path to kubeconfig file")
	cmd.Flags().StringVar(&diffContextFlag, "context", "",
		"Kubernetes context to use")

	return cmd
}

// runDiff executes the diff command.
func runDiff(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Determine module path
	modulePath := "."
	if len(args) > 0 {
		modulePath = args[0]
	}

	// Resolve flags with global fallback
	kubeconfig := resolveFlag(diffKubeconfigFlag, GetKubeconfig())
	kubeContext := resolveFlag(diffContextFlag, GetContext())
	namespace := resolveFlag(diffNamespaceFlag, GetNamespace())

	// Get pre-loaded configuration
	opmConfig := GetOPMConfig()
	if opmConfig == nil {
		return &ExitError{Code: ExitGeneralError, Err: fmt.Errorf("configuration not loaded")}
	}

	// Build render options
	renderOpts := build.RenderOptions{
		ModulePath: modulePath,
		Values:     diffValuesFlags,
		Name:       diffReleaseNameFlag,
		Namespace:  namespace,
		Registry:   GetRegistry(),
	}

	if err := renderOpts.Validate(); err != nil {
		return &ExitError{Code: ExitGeneralError, Err: err}
	}

	// Create and execute pipeline
	pipeline := build.NewPipeline(opmConfig)

	output.Debug("rendering module for diff",
		"module", modulePath,
		"namespace", namespace,
	)

	result, err := pipeline.Render(ctx, renderOpts)
	if err != nil {
		printValidationError("render failed", err)
		return &ExitError{Code: ExitValidationError, Err: err, Printed: true}
	}

	// Create scoped module logger
	modLog := output.ModuleLogger(result.Module.Name)

	// Print warnings
	if result.HasWarnings() {
		for _, w := range result.Warnings {
			modLog.Warn(w)
		}
	}

	if len(result.Resources) == 0 && !result.HasErrors() {
		modLog.Info("no resources to diff")
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

	// Create comparer
	comparer := kubernetes.NewComparer()

	// Run diff — handle partial render results
	var diffResult *kubernetes.DiffResult
	if result.HasErrors() {
		diffResult, err = kubernetes.DiffPartial(ctx, k8sClient, result.Resources, result.Errors, result.Module, comparer)
	} else {
		diffResult, err = kubernetes.Diff(ctx, k8sClient, result.Resources, result.Module, comparer)
	}
	if err != nil {
		modLog.Error("diff failed", "error", err)
		return &ExitError{Code: ExitGeneralError, Err: err, Printed: true}
	}

	// Print warnings from diff
	for _, w := range diffResult.Warnings {
		modLog.Warn(w)
	}

	// Print summary
	if diffResult.IsEmpty() {
		output.Println("No differences found")
		return nil
	}

	output.Println(diffResult.SummaryLine())
	output.Println("")

	// Print detailed diff output
	for _, rd := range diffResult.Resources {
		switch rd.State {
		case kubernetes.ResourceModified:
			if rd.Namespace != "" {
				output.Println(fmt.Sprintf("--- %s/%s (%s) [modified]", rd.Kind, rd.Name, rd.Namespace))
			} else {
				output.Println(fmt.Sprintf("--- %s/%s [modified]", rd.Kind, rd.Name))
			}
			output.Println(rd.Diff)

		case kubernetes.ResourceAdded:
			if rd.Namespace != "" {
				output.Println(fmt.Sprintf("+++ %s/%s (%s) [new resource]", rd.Kind, rd.Name, rd.Namespace))
			} else {
				output.Println(fmt.Sprintf("+++ %s/%s [new resource]", rd.Kind, rd.Name))
			}

		case kubernetes.ResourceOrphaned:
			if rd.Namespace != "" {
				output.Println(fmt.Sprintf("~~~ %s/%s (%s) [orphaned - will be removed on next apply]", rd.Kind, rd.Name, rd.Namespace))
			} else {
				output.Println(fmt.Sprintf("~~~ %s/%s [orphaned - will be removed on next apply]", rd.Kind, rd.Name))
			}
		}
	}

	return nil
}
