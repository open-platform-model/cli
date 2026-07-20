package instance

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/operator"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/workflow/query"
)

// NewInstanceDeleteCmd creates the instance delete command.
func NewInstanceDeleteCmd(cfg *config.GlobalConfig) *cobra.Command {
	var kf cmdutil.K8sFlags
	var namespace string

	var (
		forceFlag   bool
		dryRunFlag  bool
		timeoutFlag time.Duration
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
			return runInstanceDelete(args[0], cfg, &kf, namespace, forceFlag, dryRunFlag, timeoutFlag)
		},
	}

	kf.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Target namespace")
	c.Flags().BoolVar(&forceFlag, "force", false, "Skip confirmation prompt")
	c.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Preview without deleting")
	c.Flags().DurationVar(&timeoutFlag, "timeout", inventory.DefaultReconcileTimeout,
		"Bound on the operator-cleanup wait (operator-managed instances only)")

	return c
}

func runInstanceDelete(identifier string, cfg *config.GlobalConfig, kf *cmdutil.K8sFlags, namespaceFlag string, force, dryRun bool, timeout time.Duration) error {
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

	// Ownership is the single branch point (0006 D18): an operator-owned
	// instance is deleted by deleting its CR and letting the operator's
	// finalizer prune the workloads.
	if inventory.ResolveOwnership(inv) == inventory.ModeOperatorOwned {
		return deleteOperatorOwned(ctx, k8sClient, inv, timeout, dryRun, instanceLog)
	}

	return executeInstanceDelete(ctx, k8sClient, rsf, namespace, inv, liveResources, dryRun, instanceLog)
}

// deleteOperatorOwned deletes an operator-managed instance by removing its
// ModuleInstance CR and waiting for the operator's cleanup finalizer to prune
// the workloads (design LD7).
//
// The readiness guard is the point of this function. A ModuleInstance carries
// the operator's cleanup finalizer, so deleting the CR with no controller
// running does not delete anything — it wedges the CR in Terminating forever,
// with its workloads orphaned and unreachable through the CLI. That is the same
// footgun `opm operator uninstall` guards from the other side, and it has no
// --force bypass here: forcing it produces the wedge, it does not avoid it.
func deleteOperatorOwned(ctx context.Context, k8sClient *kubernetes.Client, inv *inventory.Record, timeout time.Duration, dryRun bool, instanceLog *log.Logger) error {
	if err := operator.CheckReady(ctx, k8sClient); err != nil {
		var notReady *operator.NotReadyError
		if errors.As(err, &notReady) {
			notReady.Hint = fmt.Sprintf(
				"instance %q is operator-managed, and deleting its ModuleInstance now would wedge it in Terminating on the %s finalizer with its workloads orphaned",
				inv.Name, inventory.CleanupFinalizer)
		}
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err}
	}

	// spec.prune decides whether removing the CR removes the workloads. It has
	// no CRD default, so it is false unless someone set it, and the operator
	// then orphans the workloads on purpose. The CLI does not write this field,
	// so a CLI-created instance orphans by default — say so rather than
	// reporting a cleanup that will not happen.
	entries := len(inv.Inventory.Entries)
	if dryRun {
		if inv.Prune {
			instanceLog.Info(fmt.Sprintf(
				"dry run complete: ModuleInstance %q would be deleted and the operator would prune its %d tracked resource(s)",
				inv.Name, entries))
		} else {
			instanceLog.Info(fmt.Sprintf(
				"dry run complete: ModuleInstance %q would be deleted; its %d tracked resource(s) would be left running (spec.prune is not set)",
				inv.Name, entries))
		}
		return nil
	}

	if inv.Prune {
		instanceLog.Info("deleting the ModuleInstance — the operator prunes its resources", "instance", inv.Name)
	} else {
		instanceLog.Warn("spec.prune is not set — the operator will remove the ModuleInstance but leave its resources running",
			"instance", inv.Name, "resources", entries)
	}

	if err := inventory.DeleteCR(ctx, k8sClient, inv.Name, inv.Namespace); err != nil {
		return &opmexit.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: err}
	}

	instanceLog.Info("waiting for the operator to finish cleanup")
	if err := inventory.WaitForAbsence(ctx, k8sClient, inv.Name, inv.Namespace, timeout); err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
	}

	// The CR's disappearance proves the finalizer completed; it does not prove
	// anything was pruned. Report only what was actually established.
	if inv.Prune {
		instanceLog.Info("all resources have been deleted")
		output.Println(output.FormatCheckmark(fmt.Sprintf("Instance deleted — operator pruned %d resources", entries)))
		return nil
	}

	output.Println(output.FormatCheckmark(fmt.Sprintf(
		"ModuleInstance deleted — %d resource(s) left running", entries)))
	output.Details(fmt.Sprintf(
		"The operator orphaned them because spec.prune is not set on this instance.\n"+
			"To have the operator remove workloads on delete, set it before deleting:\n"+
			"  kubectl patch moduleinstance %s -n %s --type=merge -p '{\"spec\":{\"prune\":true}}'\n"+
			"Otherwise remove them yourself with 'kubectl delete'.",
		inv.Name, inv.Namespace))
	return nil
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
