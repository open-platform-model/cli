package instance

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
	"github.com/opmodel/cli/internal/workflow/query"
	"github.com/opmodel/cli/pkg/ownership"
)

// NewInstanceDeleteCmd creates the instance delete command.
func NewInstanceDeleteCmd(cfg *config.GlobalConfig) *cobra.Command {
	var kf cmdutil.K8sFlags
	var namespace string

	var (
		forceFlag  bool
		dryRunFlag bool
	)

	c := &cobra.Command{
		Use:   "delete <file|name|uuid>",
		Short: "Delete instance resources from cluster",
		Long: `Delete all resources belonging to an OPM instance from a Kubernetes cluster.

Arguments:
  file         Path to an instance.cue file or directory containing one.
               The instance name and namespace are read from the file's metadata.
               --namespace overrides the namespace found in the file.
  name         Instance name (use -n / --namespace to scope by namespace).
  uuid         Instance UUID.

Examples:
  # Delete by instance.cue file in the current directory
  opm instance delete .

  # Delete by instance.cue file path
  opm instance delete ./instances/jellyfin/instance.cue -n media

  # Delete by name
  opm instance delete jellyfin -n media

  # Preview what would be deleted
  opm instance delete jellyfin -n media --dry-run

  # Skip confirmation prompt
  opm instance delete jellyfin -n media --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runInstanceDelete(args[0], cfg, &kf, namespace, forceFlag, dryRunFlag)
		},
	}

	kf.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Target namespace")
	c.Flags().BoolVar(&forceFlag, "force", false, "Skip confirmation prompt")
	c.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Preview without deleting")

	return c
}

func runInstanceDelete(identifier string, cfg *config.GlobalConfig, kf *cmdutil.K8sFlags, namespaceFlag string, force, dryRun bool) error {
	ctx := context.Background()

	target, err := cmdutil.ResolveInstanceTarget(identifier, cfg, kf, namespaceFlag)
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
		if !confirmInstanceDelete(rsf.ReleaseName, rsf.ReleaseID, namespace) {
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
		output.Println(output.FormatCheckmark("Instance deleted"))
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

func ensureDeleteAllowed(inv *inventory.ReleaseInventoryRecord) error {
	if inv == nil {
		return nil
	}
	return ownership.EnsureCLIMutable(string(inv.NormalizedCreatedBy()), inv.ReleaseMetadata.ReleaseName, inv.ReleaseMetadata.ReleaseNamespace)
}

func confirmInstanceDelete(releaseName, releaseID, namespace string) bool {
	var prompt string
	if releaseName != "" {
		prompt = fmt.Sprintf("Delete all resources for instance %q in namespace %q? [y/N]: ", releaseName, namespace)
	} else {
		prompt = fmt.Sprintf("Delete all resources for instance-id %q in namespace %q? [y/N]: ", releaseID, namespace)
	}
	output.Prompt(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return answer == "y" || answer == "yes"
	}
	return false
}
