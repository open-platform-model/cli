package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
)

// Delete command flags.
var (
	deleteNamespaceFlag  string
	deleteNameFlag       string
	deleteForceFlag      bool
	deleteDryRunFlag     bool
	deleteWaitFlag       bool
	deleteKubeconfigFlag string
	deleteContextFlag    string
)

// NewModDeleteCmd creates the mod delete command.
func NewModDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete module resources from cluster",
		Long: `Delete all resources belonging to an OPM module from a Kubernetes cluster.

Resources are discovered via OPM labels, so the original module source is
not required. Resources are deleted in reverse weight order (webhooks first,
CRDs last).

Both --name and --namespace are required to identify the module deployment.

Examples:
  # Delete module from cluster
  opm mod delete --name my-app -n production

  # Preview what would be deleted
  opm mod delete --name my-app -n production --dry-run

  # Skip confirmation prompt
  opm mod delete --name my-app -n production --force`,
		RunE: runDelete,
	}

	// Add flags
	cmd.Flags().StringVarP(&deleteNamespaceFlag, "namespace", "n", "",
		"Target namespace (required)")
	cmd.Flags().StringVar(&deleteNameFlag, "name", "",
		"Module name (required)")
	cmd.Flags().BoolVar(&deleteForceFlag, "force", false,
		"Skip confirmation prompt")
	cmd.Flags().BoolVar(&deleteDryRunFlag, "dry-run", false,
		"Preview without deleting")
	cmd.Flags().BoolVar(&deleteWaitFlag, "wait", false,
		"Wait for resources to be deleted")
	cmd.Flags().StringVar(&deleteKubeconfigFlag, "kubeconfig", "",
		"Path to kubeconfig file")
	cmd.Flags().StringVar(&deleteContextFlag, "context", "",
		"Kubernetes context to use")

	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("namespace")

	return cmd
}

// runDelete executes the delete command.
func runDelete(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Resolve flags with global fallback
	kubeconfig := resolveFlag(deleteKubeconfigFlag, GetKubeconfig())
	kubeContext := resolveFlag(deleteContextFlag, GetContext())

	// Create Kubernetes client
	k8sClient, err := kubernetes.NewClient(kubernetes.ClientOptions{
		Kubeconfig: kubeconfig,
		Context:    kubeContext,
	})
	if err != nil {
		output.Error("connecting to cluster", "error", err)
		return &ExitError{Code: ExitConnectivityError, Err: err}
	}

	// If dry-run, skip confirmation
	if deleteDryRunFlag {
		output.Info("dry run - no changes will be made")
	} else if !deleteForceFlag {
		// Prompt for confirmation
		if !confirmDelete(deleteNameFlag, deleteNamespaceFlag) {
			output.Info("deletion cancelled")
			return nil
		}
	}

	// Delete resources
	output.Info(fmt.Sprintf("deleting resources for module %q in namespace %q", deleteNameFlag, deleteNamespaceFlag))

	deleteResult, err := kubernetes.Delete(ctx, k8sClient, kubernetes.DeleteOptions{
		ModuleName: deleteNameFlag,
		Namespace:  deleteNamespaceFlag,
		DryRun:     deleteDryRunFlag,
		Wait:       deleteWaitFlag,
	})
	if err != nil {
		output.Error("delete failed", "error", err)
		return &ExitError{Code: ExitGeneralError, Err: err}
	}

	// Report results
	if len(deleteResult.Errors) > 0 {
		output.Warn(fmt.Sprintf("%d resource(s) had errors", len(deleteResult.Errors)))
		for _, e := range deleteResult.Errors {
			output.Error(e.Error())
		}
	}

	if deleteDryRunFlag {
		output.Info(fmt.Sprintf("dry run complete: %d resources would be deleted", deleteResult.Deleted))
	} else {
		output.Info(fmt.Sprintf("delete complete: %d resources deleted", deleteResult.Deleted))
	}

	if len(deleteResult.Errors) > 0 {
		return &ExitError{
			Code: ExitGeneralError,
			Err:  fmt.Errorf("%d resource(s) failed to delete", len(deleteResult.Errors)),
		}
	}

	return nil
}

// confirmDelete prompts the user for confirmation.
func confirmDelete(name, namespace string) bool {
	fmt.Fprintf(os.Stderr, "Delete all resources for module %q in namespace %q? [y/N]: ", name, namespace)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return answer == "y" || answer == "yes"
	}
	return false
}
