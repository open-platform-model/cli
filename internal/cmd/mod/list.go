package mod

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
	oerrors "github.com/opmodel/cli/pkg/errors"
)

// listConcurrency is the maximum number of concurrent release health evaluations.
const listConcurrency = 5

// NewModListCmd creates the mod list command.
func NewModListCmd(cfg *config.GlobalConfig) *cobra.Command {
	var kf cmdutil.K8sFlags

	var (
		namespace     string
		allNamespaces bool
		outputFlag    string
	)

	c := &cobra.Command{
		Use:        "list",
		Deprecated: "use 'opm release list' instead",
		Short:      "List deployed module releases",
		Long: `List all deployed module releases in a namespace.

Releases are discovered via inventory Secrets. Health status is evaluated
for each release by checking the state of its tracked resources.

By default, releases are listed in the namespace configured in ~/.opm/config.cue
(or "default"). Use -A to list releases across all namespaces.

Examples:
  # List releases in the default namespace
  opm mod list

  # List releases in a specific namespace
  opm mod list -n production

  # List releases across all namespaces
  opm mod list -A

  # Wide output with release ID and last applied time
  opm mod list -o wide

  # Machine-readable output
  opm mod list -o json
  opm mod list -o yaml`,
		RunE: func(c *cobra.Command, args []string) error {
			return runList(cfg, &kf, namespace, allNamespaces, outputFlag)
		},
	}

	kf.AddTo(c)

	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace (default from config)")
	c.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "List releases across all namespaces")
	c.Flags().StringVarP(&outputFlag, "output", "o", "table", "Output format (table, wide, yaml, json)")

	return c
}

// releaseSummary holds display data for a single release.
type releaseSummary struct {
	Name        string `json:"name" yaml:"name"`
	Module      string `json:"module" yaml:"module"`
	Namespace   string `json:"namespace" yaml:"namespace"`
	Version     string `json:"version" yaml:"version"`
	Status      string `json:"status" yaml:"status"`
	ReadyCount  int    `json:"readyCount" yaml:"readyCount"`
	TotalCount  int    `json:"totalCount" yaml:"totalCount"`
	ReleaseID   string `json:"releaseId" yaml:"releaseId"`
	LastApplied string `json:"lastApplied" yaml:"lastApplied"`
	Age         string `json:"age" yaml:"age"`
}

// runList executes the list command.
func runList(cfg *config.GlobalConfig, kf *cmdutil.K8sFlags, namespaceFlag string, allNamespaces bool, outputFmt string) error {
	ctx := context.Background()

	// Validate output format
	outputFormat, valid := output.ParseFormat(outputFmt)
	if !valid || outputFormat == output.FormatDir {
		return &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("invalid output format %q (valid: table, wide, yaml, json)", outputFmt),
		}
	}

	// Resolve Kubernetes configuration
	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
		NamespaceFlag:  namespaceFlag,
	})
	if err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}

	// Determine target namespace
	targetNamespace := k8sConfig.Namespace.Value
	if allNamespaces {
		targetNamespace = "" // K8s convention: empty = all namespaces
	}

	// Create Kubernetes client
	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		return err
	}

	// Discover all inventory Secrets
	inventories, err := inventory.ListInventories(ctx, k8sClient, targetNamespace)
	if err != nil {
		return &oerrors.ExitError{Code: cmdutil.ExitCodeFromK8sError(err), Err: fmt.Errorf("listing releases: %w", err)}
	}

	if len(inventories) == 0 {
		if allNamespaces {
			output.Println("No releases found")
		} else {
			output.Println(fmt.Sprintf("No releases found in namespace %q", k8sConfig.Namespace.Value))
		}
		return nil
	}

	// Evaluate health for each release in parallel
	summaries := evaluateReleaseHealth(ctx, k8sClient, inventories)

	// Render output
	return renderListOutput(summaries, outputFormat, allNamespaces)
}

// releaseHealthResult holds the result of a single release health evaluation.
type releaseHealthResult struct {
	index  int
	status kubernetes.HealthStatus
	ready  int
	total  int
}

// evaluateReleaseHealth evaluates health for all releases concurrently.
func evaluateReleaseHealth(ctx context.Context, client *kubernetes.Client, inventories []*inventory.InventorySecret) []releaseSummary {
	summaries := make([]releaseSummary, len(inventories))

	// Pre-fill metadata (no API calls needed)
	for i, inv := range inventories {
		summaries[i] = buildReleaseSummary(inv)
	}

	// Evaluate health in parallel with bounded concurrency
	var wg sync.WaitGroup
	sem := make(chan struct{}, listConcurrency)
	results := make([]releaseHealthResult, len(inventories))

	for i, inv := range inventories {
		wg.Add(1)
		go func(idx int, inv *inventory.InventorySecret) {
			defer wg.Done()
			sem <- struct{}{}        // acquire
			defer func() { <-sem }() // release

			live, missing, err := inventory.DiscoverResourcesFromInventory(ctx, client, inv)
			if err != nil {
				output.Debug("failed to discover resources for release",
					"release", inv.ReleaseMetadata.ReleaseName, "error", err)
				results[idx] = releaseHealthResult{index: idx, status: kubernetes.HealthUnknown}
				return
			}

			status, ready, total := kubernetes.QuickReleaseHealth(live, len(missing))
			results[idx] = releaseHealthResult{index: idx, status: status, ready: ready, total: total}
		}(i, inv)
	}
	wg.Wait()

	// Apply health results to summaries
	for _, r := range results {
		summaries[r.index].Status = string(r.status)
		summaries[r.index].ReadyCount = r.ready
		summaries[r.index].TotalCount = r.total
	}

	return summaries
}

// buildReleaseSummary extracts display metadata from an inventory Secret.
func buildReleaseSummary(inv *inventory.InventorySecret) releaseSummary {
	s := releaseSummary{
		Name:      inv.ReleaseMetadata.ReleaseName,
		Module:    inv.ModuleMetadata.Name,
		Namespace: inv.ReleaseMetadata.ReleaseNamespace,
		ReleaseID: inv.ReleaseMetadata.ReleaseID,
	}

	// Extract version and last applied from latest change.
	// Index[0] is the most recent change entry (move-to-front ordering).
	if len(inv.Index) > 0 {
		if change, ok := inv.Changes[inv.Index[0]]; ok {
			s.Version = change.Source.Version
			s.LastApplied = change.Timestamp
		}
	}

	// Compute age from LastTransitionTime
	if inv.ReleaseMetadata.LastTransitionTime != "" {
		if t, err := time.Parse(time.RFC3339, inv.ReleaseMetadata.LastTransitionTime); err == nil {
			s.Age = kubernetes.FormatDuration(time.Since(t))
		}
	}

	// Defaults for empty values
	if s.Version == "" {
		s.Version = "-"
	}
	if s.Module == "" {
		s.Module = "-"
	}
	if s.Age == "" {
		s.Age = "<unknown>"
	}

	return s
}

// renderListOutput formats and prints the release summaries.
func renderListOutput(summaries []releaseSummary, format output.Format, allNamespaces bool) error {
	switch format { //nolint:exhaustive // only JSON, YAML, wide are distinct; table is default
	case output.FormatJSON:
		return renderListJSON(summaries)
	case output.FormatYAML:
		return renderListYAML(summaries)
	case output.FormatWide:
		renderListWide(summaries, allNamespaces)
		return nil
	default:
		renderListTable(summaries, allNamespaces)
		return nil
	}
}

// formatStatusColumn formats the status with ready/total count.
func formatStatusColumn(status string, ready, total int) string {
	return fmt.Sprintf("%s (%d/%d)", output.FormatHealthStatus(status), ready, total)
}

// renderListTable renders the default table output.
func renderListTable(summaries []releaseSummary, allNamespaces bool) {
	var headers []string
	if allNamespaces {
		headers = []string{"NAMESPACE", "NAME", "MODULE", "VERSION", "STATUS", "AGE"}
	} else {
		headers = []string{"NAME", "MODULE", "VERSION", "STATUS", "AGE"}
	}

	tbl := output.NewTable(headers...)
	for i := range summaries {
		s := &summaries[i]
		status := formatStatusColumn(s.Status, s.ReadyCount, s.TotalCount)
		if allNamespaces {
			tbl.Row(s.Namespace, s.Name, s.Module, s.Version, status, s.Age)
		} else {
			tbl.Row(s.Name, s.Module, s.Version, status, s.Age)
		}
	}
	output.Println(tbl.String())
}

// renderListWide renders the wide table output with extra columns.
func renderListWide(summaries []releaseSummary, allNamespaces bool) {
	var headers []string
	if allNamespaces {
		headers = []string{"NAMESPACE", "NAME", "MODULE", "VERSION", "STATUS", "AGE", "RELEASE-ID", "LAST-APPLIED"}
	} else {
		headers = []string{"NAME", "MODULE", "VERSION", "STATUS", "AGE", "RELEASE-ID", "LAST-APPLIED"}
	}

	tbl := output.NewTable(headers...)
	for i := range summaries {
		s := &summaries[i]
		status := formatStatusColumn(s.Status, s.ReadyCount, s.TotalCount)
		if allNamespaces {
			tbl.Row(s.Namespace, s.Name, s.Module, s.Version, status, s.Age, s.ReleaseID, s.LastApplied)
		} else {
			tbl.Row(s.Name, s.Module, s.Version, status, s.Age, s.ReleaseID, s.LastApplied)
		}
	}
	output.Println(tbl.String())
}

// renderListJSON renders JSON output.
func renderListJSON(summaries []releaseSummary) error {
	data, err := json.MarshalIndent(summaries, "", "  ")
	if err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("marshaling to JSON: %w", err)}
	}
	output.Println(string(data))
	return nil
}

// renderListYAML renders YAML output.
func renderListYAML(summaries []releaseSummary) error {
	data, err := yaml.Marshal(summaries)
	if err != nil {
		return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("marshaling to YAML: %w", err)}
	}
	output.Println(strings.TrimSpace(string(data)))
	return nil
}
