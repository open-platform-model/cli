package instance

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/workflow/query"
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
	instanceLog := output.InstanceLogger(target.LogName)

	k8sClient, err := cmdutil.NewK8sClient(target.K8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		instanceLog.Error("connecting to cluster", "error", err)
		return err
	}

	if dryRun {
		instanceLog.Info("dry run - no changes will be made")
	} else if !force {
		if !confirmInstanceDelete(rsf.InstanceName, rsf.InstanceID, namespace) {
			instanceLog.Info("deletion canceled")
			return nil
		}
	}

	inv, liveResources, _, err := query.ResolveInventory(ctx, k8sClient, rsf, namespace, instanceLog)
	if err != nil {
		return err
	}
	if err := ensureDeleteAllowed(inv); err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	return executeInstanceDelete(ctx, k8sClient, rsf, namespace, inv, liveResources, dryRun, instanceLog)
}

// executeInstanceDelete deletes the instance's tracked workloads, then the
// ModuleInstance CR last (after all workloads are gone; skipped on dry-run).
func executeInstanceDelete(ctx context.Context, k8sClient *kubernetes.Client, rsf *cmdutil.InstanceSelectorFlags, namespace string, inv *inventory.Record, liveResources []*unstructured.Unstructured, dryRun bool, instanceLog *log.Logger) error {
	instanceLog.Info(fmt.Sprintf("deleting resources in namespace %q", namespace))

	deleteResult, err := kubernetes.Delete(ctx, k8sClient, kubernetes.DeleteOptions{
		InstanceName:          rsf.InstanceName,
		Namespace:             namespace,
		InstanceID:            rsf.InstanceID,
		DryRun:                dryRun,
		InventoryLive:         liveResources,
		InventoryRecordExists: inv != nil,
	})
	if err != nil {
		instanceLog.Error("delete failed", "error", err)
		return &opmexit.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: err, Printed: true}
	}

	if len(deleteResult.Errors) > 0 {
		instanceLog.Warn(fmt.Sprintf("%d resource(s) had errors", len(deleteResult.Errors)))
		for _, e := range deleteResult.Errors {
			instanceLog.Error(e.Error())
		}
	}

	// Delete the ModuleInstance CR last — only after every tracked workload
	// resource is gone (enhancement 0006 D1). Skipped on dry-run and on partial
	// failure (so a re-run can retry the remaining workloads).
	if !dryRun && inv != nil && len(deleteResult.Errors) == 0 {
		if err := inventory.DeleteCR(ctx, k8sClient, inv.Name, inv.Namespace); err != nil {
			instanceLog.Warn("could not delete ModuleInstance CR", "error", err)
		}
	}

	if dryRun {
		instanceLog.Info(fmt.Sprintf("dry run complete: %d resources would be deleted", deleteResult.Deleted))
	} else {
		instanceLog.Info("all resources have been deleted")
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

func ensureDeleteAllowed(inv *inventory.Record) error {
	if inventory.ResolveOwnership(inv) == inventory.ModeOperatorOwned {
		return inventory.OperatorOwnedDeleteError(inv.Name, inv.Namespace)
	}
	return nil
}

func confirmInstanceDelete(instanceName, instanceID, namespace string) bool {
	var prompt string
	if instanceName != "" {
		prompt = fmt.Sprintf("Delete all resources for instance %q in namespace %q? [y/N]: ", instanceName, namespace)
	} else {
		prompt = fmt.Sprintf("Delete all resources for instance-id %q in namespace %q? [y/N]: ", instanceID, namespace)
	}
	output.Prompt(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return answer == "y" || answer == "yes"
	}
	return false
}
