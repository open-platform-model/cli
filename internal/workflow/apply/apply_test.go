package apply

import (
	"testing"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	pkginventory "github.com/open-platform-model/cli/pkg/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestCurrentInventoryEntries(t *testing.T) {
	resources := []*unstructured.Unstructured{{Object: map[string]any{"apiVersion": "v1", "kind": "ConfigMap", "metadata": map[string]any{"name": "demo", "namespace": "apps"}}}}
	entries := CurrentInventoryEntries(resources)
	require.Len(t, entries, 1)
	assert.Equal(t, "ConfigMap", entries[0].Kind)
	assert.Equal(t, "demo", entries[0].Name)
	assert.Equal(t, "apps", entries[0].Namespace)
}

func TestPreviousEntries_FromCRRecord(t *testing.T) {
	prev := &inventory.Record{Inventory: inventory.Inventory{Entries: []inventory.InventoryEntry{{Kind: "Service", Name: "web"}}}}
	entries := previousEntries(prev, nil)
	require.Len(t, entries, 1)
	assert.Equal(t, "Service", entries[0].Kind)
	assert.Equal(t, "web", entries[0].Name)
}

func TestPreviousEntries_FromMigrationSource(t *testing.T) {
	legacy := &inventory.LegacyInventory{
		Inventory: pkginventory.Inventory{Entries: []inventory.InventoryEntry{{Kind: "ConfigMap", Name: "cfg"}}},
	}
	entries := previousEntries(nil, legacy)
	require.Len(t, entries, 1)
	assert.Equal(t, "ConfigMap", entries[0].Kind)
}

func TestNextRevision(t *testing.T) {
	assert.Equal(t, 1, nextRevision(nil, nil))
	assert.Equal(t, 3, nextRevision(&inventory.Record{Inventory: inventory.Inventory{Revision: 2}}, nil))

	legacy := &inventory.LegacyInventory{Inventory: pkginventory.Inventory{Revision: 4}}
	assert.Equal(t, 5, nextRevision(nil, legacy))
}

func TestGuardEmptyRender(t *testing.T) {
	instanceLog := output.InstanceLogger("test")
	err := GuardEmptyRender(0, []inventory.InventoryEntry{{Kind: "ConfigMap"}}, false, instanceLog)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render produced 0 resources")
}

func TestSourceDigest(t *testing.T) {
	// Deterministic and reference-derived.
	a := sourceDigest("opmodel.dev/modules/podinfo@v0", "0.1.0")
	b := sourceDigest("opmodel.dev/modules/podinfo@v0", "0.1.0")
	assert.Equal(t, a, b)
	assert.Contains(t, a, "sha256:")
	// Different reference → different digest.
	assert.NotEqual(t, a, sourceDigest("opmodel.dev/modules/podinfo@v0", "0.2.0"))
	// Empty reference → empty digest.
	assert.Equal(t, "", sourceDigest("", ""))
}

func TestFormatApplySummary(t *testing.T) {
	summary := FormatApplySummary(&kubernetes.ApplyResult{Applied: 5, Created: 2, Configured: 1, Unchanged: 2})
	assert.Equal(t, "applied 5 resources successfully (2 created, 1 configured, 2 unchanged)", summary)
}

// Ownership refusal is unit-tested at the resolver (inventory.ResolveOwnership)
// and exercised end-to-end with a live CRD in the e2e gate tests; the full
// gate+ownership Execute path needs a seeded CRD/Platform/CR fixture that the
// e2e suite provides.
