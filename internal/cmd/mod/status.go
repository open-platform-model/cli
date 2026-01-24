package mod

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/opmodel/cli/internal/cmd"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// statusOptions holds the flags for the status command.
type statusOptions struct {
	name      string
	namespace string
	context   string
	output    string
}

// NewStatusCmd creates the mod status command.
func NewStatusCmd() *cobra.Command {
	opts := &statusOptions{}

	c := &cobra.Command{
		Use:   "status <module-name>",
		Short: "Show module deployment status",
		Long:  `Shows the status of a deployed module's resources.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts.name = args[0]
			return runStatus(c.Context(), opts)
		},
	}

	c.Flags().StringVarP(&opts.namespace, "namespace", "n", "", "Namespace to check")
	c.Flags().StringVar(&opts.context, "context", "", "Kubernetes context to use")
	c.Flags().StringVarP(&opts.output, "output", "o", "table", "Output format (table, yaml, json)")

	return c
}

// runStatus shows the module status.
func runStatus(ctx context.Context, opts *statusOptions) error {
	// Validate output format
	if opts.output != "table" && opts.output != "yaml" && opts.output != "json" {
		return fmt.Errorf("%w: invalid output format %q, use table, yaml, or json", cmd.ErrValidation, opts.output)
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

	// Get module status
	status, err := k8sClient.GetModuleStatus(ctx, opts.name, namespace, "")
	if err != nil {
		if errors.Is(err, kubernetes.ErrPermissionDenied) {
			return fmt.Errorf("%w: %v", cmd.ErrPermission, err)
		}
		return fmt.Errorf("getting status: %w", err)
	}

	// Output based on format
	switch opts.output {
	case "table":
		table := output.NewStatusTable()
		fmt.Print(table.RenderModuleStatus(status))
	case "yaml":
		data, err := yaml.Marshal(status)
		if err != nil {
			return fmt.Errorf("marshaling status: %w", err)
		}
		fmt.Print(string(data))
	case "json":
		data, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling status: %w", err)
		}
		fmt.Println(string(data))
	}

	return nil
}
