package inventory

import (
	"context"
	"fmt"
	"sort"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/opmodel/cli/internal/core"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
)

// ComputeStaleSet computes the set of resources that were in the previous inventory
// but are not in the current inventory (set difference: previous - current).
// Identity comparison uses IdentityEqual (Group + Kind + Namespace + Name + Component).
//
// Returns an empty slice when previous is nil or empty (first-time apply).
func ComputeStaleSet(previous, current []InventoryEntry) []InventoryEntry {
	if len(previous) == 0 {
		return []InventoryEntry{}
	}

	var stale []InventoryEntry
	for _, prev := range previous {
		found := false
		for _, cur := range current {
			if IdentityEqual(prev, cur) {
				found = true
				break
			}
		}
		if !found {
			stale = append(stale, prev)
		}
	}

	if stale == nil {
		return []InventoryEntry{}
	}
	return stale
}

// ApplyComponentRenameSafetyCheck filters the stale set to remove entries
// where the current set contains the same K8s resource (same Group, Kind, Namespace, Name)
// but under a different Component name.
//
// This prevents a component rename from triggering destructive deletion of resources
// that are still desired — they have simply moved to a different component.
func ApplyComponentRenameSafetyCheck(stale, current []InventoryEntry) []InventoryEntry {
	if len(stale) == 0 {
		return stale
	}

	filtered := make([]InventoryEntry, 0, len(stale))
	for _, s := range stale {
		isRename := false
		for _, c := range current {
			if K8sIdentityEqual(s, c) && s.Component != c.Component {
				isRename = true
				output.Debug("component rename detected, skipping prune",
					"group", s.Group, "kind", s.Kind, "namespace", s.Namespace, "name", s.Name,
					"oldComponent", s.Component, "newComponent", c.Component,
				)
				break
			}
		}
		if !isRename {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// PreApplyExistenceCheck verifies that resources do not conflict with existing
// cluster state on a first-time apply (no previous inventory).
//
// For each rendered resource entry, a GET is performed:
//   - If the resource exists with a deletionTimestamp → error (terminating)
//   - If the resource exists without OPM managed-by label → error (untracked)
//   - If the resource does not exist → OK
//
// This check should be skipped entirely when a previous inventory exists.
func PreApplyExistenceCheck(ctx context.Context, client *kubernetes.Client, entries []InventoryEntry) error {
	for _, entry := range entries {
		gvr := schema.GroupVersionResource{
			Group:    entry.Group,
			Version:  entry.Version,
			Resource: kindToResource(entry.Kind),
		}

		var obj interface{ GetDeletionTimestamp() *metav1.Time }
		unstrObj, err := client.ResourceClient(gvr, entry.Namespace).Get(ctx, entry.Name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue // Resource doesn't exist — OK for first install
			}
			// Other errors (RBAC, etc.) — warn but don't fail
			output.Debug("could not check resource existence (skipping)",
				"kind", entry.Kind, "name", entry.Name, "err", err)
			continue
		}

		obj = unstrObj

		// Check for terminating resources
		if obj.GetDeletionTimestamp() != nil {
			return fmt.Errorf("resource %s/%s in namespace %q is terminating (deletionTimestamp set) — wait for deletion to complete before applying",
				entry.Kind, entry.Name, entry.Namespace)
		}

		// Check for untracked resources (not managed by OPM)
		labels := unstrObj.GetLabels()
		if labels[core.LabelManagedBy] != core.LabelManagedByValue {
			return fmt.Errorf("resource %s/%s in namespace %q already exists and is not managed by OPM — use --force to proceed",
				entry.Kind, entry.Name, entry.Namespace)
		}
	}
	return nil
}

// PruneStaleResources deletes the stale resources from the cluster.
// Resources are deleted in reverse weight order (highest weight first).
// Namespace resources are excluded unless explicitly included.
// 404 (not found) errors are treated as success (idempotent).
func PruneStaleResources(ctx context.Context, client *kubernetes.Client, stale []InventoryEntry) error {
	if len(stale) == 0 {
		return nil
	}

	// Sort in reverse weight order (highest weight deleted first)
	sorted := make([]InventoryEntry, len(stale))
	copy(sorted, stale)
	sort.SliceStable(sorted, func(i, j int) bool {
		wi := core.GetWeight(schema.GroupVersionKind{Group: sorted[i].Group, Version: sorted[i].Version, Kind: sorted[i].Kind})
		wj := core.GetWeight(schema.GroupVersionKind{Group: sorted[j].Group, Version: sorted[j].Version, Kind: sorted[j].Kind})
		return wi > wj // descending
	})

	var errs []error
	for _, entry := range sorted {
		// Exclude Namespace resources from pruning by default
		if entry.Kind == "Namespace" && entry.Group == "" {
			output.Debug("skipping Namespace pruning", "name", entry.Name)
			continue
		}

		gvr := schema.GroupVersionResource{
			Group:    entry.Group,
			Version:  entry.Version,
			Resource: kindToResource(entry.Kind),
		}

		propagation := metav1.DeletePropagationForeground
		err := client.ResourceClient(gvr, entry.Namespace).Delete(ctx, entry.Name, metav1.DeleteOptions{
			PropagationPolicy: &propagation,
		})

		if err != nil && !apierrors.IsNotFound(err) {
			output.Warn("failed to prune stale resource",
				"kind", entry.Kind, "name", entry.Name, "err", err)
			errs = append(errs, fmt.Errorf("deleting %s/%s: %w", entry.Kind, entry.Name, err))
			continue
		}

		output.Debug("pruned stale resource", "kind", entry.Kind, "namespace", entry.Namespace, "name", entry.Name)
	}

	if len(errs) > 0 {
		return fmt.Errorf("pruning stale resources: %d error(s): %v", len(errs), errs[0])
	}
	return nil
}

// kindToResource converts a Kind to its plural resource name for use in GVR.
// This is a best-effort heuristic for the most common cases.
// For accurate GVR resolution in production, use server-side discovery.
func kindToResource(kind string) string {
	switch kind {
	case "Endpoints":
		return "endpoints"
	case "Series":
		return "series"
	default:
		// Standard lowercase + "s" pluralization for most K8s resources
		lower := ""
		for _, c := range kind {
			if c >= 'A' && c <= 'Z' {
				lower += string(c + 32)
			} else {
				lower += string(c)
			}
		}
		return lower + "s"
	}
}
