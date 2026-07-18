package inventory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestGetRecord_NotFoundIsNoInventory(t *testing.T) {
	client := newDynamicClient()
	rec, err := GetRecord(context.Background(), client, "missing", "demo")
	require.NoError(t, err)
	assert.Nil(t, rec)
}

// TestRecordFromUnstructured_FullMapping validates the read mapping — the
// inverse of the spec/status write documents. (The server-side-apply write path
// itself is exercised in the e2e suite against a live API server; the fake
// dynamic client does not implement SSA create-or-update.)
func TestRecordFromUnstructured_FullMapping(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": APIVersionModuleInstance,
		"kind":       KindModuleInstance,
		"metadata": map[string]any{
			"name":        "podinfo",
			"namespace":   "demo",
			"annotations": map[string]any{AnnotationSource: SourceLocal},
		},
		"spec": map[string]any{
			"owner":  OwnerCLI,
			"module": map[string]any{"path": "opmodel.dev/modules/podinfo@v0", "version": "0.1.0"},
		},
		"status": map[string]any{
			"instanceUUID":            "uuid-1",
			"lastAppliedRenderDigest": "sha256:render",
			"lastAppliedAt":           "2026-01-01T00:00:00Z",
			"inventory": map[string]any{
				"revision": int64(2),
				"count":    int64(1),
				"digest":   "sha256:abc",
				"entries": []any{
					map[string]any{"group": "apps", "kind": "Deployment", "namespace": "demo", "name": "podinfo", "v": "v1", "component": "web"},
				},
			},
		},
	}}

	rec := recordFromUnstructured(obj)
	assert.Equal(t, "podinfo", rec.Name)
	assert.Equal(t, "demo", rec.Namespace)
	assert.Equal(t, OwnerCLI, rec.Owner)
	assert.Equal(t, "opmodel.dev/modules/podinfo@v0", rec.ModulePath)
	assert.Equal(t, "0.1.0", rec.ModuleVersion)
	assert.Equal(t, "uuid-1", rec.InstanceUUID)
	assert.True(t, rec.SourceLocal)
	assert.Equal(t, "sha256:render", rec.LastAppliedRenderDigest)
	assert.Equal(t, 2, rec.Inventory.Revision)
	require.Len(t, rec.Inventory.Entries, 1)
	assert.Equal(t, "Deployment", rec.Inventory.Entries[0].Kind)
	assert.Equal(t, "v1", rec.Inventory.Entries[0].Version)
	assert.Equal(t, "web", rec.Inventory.Entries[0].Component)
}

func TestRecordFromUnstructured_EmptyStatusYieldsEmptyInventory(t *testing.T) {
	obj := moduleInstanceObj("podinfo", "uuid-1")
	rec := recordFromUnstructured(obj)
	assert.NotNil(t, rec.Inventory.Entries)
	assert.Empty(t, rec.Inventory.Entries)
	assert.False(t, rec.SourceLocal)
}

func TestListRecords_SkipsMalformedInventory(t *testing.T) {
	ctx := context.Background()
	good := moduleInstanceObj("good", "uuid-good")
	// A CR whose status.inventory is a scalar cannot be interpreted.
	bad := moduleInstanceObj("bad", "uuid-bad")
	_ = unstructured.SetNestedField(bad.Object, "not-an-object", "status", "inventory")

	records, err := ListRecords(ctx, newDynamicClient(good, bad), "demo")
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "good", records[0].Name)
}

func TestListRecords_SortedByName(t *testing.T) {
	ctx := context.Background()
	client := newDynamicClient(
		moduleInstanceObj("zebra", "uuid-z"),
		moduleInstanceObj("alpha", "uuid-a"),
		moduleInstanceObj("mid", "uuid-m"),
	)
	records, err := ListRecords(ctx, client, "demo")
	require.NoError(t, err)
	require.Len(t, records, 3)
	assert.Equal(t, "alpha", records[0].Name)
	assert.Equal(t, "mid", records[1].Name)
	assert.Equal(t, "zebra", records[2].Name)
}

func TestFindRecordByInstanceUUID(t *testing.T) {
	ctx := context.Background()
	client := newDynamicClient(
		moduleInstanceObj("a", "uuid-a"),
		moduleInstanceObj("b", "uuid-b"),
	)
	rec, err := FindRecordByInstanceUUID(ctx, client, "demo", "uuid-b")
	require.NoError(t, err)
	require.NotNil(t, rec)
	assert.Equal(t, "b", rec.Name)

	rec, err = FindRecordByInstanceUUID(ctx, client, "demo", "uuid-missing")
	require.NoError(t, err)
	assert.Nil(t, rec)
}

func TestDeleteCR_NotFoundIsSuccess(t *testing.T) {
	ctx := context.Background()
	client := newDynamicClient()
	require.NoError(t, DeleteCR(ctx, client, "missing", "demo"))
}

func TestDeleteCR_RemovesExisting(t *testing.T) {
	ctx := context.Background()
	client := newDynamicClient(moduleInstanceObj("a", "uuid-a"))
	require.NoError(t, DeleteCR(ctx, client, "a", "demo"))
	rec, err := GetRecord(ctx, client, "a", "demo")
	require.NoError(t, err)
	assert.Nil(t, rec)
}

func moduleInstanceObj(name, uuid string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": APIVersionModuleInstance,
		"kind":       KindModuleInstance,
		"metadata":   map[string]any{"name": name, "namespace": "demo"},
		"spec":       map[string]any{"owner": OwnerCLI},
		"status":     map[string]any{"instanceUUID": uuid},
	}}
}
