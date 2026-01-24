package mod

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/huh/spinner"
	"github.com/opmodel/cli/internal/cmd"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// deleteOptions holds the flags for the delete command.
type deleteOptions struct {
	name      string
	namespace string
	context   string
	dryRun    bool
	force     bool
	timeout   time.Duration
}

// NewDeleteCmd creates the mod delete command.
func NewDeleteCmd() *cobra.Command {
	opts := &deleteOptions{}

	c := &cobra.Command{
		Use:   "delete <module-name>",
		Short: "Delete a deployed module",
		Long:  `Deletes all resources belonging to a deployed module.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts.name = args[0]
			return runDelete(c.Context(), opts)
		},
	}

	c.Flags().StringVarP(&opts.namespace, "namespace", "n", "", "Namespace to delete from")
	c.Flags().StringVar(&opts.context, "context", "", "Kubernetes context to use")
	c.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Show what would be deleted")
	c.Flags().BoolVar(&opts.force, "force", false, "Skip confirmation and remove finalizers")
	c.Flags().DurationVar(&opts.timeout, "timeout", 5*time.Minute, "Timeout for the operation")

	return c
}

// runDelete deletes the module.
func runDelete(ctx context.Context, opts *deleteOptions) error {
	// Check for TTY and require --force in non-TTY environments
	if !term.IsTerminal(int(os.Stdin.Fd())) && !opts.force && !opts.dryRun {
		return fmt.Errorf("--force required in non-interactive mode")
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

	// Discover resources first
	resources, err := k8sClient.DiscoverModuleResources(ctx, opts.name, namespace)
	if err != nil {
		if errors.Is(err, kubernetes.ErrPermissionDenied) {
			return fmt.Errorf("%w: %v", cmd.ErrPermission, err)
		}
		return fmt.Errorf("discovering resources: %w", err)
	}

	if len(resources) == 0 {
		fmt.Printf("No resources found for module %s in namespace %s\n", opts.name, namespace)
		return nil
	}

	// Show what will be deleted
	if opts.dryRun {
		fmt.Printf("Dry-run: would delete %d resources for module %s in namespace %s:\n",
			len(resources), opts.name, namespace)
		for _, r := range resources {
			ns := r.GetNamespace()
			if ns == "" {
				ns = "(cluster-scoped)"
			}
			fmt.Printf("  - %s/%s [%s]\n", r.GetKind(), r.GetName(), ns)
		}
		return nil
	}

	// Confirm deletion if not force
	if !opts.force {
		fmt.Printf("About to delete %d resources for module %s in namespace %s\n",
			len(resources), opts.name, namespace)
		fmt.Print("Continue? [y/N]: ")

		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Aborted")
			return nil
		}
	}

	// Delete options
	deleteOpts := kubernetes.DeleteOptions{
		DryRun:  opts.dryRun,
		Force:   opts.force,
		Timeout: opts.timeout,
	}

	fmt.Printf("Deleting %d resources for module %s...\n", len(resources), opts.name)

	// Delete with spinner
	var result *kubernetes.DeleteResult
	err = spinner.New().
		Title("Deleting resources...").
		Action(func() {
			result, err = k8sClient.Delete(ctx, resources, deleteOpts)
		}).
		Run()

	if err != nil {
		if errors.Is(err, kubernetes.ErrPermissionDenied) {
			return fmt.Errorf("%w: %v", cmd.ErrPermission, err)
		}
		return fmt.Errorf("deleting resources: %w", err)
	}

	// Report results
	fmt.Printf("Deleted %d resources", result.Deleted)
	if result.NotFound > 0 {
		fmt.Printf(" (%d already gone)", result.NotFound)
	}
	fmt.Println()

	if len(result.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, e := range result.Errors {
			fmt.Printf("  - %v\n", e)
		}
	}

	return nil
}
