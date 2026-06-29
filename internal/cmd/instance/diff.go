package instance

import (
	"context"
	"fmt"

	opmexit "github.com/opmodel/cli/internal/exit"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/internal/workflow/render"
)

// NewInstanceDiffCmd creates the instance diff command.
func NewInstanceDiffCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rff cmdutil.InstanceFileFlags
	var kf cmdutil.K8sFlags
	var namespace string

	c := &cobra.Command{
		Use:   "diff <instance.cue>",
		Short: "Show differences between instance file and cluster",
		Long: `Show differences between an instance file and live cluster state.

Arguments:
  instance.cue    Path to the instance .cue file (required)

Examples:
  # Diff an instance file against the cluster
  opm instance diff ./jellyfin_instance.cue`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runInstanceDiff(args[0], cfg, &rff, &kf, namespace)
		},
	}

	rff.AddTo(c)
	kf.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Target namespace")

	return c
}

// runInstanceDiff executes the instance diff command.
func runInstanceDiff(instanceFile string, cfg *config.GlobalConfig, rff *cmdutil.InstanceFileFlags, kf *cmdutil.K8sFlags, namespaceFlag string) error { //nolint:gocyclo // orchestration function; complexity is inherent
	ctx := context.Background()

	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
		NamespaceFlag:  namespaceFlag,
		ProviderFlag:   rff.Provider,
	})
	if err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}

	result, err := render.FromInstanceFile(ctx, render.InstanceFileOpts{
		InstanceFilePath: instanceFile,
		ValuesFiles:      rff.Values,
		K8sConfig:        k8sConfig,
		Config:           cfg,
	})
	if err != nil {
		return err
	}

	instanceLog := output.InstanceLogger(result.Instance.Name)

	if result.HasWarnings() {
		for _, w := range result.Warnings {
			instanceLog.Warn(w)
		}
	}

	if len(result.Resources) == 0 {
		instanceLog.Info("no resources to diff")
		return nil
	}

	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		instanceLog.Error("connecting to cluster", "error", err)
		return err
	}

	comparer := kubernetes.NewComparer()

	var diffOpts kubernetes.DiffOptions
	instanceID := result.Instance.UUID
	if instanceID != "" {
		inv, invErr := inventory.GetInventory(ctx, k8sClient, result.Instance.Name, result.Instance.Namespace, instanceID)
		if invErr != nil {
			instanceLog.Debug("could not read inventory for diff", "error", invErr)
		} else if inv != nil {
			liveResources, _, invDiscoverErr := inventory.DiscoverResourcesFromInventory(ctx, k8sClient, inv)
			if invDiscoverErr != nil {
				instanceLog.Debug("inventory discovery failed", "error", invDiscoverErr)
			} else {
				diffOpts.InventoryLive = liveResources
			}
		}
	}

	diffResult, err := kubernetes.Diff(ctx, k8sClient, result.Resources, result.Instance.Name, comparer, diffOpts)
	if err != nil {
		instanceLog.Error("diff failed", "error", err)
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err, Printed: true}
	}

	for _, w := range diffResult.Warnings {
		instanceLog.Warn(w)
	}

	if diffResult.IsEmpty() {
		output.Println("No differences found")
		return nil
	}

	output.Println(diffResult.SummaryLine())
	output.Println("")

	for _, rd := range diffResult.Resources {
		switch rd.State {
		case kubernetes.ResourceModified:
			if rd.Namespace != "" {
				output.Println(fmt.Sprintf("--- %s/%s (%s) [modified]", rd.Kind, rd.Name, rd.Namespace))
			} else {
				output.Println(fmt.Sprintf("--- %s/%s [modified]", rd.Kind, rd.Name))
			}
			output.Println(rd.Diff)
		case kubernetes.ResourceAdded:
			if rd.Namespace != "" {
				output.Println(fmt.Sprintf("+++ %s/%s (%s) [new resource]", rd.Kind, rd.Name, rd.Namespace))
			} else {
				output.Println(fmt.Sprintf("+++ %s/%s [new resource]", rd.Kind, rd.Name))
			}
		case kubernetes.ResourceOrphaned:
			if rd.Namespace != "" {
				output.Println(fmt.Sprintf("~~~ %s/%s (%s) [orphaned - will be removed on next apply]", rd.Kind, rd.Name, rd.Namespace))
			} else {
				output.Println(fmt.Sprintf("~~~ %s/%s [orphaned - will be removed on next apply]", rd.Kind, rd.Name))
			}
		case kubernetes.ResourceUnchanged:
			// No output for unchanged resources in diff view
		}
	}

	return nil
}
