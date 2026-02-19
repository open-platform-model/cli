package mod

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdtypes"
	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
)

// NewModDeleteCmd creates the mod delete command.
func NewModDeleteCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rsf cmdutil.ReleaseSelectorFlags
	var kf cmdutil.K8sFlags

	// Delete-specific flags (local to this command)
	var (
		forceFlag          bool
		dryRunFlag         bool
		waitFlag           bool
		ignoreNotFoundFlag bool
	)

	c := &cobra.Command{
		Use:   "delete",
		Short: "Delete release resources from cluster",
		Long: `Delete all resources belonging to an OPM release from a Kubernetes cluster.

Resources are discovered via the release inventory Secret, so the original
module source is not required. Resources are deleted in reverse weight order
(webhooks first, CRDs last).

Exactly one of --release-name or --release-id is required to identify the release.
The --namespace flag defaults to the value configured in ~/.opm/config.cue.

Examples:
  # Delete release by name
  opm mod delete --release-name my-app -n production

  # Delete release by release ID
  opm mod delete --release-id a1b2c3d4-e5f6-7890-abcd-ef1234567890 -n production

  # Preview what would be deleted
  opm mod delete --release-name my-app -n production --dry-run

  # Skip confirmation prompt
  opm mod delete --release-name my-app -n production --force`,
		RunE: func(c *cobra.Command, args []string) error {
			return runDelete(args, cfg, &rsf, &kf, forceFlag, dryRunFlag, waitFlag, ignoreNotFoundFlag)
		},
	}

	rsf.AddTo(c)
	kf.AddTo(c)

	// Delete-specific flags
	c.Flags().BoolVar(&forceFlag, "force", false,
		"Skip confirmation prompt")
	c.Flags().BoolVar(&dryRunFlag, "dry-run", false,
		"Preview without deleting")
	c.Flags().BoolVar(&waitFlag, "wait", false,
		"Wait for resources to be deleted")
	c.Flags().BoolVar(&ignoreNotFoundFlag, "ignore-not-found", false,
		"Exit 0 when no resources match the selector")

	return c
}

// runDelete executes the delete command.
func runDelete(_ []string, cfg *config.GlobalConfig, rsf *cmdutil.ReleaseSelectorFlags, kf *cmdutil.K8sFlags, force, dryRun, _ /* wait */, ignoreNotFound bool) error { //nolint:gocyclo // orchestration function; complexity is inherent
	ctx := context.Background()

	// Validate release selector flags
	if err := rsf.Validate(); err != nil {
		return &cmdtypes.ExitError{Code: cmdtypes.ExitGeneralError, Err: err}
	}

	// Resolve Kubernetes configuration with local flags
	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
		NamespaceFlag:  rsf.Namespace,
	})
	if err != nil {
		return &cmdtypes.ExitError{Code: cmdtypes.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}

	namespace := k8sConfig.Namespace.Value

	// Log resolved k8s config at DEBUG level
	output.Debug("resolved kubernetes config",
		"kubeconfig", k8sConfig.Kubeconfig.Value,
		"context", k8sConfig.Context.Value,
		"namespace", namespace,
	)

	// Create scoped release logger using shared LogName helper
	releaseLog := output.ReleaseLogger(rsf.LogName())

	// Create Kubernetes client from pre-resolved config
	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		releaseLog.Error("connecting to cluster", "error", err)
		return err
	}

	// If dry-run, skip confirmation
	if dryRun {
		releaseLog.Info("dry run - no changes will be made")
	} else if !force {
		// Prompt for confirmation
		if !confirmDelete(rsf.ReleaseName, rsf.ReleaseID, namespace) {
			releaseLog.Info("deletion canceled")
			return nil
		}
	}

	// Resolve the inventory Secret for this release.
	// --release-id: direct GET by name (opm.<name>.<uuid>), with UUID label fallback.
	// --release-name: label scan on inventory Secrets only (FindInventoryByReleaseName).
	var inv *inventory.InventorySecret
	var invErr error
	switch {
	case rsf.ReleaseID != "":
		relName := rsf.ReleaseName
		if relName == "" {
			relName = rsf.ReleaseID
		}
		inv, invErr = inventory.GetInventory(ctx, k8sClient, relName, namespace, rsf.ReleaseID)
	case rsf.ReleaseName != "":
		inv, invErr = inventory.FindInventoryByReleaseName(ctx, k8sClient, rsf.ReleaseName, namespace)
	}
	if invErr != nil {
		releaseLog.Error("reading inventory", "error", invErr)
		return &cmdtypes.ExitError{Code: cmdtypes.ExitGeneralError, Err: fmt.Errorf("reading inventory: %w", invErr)}
	}
	if inv == nil {
		name := rsf.ReleaseName
		if name == "" {
			name = rsf.ReleaseID
		}
		notFound := &kubernetes.ReleaseNotFoundError{Name: name, Namespace: namespace}
		if ignoreNotFound {
			releaseLog.Info("release not found (ignored)")
			return nil
		}
		releaseLog.Error("release not found", "name", name, "namespace", namespace)
		return &cmdtypes.ExitError{Code: cmdtypes.ExitNotFound, Err: notFound, Printed: true}
	}

	liveResources, _, discoverErr := inventory.DiscoverResourcesFromInventory(ctx, k8sClient, inv)
	if discoverErr != nil {
		releaseLog.Error("discovering resources from inventory", "error", discoverErr)
		return &cmdtypes.ExitError{Code: cmdtypes.ExitGeneralError, Err: fmt.Errorf("discovering resources: %w", discoverErr)}
	}

	// Delete resources
	releaseLog.Info(fmt.Sprintf("deleting resources in namespace %q", namespace))

	deleteOpts := kubernetes.DeleteOptions{
		ReleaseName:              rsf.ReleaseName,
		Namespace:                namespace,
		ReleaseID:                rsf.ReleaseID,
		DryRun:                   dryRun,
		InventoryLive:            liveResources,
		InventorySecretName:      inventory.SecretName(inv.ReleaseMetadata.ReleaseName, inv.ReleaseMetadata.ReleaseID),
		InventorySecretNamespace: namespace,
	}

	deleteResult, err := kubernetes.Delete(ctx, k8sClient, deleteOpts)
	if err != nil {
		if ignoreNotFound && kubernetes.IsNoResourcesFound(err) {
			releaseLog.Info("no resources found (ignored)")
			return nil
		}
		releaseLog.Error("delete failed", "error", err)
		return &cmdtypes.ExitError{Code: cmdtypes.ExitCodeFromK8sError(err), Err: err, Printed: true}
	}

	// Report results
	if len(deleteResult.Errors) > 0 {
		releaseLog.Warn(fmt.Sprintf("%d resource(s) had errors", len(deleteResult.Errors)))
		for _, e := range deleteResult.Errors {
			releaseLog.Error(e.Error())
		}
	}

	if dryRun {
		releaseLog.Info(fmt.Sprintf("dry run complete: %d resources would be deleted", deleteResult.Deleted))
	} else {
		releaseLog.Info("all resources have been deleted")
		output.Println(output.FormatCheckmark("Release deleted"))
	}

	if len(deleteResult.Errors) > 0 {
		return &cmdtypes.ExitError{
			Code:    cmdtypes.ExitGeneralError,
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
