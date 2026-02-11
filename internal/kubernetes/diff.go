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

	"github.com/opmodel/cli/internal/build"
)

// resourceState represents the state of a resource in a diff comparison.
type resourceState string

const (
	// ResourceModified means the resource exists both locally and on the cluster with differences.
	ResourceModified resourceState = "modified"
	// ResourceAdded means the resource exists locally but not on the cluster.
	ResourceAdded resourceState = "added"
	// ResourceOrphaned means the resource exists on the cluster but not in the local render.
	ResourceOrphaned resourceState = "orphaned"
	// resourceUnchanged means the resource exists both locally and on the cluster with no differences.
	resourceUnchanged resourceState = "unchanged"
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
	State resourceState
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

// FetchLiveState fetches a single resource from the cluster.
func fetchLiveState(ctx context.Context, client *Client, resource *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	gvr := gvrFromUnstructured(resource)
	ns := resource.GetNamespace()
	name := resource.GetName()

	var live *unstructured.Unstructured
	var err error

	if ns != "" {
		live, err = client.Dynamic.Resource(gvr).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
	} else {
		live, err = client.Dynamic.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	}

	if err != nil {
		return nil, err
	}

	return live, nil
}

// resourceKey generates a unique key for a resource based on GVK, namespace, and name.
func resourceKey(gvk schema.GroupVersionKind, namespace, name string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", gvk.Group, gvk.Version, gvk.Kind, namespace, name)
}

// Diff compares rendered resources against the live cluster state and returns categorized results.
func Diff(ctx context.Context, client *Client, resources []*build.Resource, meta build.ModuleMetadata, comparer comparer) (*DiffResult, error) {
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
				State:     resourceUnchanged,
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
	orphans, err := findOrphans(ctx, client, meta, renderedKeys)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("detecting orphans: %v", err))
	} else {
		for _, orphan := range orphans {
			result.Resources = append(result.Resources, resourceDiff{
				Kind:      orphan.GetKind(),
				Name:      orphan.GetName(),
				Namespace: orphan.GetNamespace(),
				State:     ResourceOrphaned,
			})
			result.Orphaned++
		}
	}

	return result, nil
}

// DiffPartial compares rendered resources against live state, handling partial render results.
// Successfully rendered resources are compared; render errors produce warnings.
func DiffPartial(ctx context.Context, client *Client, resources []*build.Resource, renderErrors []error, meta build.ModuleMetadata, comparer comparer) (*DiffResult, error) {
	result, err := Diff(ctx, client, resources, meta, comparer)
	if err != nil {
		return nil, err
	}

	// Add render errors as warnings
	for _, renderErr := range renderErrors {
		result.Warnings = append(result.Warnings, fmt.Sprintf("render error: %v", renderErr))
	}

	return result, nil
}

// findOrphans discovers resources on the cluster that are labeled for this module
// but are not in the rendered resource set.
func findOrphans(ctx context.Context, client *Client, meta build.ModuleMetadata, renderedKeys map[string]bool) ([]*unstructured.Unstructured, error) {
	// Discover all resources on the cluster with OPM labels for this module
	// Use module name selector for orphan detection.
	// Diff always has a module name from the build pipeline.
	liveResources, err := DiscoverResources(ctx, client, DiscoveryOptions{
		ModuleName: meta.Name,
		Namespace:  meta.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("discovering live resources: %w", err)
	}

	var orphans []*unstructured.Unstructured
	for _, live := range liveResources {
		key := resourceKey(live.GroupVersionKind(), live.GetNamespace(), live.GetName())
		if !renderedKeys[key] {
			orphans = append(orphans, live)
		}
	}

	return orphans, nil
}
