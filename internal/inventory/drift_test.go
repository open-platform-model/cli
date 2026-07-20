package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func driftEntry(kind, name string) InventoryEntry {
	return InventoryEntry{Group: "apps", Version: "v1", Kind: kind, Namespace: "demo", Name: name, Component: "web"}
}

func TestDescribeEntrySetDrift_EqualSetsRegardlessOfOrder(t *testing.T) {
	before := []InventoryEntry{driftEntry("Deployment", "a"), driftEntry("Service", "b")}
	after := []InventoryEntry{driftEntry("Service", "b"), driftEntry("Deployment", "a")}

	assert.Empty(t, DescribeEntrySetDrift(before, after),
		"entry order is not an observable — the two actors have no reason to agree on it")
}

func TestDescribeEntrySetDrift_BothEmpty(t *testing.T) {
	assert.Empty(t, DescribeEntrySetDrift(nil, nil))
}

func TestDescribeEntrySetDrift_ReportsRemoved(t *testing.T) {
	before := []InventoryEntry{driftEntry("Deployment", "a"), driftEntry("Service", "b")}
	after := []InventoryEntry{driftEntry("Deployment", "a")}

	drift := DescribeEntrySetDrift(before, after)
	assert.Contains(t, drift, "no longer tracked")
	assert.Contains(t, drift, "Service/demo/b")
}

func TestDescribeEntrySetDrift_ReportsAdded(t *testing.T) {
	before := []InventoryEntry{driftEntry("Deployment", "a")}
	after := []InventoryEntry{driftEntry("Deployment", "a"), driftEntry("Ingress", "c")}

	drift := DescribeEntrySetDrift(before, after)
	assert.Contains(t, drift, "newly tracked")
	assert.Contains(t, drift, "Ingress/demo/c")
}

func TestDescribeEntrySetDrift_ReportsBothDirections(t *testing.T) {
	before := []InventoryEntry{driftEntry("Deployment", "a"), driftEntry("Service", "b")}
	after := []InventoryEntry{driftEntry("Deployment", "a"), driftEntry("Ingress", "c")}

	drift := DescribeEntrySetDrift(before, after)
	assert.Contains(t, drift, "Service/demo/b")
	assert.Contains(t, drift, "Ingress/demo/c")
}

// Identity, not content, is what the handoff verdict compares: the operator
// relabels every resource on adoption, so a content-sensitive comparison would
// fail on every successful handoff (enhancement 0006 D40).
func TestDescribeEntrySetDrift_IgnoresNonIdentityFields(t *testing.T) {
	before := []InventoryEntry{{Group: "apps", Version: "v1", Kind: "Deployment", Namespace: "demo", Name: "a", Component: "web"}}
	after := []InventoryEntry{{Group: "apps", Version: "v1beta1", Kind: "Deployment", Namespace: "demo", Name: "a", Component: "web"}}

	assert.Empty(t, DescribeEntrySetDrift(before, after))
}

func TestDescribeEntry_ClusterScoped(t *testing.T) {
	assert.Equal(t, "ClusterRole/admin", DescribeEntry(InventoryEntry{Kind: "ClusterRole", Name: "admin"}))
}
