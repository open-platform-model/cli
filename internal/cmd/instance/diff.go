package instance

import (
	"context"
	"fmt"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/platform"
	"github.com/open-platform-model/cli/internal/workflow/query"
	"github.com/open-platform-model/cli/internal/workflow/render"
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
	c.Flags().StringVarP(&namespace, "namespace", "n", "", renderNamespaceFlagHelp)

	return c
}

// runInstanceDiff executes the instance diff command.
func runInstanceDiff(instanceFile string, cfg *config.GlobalConfig, rff *cmdutil.InstanceFileFlags, kf *cmdutil.K8sFlags, namespaceFlag string) error {
	ctx := context.Background()

	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
		NamespaceFlag:  namespaceFlag,
	})
	if err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}

	// Cluster client before render: diff follows apply's platform-source
	// precedence so the diff reflects what apply would do (0006:D21/OQ12).
	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		output.Error("connecting to cluster", "error", err)
		return err
	}

	result, err := render.FromInstanceFile(ctx, render.InstanceFileOpts{
		InstanceFilePath: instanceFile,
		ValuesFiles:      rff.Values,
		PlatformFlag:     rff.Platform,
		ClusterPlatform:  platform.ClusterPlatformGetterFor(k8sClient.Dynamic),
		SkipUnprovided:   rff.SkipUnprovided,
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

	return executeInstanceDiff(ctx, k8sClient, result.Resources, result.Instance.Name, result.Instance.Namespace, result.Instance.UUID, instanceLog)
}

// executeInstanceDiff compares the rendered resources with the cluster, prints
// the differences and reports every rendered resource it could not read or
// compare. Any such failure makes the diff incomplete, so the command then
// never prints "No differences found" and exits non-zero. An empty instanceID
// skips orphan detection.
func executeInstanceDiff(ctx context.Context, k8sClient *kubernetes.Client, resources []*unstructured.Unstructured, name, namespace, instanceID string, instanceLog *log.Logger) error {
	var diffOpts kubernetes.DiffOptions
	if instanceID != "" {
		diffOpts.InventoryLive = discoverOrphanCandidates(ctx, k8sClient, name, namespace, instanceLog)
	}

	diffResult, err := kubernetes.Diff(ctx, k8sClient, resources, name, kubernetes.NewComparer(), diffOpts)
	if err != nil {
		instanceLog.Error("diff failed", "error", err)
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err, Printed: true}
	}

	switch {
	case !diffResult.IsEmpty():
		printDifferences(diffResult)
	case len(diffResult.Errors) == 0:
		output.Println("No differences found")
	}
	if len(diffResult.Errors) == 0 {
		return nil
	}

	failures := make([]error, 0, len(diffResult.Errors))
	for _, e := range diffResult.Errors {
		instanceLog.Error("could not diff resource", "kind", e.Kind, "namespace", e.Namespace, "name", e.Name, "error", e.Err)
		failures = append(failures, e)
	}
	incomplete := fmt.Errorf("diff is incomplete: %d resource(s) could not be read or compared", len(failures))
	instanceLog.Error(incomplete.Error())
	output.Details("Fix the cause (for example missing RBAC) and run the diff again.")
	return &opmexit.ExitError{Code: failureExitCode(failures), Err: incomplete, Printed: true}
}

// failureExitCode is the exit code of an incomplete diff: the code the
// failures share (4 for a denied call, 3 for a server timeout or an unavailable
// server, 1 otherwise), or 1 when their codes differ.
func failureExitCode(failures []error) int {
	code := cmdutil.ExitCodeFromK8sError(failures[0])
	for _, f := range failures[1:] {
		if cmdutil.ExitCodeFromK8sError(f) != code {
			return opmexit.ExitGeneralError
		}
	}
	return code
}

// printDifferences prints the summary line and one block per changed resource.
func printDifferences(diffResult *kubernetes.DiffResult) {
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
}

// discoverOrphanCandidates returns the live resources the instance's
// ModuleInstance inventory tracks, for orphan detection. A missing or
// unreadable record yields none. Each tracked resource that could not be read
// is warned about, followed by one line saying orphan detection could not
// check it.
func discoverOrphanCandidates(ctx context.Context, k8sClient *kubernetes.Client, name, namespace string, instanceLog *log.Logger) []*unstructured.Unstructured {
	// Orphan detection reads status.inventory from the ModuleInstance CR.
	inv, invErr := inventory.GetRecord(ctx, k8sClient, name, namespace)
	if invErr != nil {
		instanceLog.Debug("could not read inventory for diff", "error", invErr)
		return nil
	}
	if inv == nil {
		return nil
	}
	live, _, unreadable, discoverErr := inventory.DiscoverResourcesFromInventory(ctx, k8sClient, inv)
	if discoverErr != nil {
		instanceLog.Debug("inventory discovery failed", "error", discoverErr)
		return nil
	}
	if n := len(unreadable); n > 0 {
		query.WarnUnreadable(instanceLog, unreadable)
		instanceLog.Warn(fmt.Sprintf("orphan detection could not check %d tracked resource(s)", n))
	}
	return live
}
