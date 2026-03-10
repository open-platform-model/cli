package cmdutil

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
	oerrors "github.com/opmodel/cli/pkg/errors"
	"gopkg.in/yaml.v3"
)

// ReleaseSummary holds display data for a single release.
type ReleaseSummary struct {
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

type releaseHealthResult struct {
	index  int
	status kubernetes.HealthStatus
	ready  int
	total  int
}

// EvaluateReleaseHealth evaluates health for all releases concurrently.
func EvaluateReleaseHealth(ctx context.Context, client *kubernetes.Client, inventories []*inventory.InventorySecret, concurrency int, logDiscoveryFailures bool) []ReleaseSummary {
	summaries := make([]ReleaseSummary, len(inventories))
	for i, inv := range inventories {
		summaries[i] = BuildReleaseSummary(inv)
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	results := make([]releaseHealthResult, len(inventories))

	for i, inv := range inventories {
		wg.Add(1)
		go func(idx int, inv *inventory.InventorySecret) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			live, missing, err := inventory.DiscoverResourcesFromInventory(ctx, client, inv)
			if err != nil {
				if logDiscoveryFailures {
					output.Debug("failed to discover resources for release", "release", inv.ReleaseMetadata.ReleaseName, "error", err)
				}
				results[idx] = releaseHealthResult{index: idx, status: kubernetes.HealthUnknown}
				return
			}

			status, ready, total := kubernetes.QuickReleaseHealth(live, len(missing))
			results[idx] = releaseHealthResult{index: idx, status: status, ready: ready, total: total}
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

// BuildReleaseSummary extracts display metadata from an inventory Secret.
func BuildReleaseSummary(inv *inventory.InventorySecret) ReleaseSummary {
	s := ReleaseSummary{
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

// RenderReleaseListOutput formats and prints release summaries.
func RenderReleaseListOutput(summaries []ReleaseSummary, format output.Format, allNamespaces bool) error {
	switch format { //nolint:exhaustive
	case output.FormatJSON:
		data, err := json.MarshalIndent(summaries, "", "  ")
		if err != nil {
			return &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("marshaling to JSON: %w", err)}
		}
		output.Println(string(data))
		return nil
	case output.FormatYAML:
		data, err := outputYAMLMarshal(summaries)
		if err != nil {
			return err
		}
		output.Println(strings.TrimSpace(string(data)))
		return nil
	case output.FormatWide:
		renderReleaseListTable(summaries, allNamespaces, true)
		return nil
	default:
		renderReleaseListTable(summaries, allNamespaces, false)
		return nil
	}
}

func outputYAMLMarshal(v any) ([]byte, error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("marshaling to YAML: %w", err)}
	}
	return data, nil
}

func formatStatusColumn(status string, ready, total int) string {
	return fmt.Sprintf("%s (%d/%d)", output.FormatHealthStatus(status), ready, total)
}

func renderReleaseListTable(summaries []ReleaseSummary, allNamespaces, wide bool) {
	var headers []string
	switch {
	case allNamespaces && wide:
		headers = []string{"NAMESPACE", "NAME", "MODULE", "VERSION", "STATUS", "AGE", "RELEASE-ID", "LAST-APPLIED"}
	case allNamespaces:
		headers = []string{"NAMESPACE", "NAME", "MODULE", "VERSION", "STATUS", "AGE"}
	case wide:
		headers = []string{"NAME", "MODULE", "VERSION", "STATUS", "AGE", "RELEASE-ID", "LAST-APPLIED"}
	default:
		headers = []string{"NAME", "MODULE", "VERSION", "STATUS", "AGE"}
	}

	tbl := output.NewTable(headers...)
	for i := range summaries {
		s := &summaries[i]
		status := formatStatusColumn(s.Status, s.ReadyCount, s.TotalCount)
		switch {
		case allNamespaces && wide:
			tbl.Row(s.Namespace, s.Name, s.Module, s.Version, status, s.Age, s.ReleaseID, s.LastApplied)
		case allNamespaces:
			tbl.Row(s.Namespace, s.Name, s.Module, s.Version, status, s.Age)
		case wide:
			tbl.Row(s.Name, s.Module, s.Version, status, s.Age, s.ReleaseID, s.LastApplied)
		default:
			tbl.Row(s.Name, s.Module, s.Version, status, s.Age)
		}
	}
	output.Println(tbl.String())
}
