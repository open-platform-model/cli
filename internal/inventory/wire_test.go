package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkginventory "github.com/open-platform-model/cli/pkg/inventory"
)

func TestEntryToWire_VersionSerializesAsV(t *testing.T) {
	e := pkginventory.InventoryEntry{
		Group:     "apps",
		Kind:      "Deployment",
		Namespace: "demo",
		Name:      "podinfo",
		Version:   "v1",
		Component: "web",
	}

	m := entryToWire(e)

	assert.Equal(t, "v1", m["v"], "Version must serialize under the CRD key %q", "v")
	assert.NotContains(t, m, "version", "the Go tag key must not leak into the wire shape")
	assert.Equal(t, "Deployment", m["kind"])
	assert.Equal(t, "podinfo", m["name"])
	assert.Equal(t, "apps", m["group"])
	assert.Equal(t, "demo", m["namespace"])
	assert.Equal(t, "web", m["component"])
}

func TestEntryToWire_OmitsEmptyOptionalFields(t *testing.T) {
	// Core-group resource with no component: group, v, component omitted.
	e := pkginventory.InventoryEntry{
		Kind:      "ConfigMap",
		Namespace: "demo",
		Name:      "settings",
	}

	m := entryToWire(e)

	assert.NotContains(t, m, "group")
	assert.NotContains(t, m, "v")
	assert.NotContains(t, m, "component")
	assert.Equal(t, "ConfigMap", m["kind"])
	assert.Equal(t, "settings", m["name"])
	assert.Equal(t, "demo", m["namespace"])
}

func TestEntryWireRoundTrip(t *testing.T) {
	cases := []pkginventory.InventoryEntry{
		{Group: "apps", Kind: "Deployment", Namespace: "demo", Name: "podinfo", Version: "v1", Component: "web"},
		{Kind: "ConfigMap", Namespace: "demo", Name: "settings"},
		{Group: "networking.k8s.io", Kind: "Ingress", Namespace: "demo", Name: "podinfo", Version: "v1"},
		{Kind: "Namespace", Name: "demo"},
	}
	for _, e := range cases {
		got := entryFromWire(entryToWire(e))
		assert.Equal(t, e, got, "entry must round-trip losslessly")
	}
}

func TestInventoryWireRoundTrip(t *testing.T) {
	inv := pkginventory.Inventory{
		Revision: 3,
		Digest:   "sha256:deadbeef",
		Count:    2,
		Entries: []pkginventory.InventoryEntry{
			{Group: "apps", Kind: "Deployment", Namespace: "demo", Name: "podinfo", Version: "v1", Component: "web"},
			{Kind: "ConfigMap", Namespace: "demo", Name: "settings", Component: "web"},
		},
	}

	got := inventoryFromWire(inventoryToWire(inv))

	assert.Equal(t, inv, got)
}

func TestInventoryToWire_UsesInt64Counters(t *testing.T) {
	inv := pkginventory.Inventory{Revision: 5, Count: 7, Entries: nil}

	m := inventoryToWire(inv)

	// The unstructured converter accepts only int64 for integers.
	assert.IsType(t, int64(0), m["revision"])
	assert.IsType(t, int64(0), m["count"])
	entries, ok := m["entries"].([]any)
	require.True(t, ok, "entries must be []any for the unstructured converter")
	assert.Empty(t, entries)
}

func TestInventoryFromWire_ToleratesFloat64Counters(t *testing.T) {
	// A JSON decode (rather than the typed unstructured path) surfaces integers
	// as float64; the reader must tolerate both.
	m := map[string]any{
		"revision": float64(4),
		"count":    float64(1),
		"digest":   "sha256:abc",
		"entries": []any{
			map[string]any{"kind": "ConfigMap", "name": "settings", "namespace": "demo"},
		},
	}

	inv := inventoryFromWire(m)

	assert.Equal(t, 4, inv.Revision)
	assert.Equal(t, 1, inv.Count)
	assert.Equal(t, "sha256:abc", inv.Digest)
	require.Len(t, inv.Entries, 1)
	assert.Equal(t, "settings", inv.Entries[0].Name)
}

func TestInventoryFromWire_NilYieldsEmptyNonNilEntries(t *testing.T) {
	inv := inventoryFromWire(nil)

	assert.Equal(t, 0, inv.Revision)
	assert.NotNil(t, inv.Entries)
	assert.Empty(t, inv.Entries)
}
