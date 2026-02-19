package mod

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/opmodel/cli/internal/cmdtypes"
	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
)

// NewModDiffCmd creates the mod diff command.
func NewModDiffCmd(cfg *config.GlobalConfig) *cobra.Command {
	var rf cmdutil.RenderFlags
	var kf cmdutil.K8sFlags

	c := &cobra.Command{
		Use:   "diff [path]",
		Short: "Show differences with cluster",
		Long: `Show differences between local module and live cluster state.

This command renders the module locally and compares each resource against
its live state on the cluster using semantic YAML diff (via dyff).

Resources are categorized as:
  - modified: exists locally and on cluster with differences
  - added:    exists locally but not on cluster
  - orphaned: exists on cluster (by OPM labels) but not in local render

Arguments:
  path    Path to module directory (default: current directory)

Examples:
  # Diff module in current directory
  opm mod diff

  # Diff with custom values
  opm mod diff -f prod-values.cue

  # Diff using specific kubeconfig
  opm mod diff --kubeconfig ~/.kube/staging --context staging-cluster`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runDiff(args, cfg, &rf, &kf)
		},
	}

	rf.AddTo(c)
	kf.AddTo(c)

	return c
}

// runDiff executes the diff command.
func runDiff(args []string, cfg *config.GlobalConfig, rf *cmdutil.RenderFlags, kf *cmdutil.K8sFlags) error {
	ctx := context.Background()

	// Resolve all Kubernetes configuration once (flag > env > config > default).
	// The resolved values are used for both the render pipeline and K8s client,
	// ensuring a single source of truth for all settings.
	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
		NamespaceFlag:  rf.Namespace,
		ProviderFlag:   rf.Provider,
	})
	if err != nil {
		return &cmdtypes.ExitError{Code: cmdtypes.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}

	// Render module via shared pipeline (diff uses RenderRelease only, NOT ShowRenderOutput,
	// because diff handles HasErrors() specially via DiffPartial)
	result, err := cmdutil.RenderRelease(ctx, cmdutil.RenderReleaseOpts{
		Args:        args,
		Values:      rf.Values,
		ReleaseName: rf.ReleaseName,
		K8sConfig:   k8sConfig,
		Config:      cfg,
	})
	if err != nil {
		return err
	}

	// Create scoped release logger
	releaseLog := output.ReleaseLogger(result.Release.Name)

	// Print warnings
	if result.HasWarnings() {
		for _, w := range result.Warnings {
			releaseLog.Warn(w)
		}
	}

	if len(result.Resources) == 0 && !result.HasErrors() {
		releaseLog.Info("no resources to diff")
		return nil
	}

	// Create Kubernetes client from pre-resolved config
	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		releaseLog.Error("connecting to cluster", "error", err)
		return err
	}

	// Create comparer
	comparer := kubernetes.NewComparer()

	// Look up the inventory to determine what's currently deployed (for orphan detection).
	// If no inventory exists, treat as "nothing previously deployed" — all rendered
	// resources will appear as additions and no orphans will be reported.
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
		// inv == nil means nothing previously deployed; diffOpts.InventoryLive stays nil.
	}

	// Run diff — handle partial render results
	var diffResult *kubernetes.DiffResult
	if result.HasErrors() {
		diffResult, err = kubernetes.DiffPartial(ctx, k8sClient, result.Resources, result.Errors, result.Release, comparer, diffOpts)
	} else {
		diffResult, err = kubernetes.Diff(ctx, k8sClient, result.Resources, result.Release, comparer, diffOpts)
	}
	if err != nil {
		releaseLog.Error("diff failed", "error", err)
		return &cmdtypes.ExitError{Code: cmdtypes.ExitGeneralError, Err: err, Printed: true}
	}

	// Print warnings from diff
	for _, w := range diffResult.Warnings {
		releaseLog.Warn(w)
	}

	// Print summary
	if diffResult.IsEmpty() {
		output.Println("No differences found")
		return nil
	}

	output.Println(diffResult.SummaryLine())
	output.Println("")

	// Print detailed diff output
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
