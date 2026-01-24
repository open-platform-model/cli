package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/gonvenience/ytbx"
	"github.com/homeport/dyff/pkg/dyff"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// DiffResult represents a diff between local and live resources.
type DiffResult struct {
	// HasChanges indicates if there are differences.
	HasChanges bool

	// Added resources (in local, not in cluster).
	Added []string

	// Removed resources (in cluster, not in local).
	Removed []string

	// Modified resources (different between local and cluster).
	Modified []ModifiedResource
}

// ModifiedResource represents a resource with changes.
type ModifiedResource struct {
	// Name is the resource identifier (kind/namespace/name).
	Name string

	// Diff is the rendered diff output.
	Diff string
}

// NewDiffResult creates a new empty DiffResult.
func NewDiffResult() *DiffResult {
	return &DiffResult{
		Added:    make([]string, 0),
		Removed:  make([]string, 0),
		Modified: make([]ModifiedResource, 0),
	}
}

// IsEmpty returns true if there are no changes.
func (r *DiffResult) IsEmpty() bool {
	return len(r.Added) == 0 && len(r.Removed) == 0 && len(r.Modified) == 0
}

// AddAdded records a resource that will be added.
func (r *DiffResult) AddAdded(name string) {
	r.Added = append(r.Added, name)
	r.HasChanges = true
}

// AddRemoved records a resource that will be removed.
func (r *DiffResult) AddRemoved(name string) {
	r.Removed = append(r.Removed, name)
	r.HasChanges = true
}

// AddModified records a resource with modifications.
func (r *DiffResult) AddModified(name, diff string) {
	r.Modified = append(r.Modified, ModifiedResource{
		Name: name,
		Diff: diff,
	})
	r.HasChanges = true
}

// Summary returns a summary string of changes.
func (r *DiffResult) Summary() string {
	if r.IsEmpty() {
		return "No changes"
	}

	parts := make([]string, 0, 3)
	if len(r.Added) > 0 {
		parts = append(parts, fmt.Sprintf("%d added", len(r.Added)))
	}
	if len(r.Removed) > 0 {
		parts = append(parts, fmt.Sprintf("%d removed", len(r.Removed)))
	}
	if len(r.Modified) > 0 {
		parts = append(parts, fmt.Sprintf("%d modified", len(r.Modified)))
	}

	return strings.Join(parts, ", ")
}

// DiffOptions configures the diff operation.
type DiffOptions struct {
	// Namespace to scope the comparison.
	Namespace string

	// UseColor enables colorized diff output.
	UseColor bool

	// ModuleName is the name of the module (for discovering removed resources).
	ModuleName string

	// ModuleNamespace is the namespace the module is deployed to.
	ModuleNamespace string
}

// Diff computes the difference between desired manifests and live cluster state.
func (c *Client) Diff(ctx context.Context, desired []*unstructured.Unstructured, opts DiffOptions) (*DiffResult, error) {
	result := NewDiffResult()

	// Build a map of desired resources by key for quick lookup
	desiredByKey := make(map[string]*unstructured.Unstructured)
	for _, obj := range desired {
		key := ResourceKey(obj)
		desiredByKey[key] = obj
	}

	// Compare each desired resource against live
	for _, desiredObj := range desired {
		key := ResourceKey(desiredObj)

		// Fetch the live resource
		liveObj, err := c.fetchLiveResource(ctx, desiredObj)
		if err != nil {
			if apierrors.IsNotFound(err) {
				// Resource doesn't exist - it will be added
				result.AddAdded(key)
				continue
			}
			// Other errors - propagate
			return nil, fmt.Errorf("fetching live resource %s: %w", key, err)
		}

		// Compare live vs desired
		diff, err := compareResources(liveObj, desiredObj, opts.UseColor)
		if err != nil {
			return nil, fmt.Errorf("comparing %s: %w", key, err)
		}

		if diff != "" {
			result.AddModified(key, diff)
		}
	}

	// Find removed resources (only if module info is provided)
	if opts.ModuleName != "" && opts.ModuleNamespace != "" {
		liveResources, err := c.DiscoverModuleResources(ctx, opts.ModuleName, opts.ModuleNamespace)
		if err != nil {
			// Log warning but don't fail - we can still show add/modify diffs
			// The discover may fail due to permissions on some resource types
		} else {
			for _, liveObj := range liveResources {
				key := ResourceKey(liveObj)
				if _, exists := desiredByKey[key]; !exists {
					result.AddRemoved(key)
				}
			}
		}
	}

	return result, nil
}

// fetchLiveResource fetches a single resource from the cluster.
func (c *Client) fetchLiveResource(ctx context.Context, desired *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	gvk := desired.GroupVersionKind()
	mapping, err := c.Mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("getting REST mapping: %w", err)
	}

	getOpts := metav1.GetOptions{}

	var live *unstructured.Unstructured
	if desired.GetNamespace() != "" {
		live, err = c.Dynamic.Resource(mapping.Resource).
			Namespace(desired.GetNamespace()).
			Get(ctx, desired.GetName(), getOpts)
	} else {
		live, err = c.Dynamic.Resource(mapping.Resource).
			Get(ctx, desired.GetName(), getOpts)
	}

	if err != nil {
		return nil, err
	}

	return live, nil
}

// compareResources compares a live and desired resource and returns a diff string.
// Returns empty string if no differences.
func compareResources(live, desired *unstructured.Unstructured, useColor bool) (string, error) {
	// Serialize both for comparison, stripping managed fields
	liveYAML, err := serializeForDiff(live)
	if err != nil {
		return "", fmt.Errorf("serializing live resource: %w", err)
	}

	desiredYAML, err := serializeForDiff(desired)
	if err != nil {
		return "", fmt.Errorf("serializing desired resource: %w", err)
	}

	// Use dyff for YAML-aware comparison
	return diffYAMLWithColor(liveYAML, desiredYAML, useColor)
}

// ResourceKey generates a unique key for a resource (kind/namespace/name).
func ResourceKey(obj *unstructured.Unstructured) string {
	return fmt.Sprintf("%s/%s/%s", obj.GetKind(), obj.GetNamespace(), obj.GetName())
}

// serializeForDiff converts an unstructured object to YAML bytes,
// removing server-managed fields that shouldn't be compared.
func serializeForDiff(obj *unstructured.Unstructured) ([]byte, error) {
	// Deep copy to avoid modifying the original
	copy := obj.DeepCopy()

	// Strip server-managed fields
	stripManagedFields(copy.Object)

	// Serialize to YAML
	return yaml.Marshal(copy.Object)
}

// stripManagedFields removes fields that are server-managed and shouldn't affect diff.
func stripManagedFields(obj map[string]interface{}) {
	// Remove status section entirely
	delete(obj, "status")

	// Clean up metadata
	if metadata, ok := obj["metadata"].(map[string]interface{}); ok {
		// Remove server-managed metadata fields
		delete(metadata, "resourceVersion")
		delete(metadata, "uid")
		delete(metadata, "creationTimestamp")
		delete(metadata, "generation")
		delete(metadata, "managedFields")
		delete(metadata, "selfLink")

		// Remove empty annotations map if present
		if annotations, ok := metadata["annotations"].(map[string]interface{}); ok {
			// Remove common server-added annotations
			delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
			if len(annotations) == 0 {
				delete(metadata, "annotations")
			}
		}
	}
}

// diffYAMLWithColor computes a colorized YAML diff using dyff.
func diffYAMLWithColor(live, desired []byte, useColor bool) (string, error) {
	// Handle empty inputs
	if len(live) == 0 && len(desired) == 0 {
		return "", nil
	}

	// Parse YAML documents
	liveInput, err := parseYAMLInput("live", live)
	if err != nil {
		return "", fmt.Errorf("parsing live YAML: %w", err)
	}

	desiredInput, err := parseYAMLInput("desired", desired)
	if err != nil {
		return "", fmt.Errorf("parsing desired YAML: %w", err)
	}

	// Compare the inputs
	report, err := dyff.CompareInputFiles(liveInput, desiredInput)
	if err != nil {
		return "", fmt.Errorf("comparing YAML: %w", err)
	}

	// If no differences, return empty string
	if len(report.Diffs) == 0 {
		return "", nil
	}

	// Render the report
	return renderDyffReport(report, useColor)
}

// parseYAMLInput parses YAML bytes into a dyff input file.
func parseYAMLInput(name string, data []byte) (ytbx.InputFile, error) {
	if len(data) == 0 {
		// Return empty input file for empty data
		return ytbx.InputFile{
			Location:  name,
			Documents: nil,
		}, nil
	}

	// Trim whitespace
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return ytbx.InputFile{
			Location:  name,
			Documents: nil,
		}, nil
	}

	docs, err := ytbx.LoadYAMLDocuments(data)
	if err != nil {
		return ytbx.InputFile{}, err
	}

	return ytbx.InputFile{
		Location:  name,
		Documents: docs,
	}, nil
}

// renderDyffReport renders a dyff report to a string.
func renderDyffReport(report dyff.Report, useColor bool) (string, error) {
	var buf bytes.Buffer

	reportWriter := &dyff.HumanReport{
		Report:            report,
		DoNotInspectCerts: true,
		NoTableStyle:      !useColor,
		OmitHeader:        true,
	}

	if err := reportWriter.WriteReport(io.Writer(&buf)); err != nil {
		return "", fmt.Errorf("writing report: %w", err)
	}

	result := buf.String()

	// Clean up output - remove trailing whitespace from lines
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}

	return strings.TrimSpace(strings.Join(lines, "\n")), nil
}
