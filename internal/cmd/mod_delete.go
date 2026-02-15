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
	deleteNamespaceFlag      string
	deleteNameFlag           string
	deleteReleaseIDFlag      string
	deleteForceFlag          bool
	deleteDryRunFlag         bool
	deleteWaitFlag           bool
	deleteIgnoreNotFoundFlag bool
	deleteKubeconfigFlag     string
	deleteContextFlag        string
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

Exactly one of --name or --release-id is required to identify the module deployment.
The --namespace flag is always required.

Examples:
  # Delete module by name
  opm mod delete --name my-app -n production

  # Delete module by release ID
  opm mod delete --release-id a1b2c3d4-e5f6-7890-abcd-ef1234567890 -n production

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
		"Module name (mutually exclusive with --release-id)")
	cmd.Flags().StringVar(&deleteReleaseIDFlag, "release-id", "",
		"Release identity UUID (mutually exclusive with --name)")
	cmd.Flags().BoolVar(&deleteForceFlag, "force", false,
		"Skip confirmation prompt")
	cmd.Flags().BoolVar(&deleteDryRunFlag, "dry-run", false,
		"Preview without deleting")
	cmd.Flags().BoolVar(&deleteWaitFlag, "wait", false,
		"Wait for resources to be deleted")
	cmd.Flags().BoolVar(&deleteIgnoreNotFoundFlag, "ignore-not-found", false,
		"Exit 0 when no resources match the selector")
	cmd.Flags().StringVar(&deleteKubeconfigFlag, "kubeconfig", "",
		"Path to kubeconfig file")
	cmd.Flags().StringVar(&deleteContextFlag, "context", "",
		"Kubernetes context to use")

	// Namespace is always required
	_ = cmd.MarkFlagRequired("namespace")

	return cmd
}

// runDelete executes the delete command.
func runDelete(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Manual validation: require exactly one of --name or --release-id (mutually exclusive)
	if deleteNameFlag != "" && deleteReleaseIDFlag != "" {
		return &ExitError{
			Code: ExitGeneralError,
			Err:  fmt.Errorf("--name and --release-id are mutually exclusive"),
		}
	}
	if deleteNameFlag == "" && deleteReleaseIDFlag == "" {
		return &ExitError{
			Code: ExitGeneralError,
			Err:  fmt.Errorf("either --name or --release-id is required"),
		}
	}

	// Resolve flags with global fallback
	kubeconfig := resolveFlag(deleteKubeconfigFlag, GetKubeconfig())
	kubeContext := resolveFlag(deleteContextFlag, GetContext())

	// Create scoped module logger - prefer name, fall back to release-id
	logName := deleteNameFlag
	if logName == "" {
		logName = fmt.Sprintf("release:%s", deleteReleaseIDFlag[:8])
	}
	modLog := output.ModuleLogger(logName)

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

	// If dry-run, skip confirmation
	if deleteDryRunFlag {
		modLog.Info("dry run - no changes will be made")
	} else if !deleteForceFlag {
		// Prompt for confirmation
		if !confirmDelete(deleteNameFlag, deleteReleaseIDFlag, deleteNamespaceFlag) {
			modLog.Info("deletion canceled")
			return nil
		}
	}

	// Delete resources
	modLog.Info(fmt.Sprintf("deleting resources in namespace %q", deleteNamespaceFlag))

	deleteResult, err := kubernetes.Delete(ctx, k8sClient, kubernetes.DeleteOptions{
		ModuleName: deleteNameFlag,
		Namespace:  deleteNamespaceFlag,
		ReleaseID:  deleteReleaseIDFlag,
		DryRun:     deleteDryRunFlag,
		Wait:       deleteWaitFlag,
	})
	if err != nil {
		if deleteIgnoreNotFoundFlag && kubernetes.IsNoResourcesFound(err) {
			modLog.Info("no resources found (ignored)")
			return nil
		}
		modLog.Error("delete failed", "error", err)
		return &ExitError{Code: ExitGeneralError, Err: err, Printed: true}
	}

	// Report results
	if len(deleteResult.Errors) > 0 {
		modLog.Warn(fmt.Sprintf("%d resource(s) had errors", len(deleteResult.Errors)))
		for _, e := range deleteResult.Errors {
			modLog.Error(e.Error())
		}
	}

	if deleteDryRunFlag {
		modLog.Info(fmt.Sprintf("dry run complete: %d resources would be deleted", deleteResult.Deleted))
	} else {
		modLog.Info("all resources have been deleted")
		output.Println(output.FormatCheckmark("Module deleted"))
	}

	if len(deleteResult.Errors) > 0 {
		return &ExitError{
			Code:    ExitGeneralError,
			Err:     fmt.Errorf("%d resource(s) failed to delete", len(deleteResult.Errors)),
			Printed: true,
		}
	}

	return nil
}

// confirmDelete prompts the user for confirmation.
func confirmDelete(name, releaseID, namespace string) bool {
	var prompt string
	if name != "" {
		prompt = fmt.Sprintf("Delete all resources for module %q in namespace %q? [y/N]: ", name, namespace)
	} else {
		prompt = fmt.Sprintf("Delete all resources for release-id %q in namespace %q? [y/N]: ", releaseID, namespace)
	}

	output.Prompt(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return answer == "y" || answer == "yes"
	}
	return false
}
