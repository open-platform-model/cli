package mod

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/charmbracelet/huh/spinner"
	"github.com/opmodel/cli/internal/cmd"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/internal/render"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// applyOptions holds the flags for the apply command.
type applyOptions struct {
	dir       string
	values    []string
	namespace string
	context   string
	dryRun    bool
	showDiff  bool
	wait      bool
	timeout   time.Duration
}

// NewApplyCmd creates the mod apply command.
func NewApplyCmd() *cobra.Command {
	opts := &applyOptions{}

	c := &cobra.Command{
		Use:   "apply",
		Short: "Apply module to cluster",
		Long:  `Renders the module and applies it to a Kubernetes cluster using server-side apply.`,
		RunE: func(c *cobra.Command, args []string) error {
			return runApply(c.Context(), opts)
		},
	}

	c.Flags().StringVar(&opts.dir, "dir", ".", "Module directory")
	c.Flags().StringSliceVarP(&opts.values, "values", "f", nil, "Values files to unify")
	c.Flags().StringVarP(&opts.namespace, "namespace", "n", "", "Namespace to deploy to")
	c.Flags().StringVar(&opts.context, "context", "", "Kubernetes context to use")
	c.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Server-side dry run without changes")
	c.Flags().BoolVar(&opts.showDiff, "diff", false, "Show diff before applying")
	c.Flags().BoolVarP(&opts.wait, "wait", "w", false, "Wait for resources to be ready")
	c.Flags().DurationVar(&opts.timeout, "timeout", 5*time.Minute, "Timeout for the operation")

	return c
}

// runApply applies the module to the cluster.
func runApply(ctx context.Context, opts *applyOptions) error {
	// Create render pipeline
	pipeline := render.NewPipeline(&render.Options{
		Dir:         opts.dir,
		ValuesFiles: opts.values,
		Verbose:     false, // Apply command doesn't show render verbosity by default
	})

	// Execute render pipeline
	result, err := pipeline.Render(ctx)
	if err != nil {
		return fmt.Errorf("%w: render failed: %v", cmd.ErrValidation, err)
	}

	if len(result.Manifests) == 0 {
		return fmt.Errorf("%w: no manifests generated", cmd.ErrValidation)
	}

	// Convert manifests to unstructured objects
	objects := make([]*unstructured.Unstructured, len(result.Manifests))
	for i, m := range result.Manifests {
		objects[i] = &unstructured.Unstructured{Object: m.Object}
	}

	// Create Kubernetes client
	k8sClient, err := kubernetes.NewClient(kubernetes.ClientOptions{
		Context:   opts.context,
		Namespace: opts.namespace,
	})
	if err != nil {
		if errors.Is(err, kubernetes.ErrNoKubeconfig) {
			return fmt.Errorf("%w: %v", cmd.ErrConnectivity, err)
		}
		return fmt.Errorf("creating kubernetes client: %w", err)
	}

	// Check connectivity
	if err := k8sClient.CheckConnection(ctx); err != nil {
		return fmt.Errorf("%w: %v", cmd.ErrConnectivity, err)
	}

	// Determine namespace
	namespace := opts.namespace
	if namespace == "" {
		namespace = k8sClient.DefaultNamespace
	}

	// Build labels
	labels := kubernetes.ModuleLabels(
		result.ModuleName,
		namespace,
		result.ModuleVersion,
		"", // component set per-resource
	)

	// Apply options
	applyOpts := kubernetes.ApplyOptions{
		DryRun:    opts.dryRun,
		Namespace: namespace,
		Labels:    labels,
		Wait:      opts.wait,
		Timeout:   opts.timeout,
	}

	// Show diff before applying if requested
	if opts.showDiff {
		diffOpts := kubernetes.DiffOptions{
			Namespace:       namespace,
			UseColor:        !output.IsNoColor(),
			ModuleName:      result.ModuleName,
			ModuleNamespace: namespace,
		}

		diffResult, diffErr := k8sClient.Diff(ctx, objects, diffOpts)
		if diffErr != nil {
			return fmt.Errorf("computing diff: %w", diffErr)
		}

		if diffResult.IsEmpty() {
			fmt.Println("No changes to apply. Module is in sync with cluster.")
			return nil
		}

		// Get styles and render diff
		styles := output.GetStyles()
		modified := make([]output.ModifiedItem, len(diffResult.Modified))
		for i, m := range diffResult.Modified {
			modified[i] = output.ModifiedItem{
				Name: m.Name,
				Diff: m.Diff,
			}
		}
		rendered := output.RenderDiff(diffResult.Added, diffResult.Removed, modified, styles)
		fmt.Print(rendered)
		fmt.Println()
	}

	// Show what we're doing
	if opts.dryRun {
		fmt.Printf("Dry-run: would apply %d resources for module %s to namespace %s\n",
			len(objects), result.ModuleName, namespace)
	} else {
		fmt.Printf("Applying %d resources for module %s to namespace %s...\n",
			len(objects), result.ModuleName, namespace)
	}

	// Apply with spinner if waiting
	var applyResult *kubernetes.ApplyResult
	if opts.wait && !opts.dryRun {
		err = spinner.New().
			Title("Applying resources...").
			Action(func() {
				applyResult, err = k8sClient.Apply(ctx, objects, applyOpts)
			}).
			Run()
	} else {
		applyResult, err = k8sClient.Apply(ctx, objects, applyOpts)
	}

	if err != nil {
		if errors.Is(err, kubernetes.ErrPermissionDenied) {
			return fmt.Errorf("%w: %v", cmd.ErrPermission, err)
		}
		return fmt.Errorf("applying resources: %w", err)
	}

	// Report results
	if opts.dryRun {
		fmt.Printf("Dry-run complete: %d resources would be created/updated\n", applyResult.Created)
	} else {
		fmt.Printf("Applied %d resources\n", applyResult.Created)
	}

	if len(applyResult.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, e := range applyResult.Errors {
			fmt.Printf("  - %v\n", e)
		}
	}

	return nil
}
