package release

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	opmexit "github.com/opmodel/cli/internal/exit"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
	workflowownership "github.com/opmodel/cli/internal/workflow/ownership"
	"github.com/opmodel/cli/internal/workflow/query"
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
		Use:   "delete <file|name|uuid>",
		Short: "Delete release resources from cluster",
		Long: `Delete all resources belonging to an OPM release from a Kubernetes cluster.

Arguments:
  file         Path to a release.cue file or directory containing one.
               The release name and namespace are read from the file's metadata.
               --namespace overrides the namespace found in the file.
  name         Release name (use -n / --namespace to scope by namespace).
  uuid         Release UUID.

Examples:
  # Delete by release.cue file in the current directory
  opm release delete .

  # Delete by release.cue file path
  opm release delete ./releases/jellyfin/release.cue -n media

  # Delete by name
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

	target, err := cmdutil.ResolveReleaseTarget(identifier, cfg, kf, namespaceFlag)
	if err != nil {
		return err
	}
	cmdutil.LogResolvedKubernetesConfig(target.Namespace, target.K8sConfig.Kubeconfig.Value, target.K8sConfig.Context.Value)

	rsf := target.Selector
	namespace := target.Namespace
	releaseLog := output.ReleaseLogger(target.LogName)

	k8sClient, err := cmdutil.NewK8sClient(target.K8sConfig, cfg.Log.Kubernetes.APIWarnings)
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

	inv, liveResources, _, err := query.ResolveInventory(ctx, k8sClient, rsf, namespace, releaseLog)
	if err != nil {
		return err
	}
	if err := ensureDeleteAllowed(inv); err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
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
		return &opmexit.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: err, Printed: true}
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
		return &opmexit.ExitError{
			Code:    opmexit.ExitGeneralError,
			Err:     fmt.Errorf("%d resource(s) failed to delete", len(deleteResult.Errors)),
			Printed: true,
		}
	}
	return nil
}

func ensureDeleteAllowed(inv *inventory.InventorySecret) error {
	return workflowownership.EnsureCLIMutable(inv)
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
