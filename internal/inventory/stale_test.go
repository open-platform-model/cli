package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// helper to build an InventoryEntry quickly
func entry(group, kind, ns, name, component string) InventoryEntry {
	return InventoryEntry{
		Group:     group,
		Kind:      kind,
		Namespace: ns,
		Name:      name,
		Version:   "v1",
		Component: component,
	}
}

// --- ComputeStaleSet ---

func TestComputeStaleSet_ResourceRemoved(t *testing.T) {
	prev := []InventoryEntry{
		entry("apps", "Deployment", "ns", "app-a", "web"),
		entry("apps", "Deployment", "ns", "app-b", "web"),
		entry("", "Service", "ns", "svc-a", "web"),
	}
	cur := []InventoryEntry{
		entry("apps", "Deployment", "ns", "app-a", "web"),
		entry("", "Service", "ns", "svc-a", "web"),
	}

	stale := ComputeStaleSet(prev, cur)
	assert.Len(t, stale, 1)
	assert.Equal(t, "app-b", stale[0].Name)
}

func TestComputeStaleSet_ResourceRenamed(t *testing.T) {
	prev := []InventoryEntry{
		entry("", "Service", "ns", "old-name", "web"),
	}
	cur := []InventoryEntry{
		entry("", "Service", "ns", "new-name", "web"),
	}

	stale := ComputeStaleSet(prev, cur)
	assert.Len(t, stale, 1)
	assert.Equal(t, "old-name", stale[0].Name, "old-name should be in stale set")

	// new-name should not be in stale set
	for _, s := range stale {
		assert.NotEqual(t, "new-name", s.Name)
	}
}

func TestComputeStaleSet_FirstTimeApply_EmptyStale(t *testing.T) {
	stale := ComputeStaleSet(nil, []InventoryEntry{
		entry("apps", "Deployment", "ns", "app", "web"),
	})
	assert.Empty(t, stale)
}

func TestComputeStaleSet_EmptyPrevious_EmptyStale(t *testing.T) {
	stale := ComputeStaleSet([]InventoryEntry{}, []InventoryEntry{
		entry("apps", "Deployment", "ns", "app", "web"),
	})
	assert.Empty(t, stale)
}

func TestComputeStaleSet_IdempotentReapply_EmptyStale(t *testing.T) {
	entries := []InventoryEntry{
		entry("apps", "Deployment", "ns", "app", "web"),
		entry("", "Service", "ns", "svc", "web"),
	}
	stale := ComputeStaleSet(entries, entries)
	assert.Empty(t, stale)
}

func TestComputeStaleSet_VersionChangedSameIdentity_NotStale(t *testing.T) {
	// Version change alone should NOT make an entry stale (version excluded from identity)
	prev := []InventoryEntry{
		{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "web"},
	}
	cur := []InventoryEntry{
		{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v2", Component: "web"},
	}
	stale := ComputeStaleSet(prev, cur)
	assert.Empty(t, stale, "version change should not create stale entry (version excluded from identity)")
}

// --- ApplyComponentRenameSafetyCheck ---

func TestComponentRenameSafetyCheck_FiltersRename(t *testing.T) {
	// Deployment/my-app was under component "web", now under "frontend"
	stale := []InventoryEntry{
		entry("apps", "Deployment", "ns", "my-app", "web"),
	}
	current := []InventoryEntry{
		entry("apps", "Deployment", "ns", "my-app", "frontend"), // same K8s resource, different component
	}

	result := ApplyComponentRenameSafetyCheck(stale, current)
	assert.Empty(t, result, "component rename should remove entry from stale set")
}

func TestComponentRenameSafetyCheck_GenuineRemovalKept(t *testing.T) {
	// Deployment/old-app removed entirely (not renamed)
	stale := []InventoryEntry{
		entry("apps", "Deployment", "ns", "old-app", "web"),
	}
	current := []InventoryEntry{
		entry("apps", "Deployment", "ns", "new-app", "web"),
	}

	result := ApplyComponentRenameSafetyCheck(stale, current)
	assert.Len(t, result, 1)
	assert.Equal(t, "old-app", result[0].Name, "genuine removal should remain in stale set")
}

func TestComponentRenameSafetyCheck_MultipleEntries_MixedResult(t *testing.T) {
	stale := []InventoryEntry{
		entry("apps", "Deployment", "ns", "renamed-app", "old-comp"), // component rename — filter out
		entry("apps", "Deployment", "ns", "removed-app", "web"),      // genuinely removed — keep
	}
	current := []InventoryEntry{
		entry("apps", "Deployment", "ns", "renamed-app", "new-comp"), // same K8s resource, new component
		entry("", "Service", "ns", "some-svc", "web"),
	}

	result := ApplyComponentRenameSafetyCheck(stale, current)
	assert.Len(t, result, 1)
	assert.Equal(t, "removed-app", result[0].Name)
}

func TestComponentRenameSafetyCheck_EmptyStale_NoOp(t *testing.T) {
	result := ApplyComponentRenameSafetyCheck([]InventoryEntry{}, []InventoryEntry{
		entry("apps", "Deployment", "ns", "app", "web"),
	})
	assert.Empty(t, result)
}

func TestComponentRenameSafetyCheck_SameComponent_NotFiltered(t *testing.T) {
	// Same K8s resource, same component → this is a genuine change, not a rename
	stale := []InventoryEntry{
		entry("apps", "Deployment", "ns", "my-app", "web"),
	}
	current := []InventoryEntry{
		entry("apps", "Deployment", "ns", "my-app", "web"), // same everything
	}

	// This case shouldn't normally arise (stale = prev - cur removes it)
	// but if it does, the safety check should keep it (component is same)
	result := ApplyComponentRenameSafetyCheck(stale, current)
	// The rename check requires component to DIFFER — same component is not a rename
	assert.Len(t, result, 1, "same component is not a rename, should not be filtered")
}

// --- kindToResource ---

func TestKindToResource(t *testing.T) {
	cases := []struct {
		kind     string
		resource string
	}{
		{"Deployment", "deployments"},
		{"Service", "services"},
		{"ConfigMap", "configmaps"},
		{"Endpoints", "endpoints"},
		{"ClusterRole", "clusterroles"},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			assert.Equal(t, tc.resource, kindToResource(tc.kind))
		})
	}
}
