package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/gonvenience/ytbx"
	"github.com/homeport/dyff/pkg/dyff"
	"gopkg.in/yaml.v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/opmodel/cli/internal/core"
	"github.com/opmodel/cli/internal/core/modulerelease"
	"github.com/opmodel/cli/internal/output"
)

// ResourceState represents the state of a resource in a diff comparison.
type ResourceState string

const (
	// ResourceModified means the resource exists both locally and on the cluster with differences.
	ResourceModified ResourceState = "modified"
	// ResourceAdded means the resource exists locally but not on the cluster.
	ResourceAdded ResourceState = "added"
	// ResourceOrphaned means the resource exists on the cluster but not in the local render.
	ResourceOrphaned ResourceState = "orphaned"
	// ResourceUnchanged means the resource exists both locally and on the cluster with no differences.
	ResourceUnchanged ResourceState = "unchanged"
)

// resourceDiff contains the diff details for a single resource.
type resourceDiff struct {
	// Kind is the Kubernetes resource kind.
	Kind string
	// Name is the resource name.
	Name string
	// Namespace is the resource namespace.
	Namespace string
	// State indicates whether the resource is modified, added, or orphaned.
	State ResourceState
	// Diff is the human-readable diff output (only for modified resources).
	Diff string
}

// DiffResult contains the full diff output.
type DiffResult struct {
	// Resources is the list of resource diffs.
	Resources []resourceDiff
	// Modified is the count of modified resources.
	Modified int
	// Added is the count of added resources.
	Added int
	// Orphaned is the count of orphaned resources.
	Orphaned int
	// Unchanged is the count of unchanged resources.
	Unchanged int
	// Warnings contains non-fatal warnings (e.g., from partial render).
	Warnings []string
}

// IsEmpty returns true if there are no differences.
func (r *DiffResult) IsEmpty() bool {
	return r.Modified == 0 && r.Added == 0 && r.Orphaned == 0
}

// SummaryLine returns a human-readable summary of the diff.
func (r *DiffResult) SummaryLine() string {
	if r.IsEmpty() {
		return "No differences found"
	}

	var parts []string
	if r.Modified > 0 {
		parts = append(parts, fmt.Sprintf("%d modified", r.Modified))
	}
	if r.Added > 0 {
		parts = append(parts, fmt.Sprintf("%d added", r.Added))
	}
	if r.Orphaned > 0 {
		parts = append(parts, fmt.Sprintf("%d orphaned", r.Orphaned))
	}
	return strings.Join(parts, ", ")
}

// comparer wraps the diff comparison logic. It uses dyff by default but
// can be replaced with a different implementation.
type comparer interface {
	// Compare compares two YAML documents and returns a human-readable diff.
	// Returns empty string if there are no differences.
	Compare(rendered, live *unstructured.Unstructured) (string, error)
}

// dyffComparer is the default comparer implementation using homeport/dyff.
type dyffComparer struct{}

// NewComparer creates a new comparer using dyff.
func NewComparer() comparer {
	return &dyffComparer{}
}

// Compare compares two unstructured resources using dyff for semantic YAML comparison.
func (c *dyffComparer) Compare(rendered, live *unstructured.Unstructured) (string, error) {
	// Marshal both to YAML for dyff comparison
	renderedYAML, err := yaml.Marshal(rendered.Object)
	if err != nil {
		return "", fmt.Errorf("marshaling rendered resource: %w", err)
	}

	liveYAML, err := yaml.Marshal(live.Object)
	if err != nil {
		return "", fmt.Errorf("marshaling live resource: %w", err)
	}

	// Parse YAML documents for dyff
	renderedDocs, err := ytbx.LoadDocuments(renderedYAML)
	if err != nil {
		return "", fmt.Errorf("parsing rendered YAML: %w", err)
	}

	liveDocs, err := ytbx.LoadDocuments(liveYAML)
	if err != nil {
		return "", fmt.Errorf("parsing live YAML: %w", err)
	}

	renderedInput := ytbx.InputFile{
		Location:  "rendered",
		Documents: renderedDocs,
	}

	liveInput := ytbx.InputFile{
		Location:  "cluster",
		Documents: liveDocs,
	}

	// Compare using dyff (live as "from", rendered as "to")
	report, err := dyff.CompareInputFiles(liveInput, renderedInput)
	if err != nil {
		return "", fmt.Errorf("comparing resources: %w", err)
	}

	// No differences
	if len(report.Diffs) == 0 {
		return "", nil
	}

	// Format the report
	var buf bytes.Buffer
	writer := &dyff.HumanReport{
		Report:     report,
		OmitHeader: true,
	}
	if err := writer.WriteReport(&buf); err != nil {
		return "", fmt.Errorf("formatting diff report: %w", err)
	}

	return buf.String(), nil
}

// stripServerManagedFields removes well-known server-only fields from a
// Kubernetes object map. These fields are never present in rendered output
// and are always noise in diff comparisons.
func stripServerManagedFields(obj map[string]interface{}) {
	// Remove top-level status block
	delete(obj, "status")

	// Remove server-managed metadata fields
	meta, ok := obj["metadata"].(map[string]interface{})
	if !ok {
		return
	}
	delete(meta, "managedFields")
	delete(meta, "uid")
	delete(meta, "resourceVersion")
	delete(meta, "creationTimestamp")
	delete(meta, "generation")
}

// projectLiveToRendered recursively walks the rendered object and retains
// only matching paths in a deep copy of the live object. Fields present in
// live but absent from rendered are server-managed noise and get stripped.
func projectLiveToRendered(rendered, live map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, renderedVal := range rendered {
		liveVal, exists := live[key]
		if !exists {
			// Key exists in rendered but not in live — keep rendered value
			// so dyff can show it as a diff (addition).
			result[key] = renderedVal
			continue
		}

		renderedMap, renderedIsMap := renderedVal.(map[string]interface{})
		liveMap, liveIsMap := liveVal.(map[string]interface{})

		if renderedIsMap && liveIsMap {
			// Both are maps — recurse
			projected := projectLiveToRendered(renderedMap, liveMap)
			// Keep the projected map if it has content, or if the rendered
			// map was empty (author explicitly declared an empty map — preserve
			// it so both sides match and dyff sees no spurious diff).
			if len(projected) > 0 || len(renderedMap) == 0 {
				result[key] = projected
			}
			continue
		}

		renderedSlice, renderedIsSlice := renderedVal.([]interface{})
		liveSlice, liveIsSlice := liveVal.([]interface{})

		if renderedIsSlice && liveIsSlice {
			result[key] = projectSlice(renderedSlice, liveSlice)
			continue
		}

		// Scalar or type mismatch — keep the live value
		result[key] = liveVal
	}

	return result
}

// projectSlice projects a live slice to match the structure of a rendered
// slice. For lists of maps, elements are matched by the "name" field;
// if no "name" field exists, index-based matching is used.
// For scalar lists, the live list is kept as-is.
func projectSlice(rendered, live []interface{}) []interface{} {
	if len(rendered) == 0 {
		return live
	}

	// Check if this is a list of maps
	if _, ok := rendered[0].(map[string]interface{}); !ok {
		// Scalar list — keep the live list as-is
		return live
	}

	// Build a lookup of live elements by name (if available)
	liveByName := make(map[string]map[string]interface{})
	for _, item := range live {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		name, ok := m["name"].(string)
		if ok {
			liveByName[name] = m
		}
	}

	result := make([]interface{}, 0, len(rendered))
	for i, renderedItem := range rendered {
		renderedMap, ok := renderedItem.(map[string]interface{})
		if !ok {
			// Non-map element in a mixed list — keep as-is from live by index
			if i < len(live) {
				result = append(result, live[i])
			} else {
				result = append(result, renderedItem)
			}
			continue
		}

		// Try matching by name first
		var liveMap map[string]interface{}
		if name, ok := renderedMap["name"].(string); ok {
			liveMap = liveByName[name]
		}

		// Fall back to index-based matching
		if liveMap == nil && i < len(live) {
			if m, ok := live[i].(map[string]interface{}); ok {
				liveMap = m
			}
		}

		if liveMap != nil {
			projected := projectLiveToRendered(renderedMap, liveMap)
			if len(projected) > 0 {
				result = append(result, projected)
			}
		} else {
			// No matching live element — keep rendered (will show as addition)
			result = append(result, renderedItem)
		}
	}

	return result
}

// FetchLiveState fetches a single resource from the cluster.
func fetchLiveState(ctx context.Context, client *Client, resource *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	gvr := gvrFromUnstructured(resource)
	ns := resource.GetNamespace()
	name := resource.GetName()

	return client.ResourceClient(gvr, ns).Get(ctx, name, metav1.GetOptions{})
}

// resourceKey generates a unique key for a resource based on GVK, namespace, and name.
func resourceKey(gvk schema.GroupVersionKind, namespace, name string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", gvk.Group, gvk.Version, gvk.Kind, namespace, name)
}

// DiffOptions configures a Diff operation.
type DiffOptions struct {
	// InventoryLive is the list of live resources pre-fetched from the inventory
	// Secret by the caller. Orphan detection uses set-difference against this list.
	// When nil, no live resources are known and no orphans are reported.
	InventoryLive []*unstructured.Unstructured
}

// Diff compares rendered resources against the live cluster state and returns categorized results.
func Diff(ctx context.Context, client *Client, resources []*core.Resource, meta modulerelease.ReleaseMetadata, comparer comparer, opts ...DiffOptions) (*DiffResult, error) {
	var diffOpts DiffOptions
	if len(opts) > 0 {
		diffOpts = opts[0]
	}

	result := &DiffResult{}

	// Build a set of rendered resource keys for orphan detection
	renderedKeys := make(map[string]bool)
	for _, res := range resources {
		key := resourceKey(res.Object.GroupVersionKind(), res.Object.GetNamespace(), res.Object.GetName())
		renderedKeys[key] = true
	}

	// Compare each rendered resource against live state
	for _, res := range resources {
		obj := res.Object
		kind := obj.GetKind()
		name := obj.GetName()
		ns := obj.GetNamespace()

		live, err := fetchLiveState(ctx, client, obj)
		if err != nil {
			// Resource not found on cluster -> added
			if apierrors.IsNotFound(err) {
				result.Resources = append(result.Resources, resourceDiff{
					Kind:      kind,
					Name:      name,
					Namespace: ns,
					State:     ResourceAdded,
				})
				result.Added++
				continue
			}
			// Other errors are warnings
			result.Warnings = append(result.Warnings, fmt.Sprintf("fetching %s/%s: %v", kind, name, err))
			continue
		}

		// Filter live object to only contain fields present in rendered output.
		// Two-layer filtering: strip server metadata, then project to rendered paths.
		stripServerManagedFields(live.Object)
		live.Object = projectLiveToRendered(obj.Object, live.Object)

		// Resource exists on both sides — compare
		diffOutput, err := comparer.Compare(obj, live)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("comparing %s/%s: %v", kind, name, err))
			continue
		}

		if diffOutput == "" {
			// No differences
			result.Resources = append(result.Resources, resourceDiff{
				Kind:      kind,
				Name:      name,
				Namespace: ns,
				State:     ResourceUnchanged,
			})
			result.Unchanged++
		} else {
			result.Resources = append(result.Resources, resourceDiff{
				Kind:      kind,
				Name:      name,
				Namespace: ns,
				State:     ResourceModified,
				Diff:      diffOutput,
			})
			result.Modified++
		}
	}

	// Detect orphaned resources (on cluster but not in local render)
	orphans := findOrphans(renderedKeys, diffOpts.InventoryLive)
	for _, orphan := range orphans {
		result.Resources = append(result.Resources, resourceDiff{
			Kind:      orphan.GetKind(),
			Name:      orphan.GetName(),
			Namespace: orphan.GetNamespace(),
			State:     ResourceOrphaned,
		})
		result.Orphaned++
	}

	return result, nil
}

// DiffPartial compares rendered resources against live state, handling partial render results.
// Successfully rendered resources are compared; render errors produce warnings.
func DiffPartial(ctx context.Context, client *Client, resources []*core.Resource, renderErrors []error, meta modulerelease.ReleaseMetadata, comparer comparer, opts ...DiffOptions) (*DiffResult, error) {
	result, err := Diff(ctx, client, resources, meta, comparer, opts...)
	if err != nil {
		return nil, err
	}

	// Add render errors as warnings
	for _, renderErr := range renderErrors {
		result.Warnings = append(result.Warnings, fmt.Sprintf("render error: %v", renderErr))
	}

	return result, nil
}

// findOrphans returns resources that exist in inventoryLive but are not present
// in the rendered resource set. When inventoryLive is nil the set is empty and
// no orphans are reported (first-time diff where no release has been deployed yet).
func findOrphans(renderedKeys map[string]bool, inventoryLive []*unstructured.Unstructured) []*unstructured.Unstructured {
	output.Debug("orphan detection from inventory", "liveCount", len(inventoryLive))

	var orphans []*unstructured.Unstructured
	for _, live := range inventoryLive {
		key := resourceKey(live.GroupVersionKind(), live.GetNamespace(), live.GetName())
		if !renderedKeys[key] {
			orphans = append(orphans, live)
		}
	}

	return orphans
}
