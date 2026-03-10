package release

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

// NewReleaseDiffCmd creates the release diff command.
func NewReleaseDiffCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rff cmdutil.ReleaseFileFlags
	var kf cmdutil.K8sFlags
	var namespace string

	c := &cobra.Command{
		Use:   "diff <release.cue>",
		Short: "Show differences between release file and cluster",
		Long: `Show differences between a release file and live cluster state.

Arguments:
  release.cue    Path to the release .cue file (required)

Examples:
  # Diff a release file against the cluster
  opm release diff ./jellyfin_release.cue

  # Diff with a local module
  opm release diff ./jellyfin_release.cue --module ./my-module`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runReleaseDiff(args[0], cfg, &rff, &kf, namespace)
		},
	}

	rff.AddTo(c)
	kf.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Target namespace")

	return c
}

// runReleaseDiff executes the release diff command.
func runReleaseDiff(releaseFile string, cfg *config.GlobalConfig, rff *cmdutil.ReleaseFileFlags, kf *cmdutil.K8sFlags, namespaceFlag string) error { //nolint:gocyclo // orchestration function; complexity is inherent
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

	result, err := render.ReleaseFile(ctx, render.ReleaseFileOpts{
		ReleaseFilePath: releaseFile,
		ValuesFiles:     rff.Values,
		ModulePath:      rff.Module,
		K8sConfig:       k8sConfig,
		Config:          cfg,
	})
	if err != nil {
		return err
	}

	releaseLog := output.ReleaseLogger(result.Release.Name)

	if result.HasWarnings() {
		for _, w := range result.Warnings {
			releaseLog.Warn(w)
		}
	}

	if len(result.Resources) == 0 {
		releaseLog.Info("no resources to diff")
		return nil
	}

	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		releaseLog.Error("connecting to cluster", "error", err)
		return err
	}

	comparer := kubernetes.NewComparer()

	var diffOpts kubernetes.DiffOptions
	releaseID := result.Release.UUID
	if releaseID != "" {
		inv, invErr := inventory.GetInventory(ctx, k8sClient, result.Release.Name, result.Release.Namespace, releaseID)
		if invErr != nil {
			releaseLog.Debug("could not read inventory for diff", "error", invErr)
		} else if inv != nil {
			liveResources, _, invDiscoverErr := inventory.DiscoverResourcesFromInventory(ctx, k8sClient, inv)
			if invDiscoverErr != nil {
				releaseLog.Debug("inventory discovery failed", "error", invDiscoverErr)
			} else {
				diffOpts.InventoryLive = liveResources
			}
		}
	}

	diffResult, err := kubernetes.Diff(ctx, k8sClient, result.Resources, result.Release.Name, comparer, diffOpts)
	if err != nil {
		releaseLog.Error("diff failed", "error", err)
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err, Printed: true}
	}

	for _, w := range diffResult.Warnings {
		releaseLog.Warn(w)
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
