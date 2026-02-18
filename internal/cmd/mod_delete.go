package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
)

// NewModDeleteCmd creates the mod delete command.
func NewModDeleteCmd() *cobra.Command {
	var rsf cmdutil.ReleaseSelectorFlags
	var kf cmdutil.K8sFlags

	// Delete-specific flags (local to this command)
	var (
		forceFlag          bool
		dryRunFlag         bool
		waitFlag           bool
		ignoreNotFoundFlag bool
	)

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete release resources from cluster",
		Long: `Delete all resources belonging to an OPM release from a Kubernetes cluster.

Resources are discovered via OPM labels, so the original module source is
not required. Resources are deleted in reverse weight order (webhooks first,
CRDs last).

Exactly one of --release-name or --release-id is required to identify the release deployment.
The --namespace flag is always required.

Examples:
  # Delete release by name
  opm mod delete --release-name my-app -n production

  # Delete release by release ID
  opm mod delete --release-id a1b2c3d4-e5f6-7890-abcd-ef1234567890 -n production

  # Preview what would be deleted
  opm mod delete --release-name my-app -n production --dry-run

  # Skip confirmation prompt
  opm mod delete --release-name my-app -n production --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(cmd, args, &rsf, &kf, forceFlag, dryRunFlag, waitFlag, ignoreNotFoundFlag)
		},
	}

	rsf.AddTo(cmd)
	kf.AddTo(cmd)

	// Delete-specific flags
	cmd.Flags().BoolVar(&forceFlag, "force", false,
		"Skip confirmation prompt")
	cmd.Flags().BoolVar(&dryRunFlag, "dry-run", false,
		"Preview without deleting")
	cmd.Flags().BoolVar(&waitFlag, "wait", false,
		"Wait for resources to be deleted")
	cmd.Flags().BoolVar(&ignoreNotFoundFlag, "ignore-not-found", false,
		"Exit 0 when no resources match the selector")

	return cmd
}

// runDelete executes the delete command.
func runDelete(_ *cobra.Command, _ []string, rsf *cmdutil.ReleaseSelectorFlags, kf *cmdutil.K8sFlags, force, dryRun, _ /* wait */, ignoreNotFound bool) error { //nolint:gocyclo // orchestration function; complexity is inherent
	ctx := context.Background()

	// Validate release selector flags
	if err := rsf.Validate(); err != nil {
		return &ExitError{Code: ExitGeneralError, Err: err}
	}

	// Resolve Kubernetes configuration with local flags
	k8sConfig, err := cmdutil.ResolveKubernetes(
		GetOPMConfig(),
		kf.Kubeconfig,
		kf.Context,
		rsf.Namespace,
		"", // no provider flag for delete
	)
	if err != nil {
		return &ExitError{Code: ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}

	namespace := k8sConfig.Namespace.Value

	// Log resolved k8s config at DEBUG level
	output.Debug("resolved kubernetes config",
		"kubeconfig", k8sConfig.Kubeconfig.Value,
		"context", k8sConfig.Context.Value,
		"namespace", namespace,
	)

	// Create scoped module logger using shared LogName helper
	modLog := output.ModuleLogger(rsf.LogName())

	opmConfig := GetOPMConfig()

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

	// If dry-run, skip confirmation
	if dryRun {
		modLog.Info("dry run - no changes will be made")
	} else if !force {
		// Prompt for confirmation
		if !confirmDelete(rsf.ReleaseName, rsf.ReleaseID, namespace) {
			modLog.Info("deletion canceled")
			return nil
		}
	}

	// Delete resources
	modLog.Info(fmt.Sprintf("deleting resources in namespace %q", namespace))

	// Attempt inventory-first discovery.
	// Works with either --release-name or --release-id (or both).
	// Falls back to label-scan when no inventory exists (backward compatibility).
	deleteOpts := kubernetes.DeleteOptions{
		ReleaseName: rsf.ReleaseName,
		Namespace:   namespace,
		ReleaseID:   rsf.ReleaseID,
		DryRun:      dryRun,
	}

	var inv *inventory.InventorySecret
	var invErr error

	switch {
	case rsf.ReleaseID != "":
		// Primary path: direct GET by name+UUID, with UUID label fallback.
		relName := rsf.ReleaseName
		if relName == "" {
			relName = rsf.ReleaseID // best-effort name for Secret name construction
		}
		inv, invErr = inventory.GetInventory(ctx, k8sClient, relName, namespace, rsf.ReleaseID)
		if invErr != nil {
			modLog.Debug("could not read inventory by release-id, using label-scan", "error", invErr)
		}
	case rsf.ReleaseName != "":
		// Name-only path: label scan by module-release.opmodel.dev/name.
		inv, invErr = inventory.FindInventoryByReleaseName(ctx, k8sClient, rsf.ReleaseName, namespace)
		if invErr != nil {
			modLog.Debug("could not read inventory by release-name, using label-scan", "error", invErr)
		}
	}

	if inv != nil {
		liveResources, _, invDiscoverErr := inventory.DiscoverResourcesFromInventory(ctx, k8sClient, inv)
		if invDiscoverErr != nil {
			modLog.Debug("inventory discovery failed, falling back to label-scan", "error", invDiscoverErr)
		} else {
			deleteOpts.InventoryLive = liveResources
			deleteOpts.InventorySecretName = inventory.SecretName(inv.Metadata.ReleaseName, inv.Metadata.ReleaseID)
			deleteOpts.InventorySecretNamespace = namespace
		}
	}

	deleteResult, err := kubernetes.Delete(ctx, k8sClient, deleteOpts)
	if err != nil {
		if ignoreNotFound && kubernetes.IsNoResourcesFound(err) {
			modLog.Info("no resources found (ignored)")
			return nil
		}
		modLog.Error("delete failed", "error", err)
		return &ExitError{Code: exitCodeFromK8sError(err), Err: err, Printed: true}
	}

	// Report results
	if len(deleteResult.Errors) > 0 {
		modLog.Warn(fmt.Sprintf("%d resource(s) had errors", len(deleteResult.Errors)))
		for _, e := range deleteResult.Errors {
			modLog.Error(e.Error())
		}
	}

	if dryRun {
		modLog.Info(fmt.Sprintf("dry run complete: %d resources would be deleted", deleteResult.Deleted))
	} else {
		modLog.Info("all resources have been deleted")
		output.Println(output.FormatCheckmark("Release deleted"))
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
func confirmDelete(releaseName, releaseID, namespace string) bool {
	var prompt string
	if releaseName != "" {
		prompt = fmt.Sprintf("Delete all resources for release %q in namespace %q? [y/N]: ", releaseName, namespace)
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
