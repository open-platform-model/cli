package query

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	"gopkg.in/yaml.v3"
)

type InstanceSummary struct {
	Name        string `json:"name" yaml:"name"`
	Module      string `json:"module" yaml:"module"`
	Namespace   string `json:"namespace" yaml:"namespace"`
	Version     string `json:"version" yaml:"version"`
	Status      string `json:"status" yaml:"status"`
	ReadyCount  int    `json:"readyCount" yaml:"readyCount"`
	TotalCount  int    `json:"totalCount" yaml:"totalCount"`
	InstanceID  string `json:"instanceID" yaml:"instanceID"`
	LastApplied string `json:"lastApplied" yaml:"lastApplied"`
	Age         string `json:"age" yaml:"age"`
	Owner       string `json:"owner" yaml:"owner"`
}

type instanceHealthResult struct {
	index  int
	status kubernetes.HealthStatus
	ready  int
	total  int
}

func EvaluateInstanceHealth(ctx context.Context, client *kubernetes.Client, inventories []*inventory.InstanceInventoryRecord, concurrency int, logDiscoveryFailures bool) []InstanceSummary {
	summaries := make([]InstanceSummary, len(inventories))
	for i, inv := range inventories {
		summaries[i] = BuildInstanceSummary(inv)
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	results := make([]instanceHealthResult, len(inventories))

	for i, inv := range inventories {
		wg.Add(1)
		go func(idx int, inv *inventory.InstanceInventoryRecord) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			live, missing, err := inventory.DiscoverResourcesFromInventory(ctx, client, inv)
			if err != nil {
				if logDiscoveryFailures {
					output.Debug("failed to discover resources for instance", "instance", inv.InstanceMetadata.InstanceName, "error", err)
				}
				results[idx] = instanceHealthResult{index: idx, status: kubernetes.HealthUnknown}
				return
			}

			status, ready, total := kubernetes.QuickInstanceHealth(live, len(missing))
			results[idx] = instanceHealthResult{index: idx, status: status, ready: ready, total: total}
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

func BuildInstanceSummary(inv *inventory.InstanceInventoryRecord) InstanceSummary {
	s := InstanceSummary{
		Name:       inv.InstanceMetadata.InstanceName,
		Module:     inv.ModuleMetadata.Name,
		Namespace:  inv.InstanceMetadata.InstanceNamespace,
		InstanceID: inv.InstanceMetadata.InstanceID,
		Owner:      string(inventory.NormalizeCreatedBy(inv.CreatedBy)),
	}
	if inv.ModuleMetadata.Version != "" {
		s.Version = inv.ModuleMetadata.Version
	}
	s.LastApplied = inv.InstanceMetadata.LastTransitionTime
	if inv.InstanceMetadata.LastTransitionTime != "" {
		if t, err := time.Parse(time.RFC3339, inv.InstanceMetadata.LastTransitionTime); err == nil {
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

func RenderInstanceListOutput(summaries []InstanceSummary, format output.Format, allNamespaces bool) error {
	switch format { //nolint:exhaustive // output.ParseFormat constrains values before this switch
	case output.FormatJSON:
		data, err := json.MarshalIndent(summaries, "", "  ")
		if err != nil {
			return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("marshaling to JSON: %w", err)}
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
		renderInstanceListTable(summaries, allNamespaces, true)
		return nil
	default:
		renderInstanceListTable(summaries, allNamespaces, false)
		return nil
	}
}

func outputYAMLMarshal(v any) ([]byte, error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("marshaling to YAML: %w", err)}
	}
	return data, nil
}

func formatStatusColumn(status string, ready, total int) string {
	return fmt.Sprintf("%s (%d/%d)", output.FormatHealthStatus(status), ready, total)
}

func renderInstanceListTable(summaries []InstanceSummary, allNamespaces, wide bool) {
	// Compose headers from shared parts to avoid duplicated column literals.
	headers := []string{"NAME", "MODULE", "OWNER", "VERSION", "STATUS", "AGE"}
	if wide {
		headers = append(headers, "INSTANCE-ID", "LAST-APPLIED")
	}
	if allNamespaces {
		headers = append([]string{"NAMESPACE"}, headers...)
	}

	tbl := output.NewTable(headers...)
	for i := range summaries {
		s := &summaries[i]
		status := formatStatusColumn(s.Status, s.ReadyCount, s.TotalCount)
		switch {
		case allNamespaces && wide:
			tbl.Row(s.Namespace, s.Name, s.Module, s.Owner, s.Version, status, s.Age, s.InstanceID, s.LastApplied)
		case allNamespaces:
			tbl.Row(s.Namespace, s.Name, s.Module, s.Owner, s.Version, status, s.Age)
		case wide:
			tbl.Row(s.Name, s.Module, s.Owner, s.Version, status, s.Age, s.InstanceID, s.LastApplied)
		default:
			tbl.Row(s.Name, s.Module, s.Owner, s.Version, status, s.Age)
		}
	}
	output.Println(tbl.String())
}
