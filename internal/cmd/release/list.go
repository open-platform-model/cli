package release

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

const releaseListConcurrency = 5

// NewReleaseListCmd creates the release list command.
func NewReleaseListCmd(cfg *config.GlobalConfig) *cobra.Command {
	var kf cmdutil.K8sFlags
	var namespace string
	var allNamespaces bool
	var outputFlag string

	c := &cobra.Command{
		Use:   "list",
		Short: "List deployed releases",
		Long: `List all deployed OPM releases in a namespace.

Examples:
  # List releases in the default namespace
  opm release list

  # List releases in a specific namespace
  opm release list -n production

  # List across all namespaces
  opm release list -A`,
		RunE: func(c *cobra.Command, args []string) error {
			return runReleaseList(cfg, &kf, namespace, allNamespaces, outputFlag)
		},
	}

	kf.AddTo(c)
	c.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace (default from config)")
	c.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "List releases across all namespaces")
	c.Flags().StringVarP(&outputFlag, "output", "o", "table", "Output format (table, wide, yaml, json)")

	return c
}

func runReleaseList(cfg *config.GlobalConfig, kf *cmdutil.K8sFlags, namespaceFlag string, allNamespaces bool, outputFmt string) error {
	ctx := context.Background()

	outputFormat, valid := output.ParseFormat(outputFmt)
	if !valid || outputFormat == output.FormatDir {
		return &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("invalid output format %q (valid: table, wide, yaml, json)", outputFmt),
		}
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

	targetNamespace := k8sConfig.Namespace.Value
	if allNamespaces {
		targetNamespace = ""
	} else {
		if err := cmdutil.RequireNamespace(k8sConfig); err != nil {
			return err
		}
	}

	k8sClient, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		return err
	}

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

	summaries := evalReleaseHealth(ctx, k8sClient, inventories)
	return renderRelListOutput(summaries, outputFormat, allNamespaces)
}

type relReleaseSummary struct {
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

type relHealthResult struct {
	index  int
	status kubernetes.HealthStatus
	ready  int
	total  int
}

func evalReleaseHealth(ctx context.Context, client *kubernetes.Client, inventories []*inventory.InventorySecret) []relReleaseSummary {
	summaries := make([]relReleaseSummary, len(inventories))
	for i, inv := range inventories {
		summaries[i] = buildRelSummary(inv)
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, releaseListConcurrency)
	results := make([]relHealthResult, len(inventories))

	for i, inv := range inventories {
		wg.Add(1)
		go func(idx int, inv *inventory.InventorySecret) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			live, missing, err := inventory.DiscoverResourcesFromInventory(ctx, client, inv)
			if err != nil {
				results[idx] = relHealthResult{index: idx, status: kubernetes.HealthUnknown}
				return
			}
			status, ready, total := kubernetes.QuickReleaseHealth(live, len(missing))
			results[idx] = relHealthResult{index: idx, status: status, ready: ready, total: total}
		}(i, inv)
	}
	wg.Wait()

	for _, r := range results {
		summaries[r.index].Status = string(r.status)
		summaries[r.index].ReadyCount = r.ready
		summaries[r.index].TotalCount = r.total
	}
	return summaries
}

func buildRelSummary(inv *inventory.InventorySecret) relReleaseSummary {
	s := relReleaseSummary{
		Name:      inv.ReleaseMetadata.ReleaseName,
		Module:    inv.ModuleMetadata.Name,
		Namespace: inv.ReleaseMetadata.ReleaseNamespace,
		ReleaseID: inv.ReleaseMetadata.ReleaseID,
	}
	if len(inv.Index) > 0 {
		if change, ok := inv.Changes[inv.Index[0]]; ok {
			s.Version = change.Source.Version
			s.LastApplied = change.Timestamp
		}
	}
	if inv.ReleaseMetadata.LastTransitionTime != "" {
		if t, err := time.Parse(time.RFC3339, inv.ReleaseMetadata.LastTransitionTime); err == nil {
			s.Age = kubernetes.FormatDuration(time.Since(t))
		}
	}
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

func renderRelListOutput(summaries []relReleaseSummary, format output.Format, allNamespaces bool) error {
	switch format { //nolint:exhaustive // only JSON, YAML, wide are distinct; table is default
	case output.FormatJSON:
		data, err := json.MarshalIndent(summaries, "", "  ")
		if err != nil {
			return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("marshaling to JSON: %w", err)}
		}
		output.Println(string(data))
		return nil
	case output.FormatYAML:
		data, err := yaml.Marshal(summaries)
		if err != nil {
			return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("marshaling to YAML: %w", err)}
		}
		output.Println(strings.TrimSpace(string(data)))
		return nil
	case output.FormatWide:
		renderRelListWide(summaries, allNamespaces)
		return nil
	default:
		renderRelListTable(summaries, allNamespaces)
		return nil
	}
}

func formatRelStatusColumn(status string, ready, total int) string {
	return fmt.Sprintf("%s (%d/%d)", output.FormatHealthStatus(status), ready, total)
}

func renderRelListTable(summaries []relReleaseSummary, allNamespaces bool) {
	var headers []string
	if allNamespaces {
		headers = []string{"NAMESPACE", "NAME", "MODULE", "VERSION", "STATUS", "AGE"}
	} else {
		headers = []string{"NAME", "MODULE", "VERSION", "STATUS", "AGE"}
	}
	tbl := output.NewTable(headers...)
	for i := range summaries {
		s := &summaries[i]
		status := formatRelStatusColumn(s.Status, s.ReadyCount, s.TotalCount)
		if allNamespaces {
			tbl.Row(s.Namespace, s.Name, s.Module, s.Version, status, s.Age)
		} else {
			tbl.Row(s.Name, s.Module, s.Version, status, s.Age)
		}
	}
	output.Println(tbl.String())
}

func renderRelListWide(summaries []relReleaseSummary, allNamespaces bool) {
	var headers []string
	if allNamespaces {
		headers = []string{"NAMESPACE", "NAME", "MODULE", "VERSION", "STATUS", "AGE", "RELEASE-ID", "LAST-APPLIED"}
	} else {
		headers = []string{"NAME", "MODULE", "VERSION", "STATUS", "AGE", "RELEASE-ID", "LAST-APPLIED"}
	}
	tbl := output.NewTable(headers...)
	for i := range summaries {
		s := &summaries[i]
		status := formatRelStatusColumn(s.Status, s.ReadyCount, s.TotalCount)
		if allNamespaces {
			tbl.Row(s.Namespace, s.Name, s.Module, s.Version, status, s.Age, s.ReleaseID, s.LastApplied)
		} else {
			tbl.Row(s.Name, s.Module, s.Version, status, s.Age, s.ReleaseID, s.LastApplied)
		}
	}
	output.Println(tbl.String())
}
