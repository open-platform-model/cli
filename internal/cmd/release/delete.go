package release

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
	oerrors "github.com/opmodel/cli/pkg/errors"
)

// NewReleaseDeleteCmd creates the release delete command.
func NewReleaseDeleteCmd(cfg *config.GlobalConfig) *cobra.Command {
	var kf cmdutil.K8sFlags
	var namespace string

	var (
		forceFlag  bool
		dryRunFlag bool
	)

	c := &cobra.Command{
		Use:   "delete <name|uuid>",
		Short: "Delete release resources from cluster",
		Long: `Delete all resources belonging to an OPM release from a Kubernetes cluster.

Arguments:
  name|uuid    Release name or UUID (required)

Examples:
  # Delete release by name
  opm release delete jellyfin -n media

  # Preview what would be deleted
  opm release delete jellyfin -n media --dry-run

  # Skip confirmation prompt
  opm release delete jellyfin -n media --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runReleaseDelete(args[0], cfg, &kf, namespace, forceFlag, dryRunFlag)
		},
	}

	kf.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Target namespace")
	c.Flags().BoolVar(&forceFlag, "force", false, "Skip confirmation prompt")
	c.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Preview without deleting")

	return c
}

func runReleaseDelete(identifier string, cfg *config.GlobalConfig, kf *cmdutil.K8sFlags, namespaceFlag string, force, dryRun bool) error {
	ctx := context.Background()

	name, uuid := cmdutil.ResolveReleaseIdentifier(identifier)
	rsf := &cmdutil.ReleaseSelectorFlags{
		ReleaseName: name,
		ReleaseID:   uuid,
		Namespace:   namespaceFlag,
	}

	if err := rsf.Validate(); err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err}
	}

	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
		NamespaceFlag:  namespaceFlag,
	})
	if err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}
	if err := cmdutil.RequireNamespace(k8sConfig); err != nil {
		return err
	}

	namespace := k8sConfig.Namespace.Value
	output.Debug("resolved kubernetes config",
		"kubeconfig", k8sConfig.Kubeconfig.Value,
		"context", k8sConfig.Context.Value,
		"namespace", namespace,
	)

	releaseLog := output.ReleaseLogger(rsf.LogName())

	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		releaseLog.Error("connecting to cluster", "error", err)
		return err
	}

	if dryRun {
		releaseLog.Info("dry run - no changes will be made")
	} else if !force {
		if !confirmReleaseDelete(rsf.ReleaseName, rsf.ReleaseID, namespace) {
			releaseLog.Info("deletion canceled")
			return nil
		}
	}

	inv, liveResources, _, err := cmdutil.ResolveInventory(ctx, k8sClient, rsf, namespace, releaseLog)
	if err != nil {
		return err
	}

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
		releaseLog.Error("delete failed", "error", err)
		return &oerrors.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: err, Printed: true}
	}

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
		return &oerrors.ExitError{
			Code:    oerrors.ExitGeneralError,
			Err:     fmt.Errorf("%d resource(s) failed to delete", len(deleteResult.Errors)),
			Printed: true,
		}
	}
	return nil
}

func confirmReleaseDelete(releaseName, releaseID, namespace string) bool {
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
