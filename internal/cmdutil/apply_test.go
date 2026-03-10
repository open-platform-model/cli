package cmdutil

import (
	"testing"

	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestCurrentInventoryEntries(t *testing.T) {
	resources := []*unstructured.Unstructured{{Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]any{
			"name":      "demo",
			"namespace": "apps",
		},
	}}}

	entries := CurrentInventoryEntries(resources)
	require.Len(t, entries, 1)
	assert.Equal(t, "ConfigMap", entries[0].Kind)
	assert.Equal(t, "demo", entries[0].Name)
	assert.Equal(t, "apps", entries[0].Namespace)
}

func TestPreviousInventoryEntries(t *testing.T) {
	prevInventory := &inventory.InventorySecret{
		Index: []string{"change-1"},
		Changes: map[string]*inventory.ChangeEntry{
			"change-1": {Inventory: inventory.InventoryList{Entries: []inventory.InventoryEntry{{Kind: "Service", Name: "web"}}}},
		},
	}

	entries := PreviousInventoryEntries(prevInventory)
	require.Len(t, entries, 1)
	assert.Equal(t, "Service", entries[0].Kind)
	assert.Equal(t, "web", entries[0].Name)
}

func TestGuardEmptyRender(t *testing.T) {
	releaseLog := output.ReleaseLogger("test")
	err := GuardEmptyRender(0, []inventory.InventoryEntry{{Kind: "ConfigMap"}}, false, releaseLog)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render produced 0 resources")
}
