package mod

import (
	"context"
	"errors"
	"fmt"

	"github.com/opmodel/cli/internal/cmd"
	opmcue "github.com/opmodel/cli/internal/cue"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
	"github.com/spf13/cobra"
)

// diffOptions holds the flags for the diff command.
type diffOptions struct {
	dir       string
	values    []string
	namespace string
	context   string
	noColor   bool
}

// NewDiffCmd creates the mod diff command.
func NewDiffCmd() *cobra.Command {
	opts := &diffOptions{}

	c := &cobra.Command{
		Use:   "diff",
		Short: "Show differences between local module and cluster state",
		Long: `Compares the rendered module manifests against the live cluster state.

Shows what would change if you ran 'opm mod apply':
  - Added resources (exist locally but not in cluster)
  - Removed resources (exist in cluster but not locally)
  - Modified resources (exist in both but have differences)

Exit codes:
  0 - No differences found
  1 - Differences exist or an error occurred
  2 - Validation error (invalid module)
  3 - Cannot connect to cluster`,
		RunE: func(c *cobra.Command, args []string) error {
			return runDiff(c.Context(), opts)
		},
	}

	c.Flags().StringVar(&opts.dir, "dir", ".", "Module directory")
	c.Flags().StringSliceVarP(&opts.values, "values", "f", nil, "Values files to unify")
	c.Flags().StringVarP(&opts.namespace, "namespace", "n", "", "Namespace to compare against")
	c.Flags().StringVar(&opts.context, "context", "", "Kubernetes context to use")
	c.Flags().BoolVar(&opts.noColor, "no-color", false, "Disable colored output")

	return c
}

// runDiff executes the diff logic.
func runDiff(ctx context.Context, opts *diffOptions) error {
	// Load the module
	loader := opmcue.NewLoader()
	module, err := loader.LoadModule(ctx, opts.dir, opts.values)
	if err != nil {
		if errors.Is(err, opmcue.ErrModuleNotFound) {
			return fmt.Errorf("%w: %v", cmd.ErrNotFound, err)
		}
		if errors.Is(err, opmcue.ErrInvalidModule) {
			return fmt.Errorf("%w: %v", cmd.ErrValidation, err)
		}
		return fmt.Errorf("loading module: %w", err)
	}

	// Render manifests
	renderer := opmcue.NewRenderer()
	manifestSet, err := renderer.RenderModule(ctx, module)
	if err != nil {
		if errors.Is(err, opmcue.ErrNoManifests) {
			return fmt.Errorf("%w: no manifests found in module", cmd.ErrValidation)
		}
		if errors.Is(err, opmcue.ErrRenderFailed) {
			return fmt.Errorf("%w: %v", cmd.ErrValidation, err)
		}
		return fmt.Errorf("rendering manifests: %w", err)
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

	// Build labels and inject them into manifests for consistent comparison
	labels := kubernetes.ModuleLabels(
		module.Metadata.Name,
		namespace,
		module.Metadata.Version,
		"", // component set per-resource
	)

	// Get objects and inject labels
	objects := manifestSet.Objects()
	for _, obj := range objects {
		kubernetes.InjectLabels(obj.Object, labels)

		// Set namespace if not set
		if obj.GetNamespace() == "" {
			if isNamespacedKind(obj.GetKind()) {
				obj.SetNamespace(namespace)
			}
		}
	}

	// Print what we're comparing
	fmt.Printf("Comparing module '%s' against namespace '%s'...\n\n", module.Metadata.Name, namespace)

	// Compute diff
	useColor := !opts.noColor && !output.IsNoColor()
	diffOpts := kubernetes.DiffOptions{
		Namespace:       namespace,
		UseColor:        useColor,
		ModuleName:      module.Metadata.Name,
		ModuleNamespace: namespace,
	}

	diffResult, err := k8sClient.Diff(ctx, objects, diffOpts)
	if err != nil {
		if errors.Is(err, kubernetes.ErrPermissionDenied) {
			return fmt.Errorf("%w: %v", cmd.ErrPermission, err)
		}
		return fmt.Errorf("computing diff: %w", err)
	}

	// Render output
	if diffResult.IsEmpty() {
		fmt.Println("No changes detected. Module is in sync with cluster.")
		return nil
	}

	// Get styles
	styles := output.GetStyles()
	if opts.noColor || output.IsNoColor() {
		styles = output.NoColorStyles()
	}

	// Convert to output format
	modified := make([]output.ModifiedItem, len(diffResult.Modified))
	for i, m := range diffResult.Modified {
		modified[i] = output.ModifiedItem{
			Name: m.Name,
			Diff: m.Diff,
		}
	}

	// Render and print
	rendered := output.RenderDiff(diffResult.Added, diffResult.Removed, modified, styles)
	fmt.Print(rendered)

	// Return error to trigger exit code 1 when there are differences
	// This follows the convention of diff(1)
	return cmd.NewExitError(errors.New("differences found"), cmd.ExitGeneralError)
}

// isNamespacedKind checks if a kind is namespace-scoped based on known cluster-scoped types.
func isNamespacedKind(kind string) bool {
	clusterScoped := map[string]bool{
		"Namespace":                        true,
		"Node":                             true,
		"PersistentVolume":                 true,
		"ClusterRole":                      true,
		"ClusterRoleBinding":               true,
		"CustomResourceDefinition":         true,
		"APIService":                       true,
		"MutatingWebhookConfiguration":     true,
		"ValidatingWebhookConfiguration":   true,
		"PriorityClass":                    true,
		"StorageClass":                     true,
		"VolumeAttachment":                 true,
		"CSIDriver":                        true,
		"CSINode":                          true,
		"RuntimeClass":                     true,
		"PodSecurityPolicy":                true,
		"CertificateSigningRequest":        true,
		"IngressClass":                     true,
		"ValidatingAdmissionPolicy":        true,
		"ValidatingAdmissionPolicyBinding": true,
	}

	return !clusterScoped[kind]
}
