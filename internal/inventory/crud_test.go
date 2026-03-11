package inventory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
)

func makeTestClient(objects ...runtime.Object) *kubernetes.Client {
	return &kubernetes.Client{Clientset: fake.NewClientset(objects...)}
}

func makeTestRecord(moduleName, releaseName, namespace, releaseID string) *inventory.ReleaseInventoryRecord {
	return &inventory.ReleaseInventoryRecord{
		CreatedBy: inventory.CreatedByCLI,
		ReleaseMetadata: inventory.ReleaseMetadata{
			Kind:               "ModuleRelease",
			APIVersion:         "core.opmodel.dev/v1alpha1",
			ReleaseName:        releaseName,
			ReleaseNamespace:   namespace,
			ReleaseID:          releaseID,
			LastTransitionTime: "2026-01-01T00:00:00Z",
		},
		ModuleMetadata: inventory.ModuleMetadata{
			Kind:       "Module",
			APIVersion: "core.opmodel.dev/v1alpha1",
			Name:       moduleName,
			Version:    "1.2.3",
		},
		Inventory: inventory.Inventory{Entries: []inventory.InventoryEntry{{Kind: "ConfigMap", Namespace: namespace, Name: "cfg"}}},
	}
}

func TestGetInventory_FirstTimeApply_ReturnsNil(t *testing.T) {
	client := makeTestClient()
	inv, err := inventory.GetInventory(context.Background(), client, "jellyfin", "media", "release-uuid-123")
	require.NoError(t, err)
	assert.Nil(t, inv)
}

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	original := makeTestRecord("jellyfin", "jellyfin", "media", "uuid-1")
	secret, err := inventory.MarshalToSecret(original)
	require.NoError(t, err)
	assert.Equal(t, inventory.SecretName("jellyfin", "uuid-1"), secret.Name)
	restored, err := inventory.UnmarshalFromSecret(secret)
	require.NoError(t, err)
	assert.Equal(t, original.CreatedBy, restored.CreatedBy)
	assert.Equal(t, original.ReleaseMetadata, restored.ReleaseMetadata)
	assert.Equal(t, original.ModuleMetadata, restored.ModuleMetadata)
	assert.Equal(t, original.Inventory, restored.Inventory)
}

func TestGetInventory_ByName_Success(t *testing.T) {
	secret, err := inventory.MarshalToSecret(makeTestRecord("jellyfin", "jellyfin", "media", "uuid-abc"))
	require.NoError(t, err)
	secret.ResourceVersion = "100"
	client := makeTestClient(secret)
	result, err := inventory.GetInventory(context.Background(), client, "jellyfin", "media", "uuid-abc")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "jellyfin", result.ModuleMetadata.Name)
	assert.Equal(t, "100", result.ResourceVersion())
}

func TestGetInventory_FallbackToLabelLookup(t *testing.T) {
	secret, err := inventory.MarshalToSecret(makeTestRecord("jellyfin", "jellyfin", "media", "uuid-xyz"))
	require.NoError(t, err)
	secret.Name = "legacy-inventory-secret"
	client := makeTestClient(secret)
	result, err := inventory.GetInventory(context.Background(), client, "jellyfin", "media", "uuid-xyz")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "uuid-xyz", result.ReleaseMetadata.ReleaseID)
}

func TestWriteInventory_CreateAndUpdate(t *testing.T) {
	ctx := context.Background()
	client := makeTestClient()
	record := &inventory.ReleaseInventoryRecord{
		ReleaseMetadata: inventory.ReleaseMetadata{
			Kind:             "ModuleRelease",
			APIVersion:       "core.opmodel.dev/v1alpha1",
			ReleaseName:      "mc",
			ReleaseNamespace: "default",
			ReleaseID:        "uuid-create-test",
		},
		Inventory: inventory.Inventory{Entries: []inventory.InventoryEntry{{Kind: "ConfigMap", Namespace: "default", Name: "cfg"}}},
	}

	err := inventory.WriteInventory(ctx, client, record, "minecraft", "mod-uuid-abc", "1.0.0", inventory.CreatedByController)
	require.NoError(t, err)

	stored, err := inventory.GetInventory(ctx, client, "mc", "default", "uuid-create-test")
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, inventory.CreatedByController, stored.CreatedBy)
	assert.Equal(t, "minecraft", stored.ModuleMetadata.Name)
	assert.Equal(t, "1.0.0", stored.ModuleMetadata.Version)

	stored.Inventory.Revision = 2
	stored.SetResourceVersion("1")
	err = inventory.WriteInventory(ctx, client, stored, "", "", "1.1.0", inventory.CreatedByCLI)
	require.NoError(t, err)

	updated, err := inventory.GetInventory(ctx, client, "mc", "default", "uuid-create-test")
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, inventory.CreatedByController, updated.CreatedBy)
	assert.Equal(t, "1.1.0", updated.ModuleMetadata.Version)
	assert.Equal(t, 2, updated.Inventory.Revision)
}

func TestFindInventoryByReleaseName_Found(t *testing.T) {
	secret, err := inventory.MarshalToSecret(makeTestRecord("minecraft", "mc", "default", "uuid-mc-001"))
	require.NoError(t, err)
	secret.ResourceVersion = "1"
	client := makeTestClient(secret)
	found, err := inventory.FindInventoryByReleaseName(context.Background(), client, "mc", "default")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "minecraft", found.ModuleMetadata.Name)
	assert.Equal(t, "mc", found.ReleaseMetadata.ReleaseName)
}

func TestLegacyCreatedByDefaultsToCLI(t *testing.T) {
	record := makeTestRecord("demo", "demo", "default", "uuid-legacy")
	record.CreatedBy = ""
	assert.Equal(t, inventory.CreatedByCLI, record.NormalizedCreatedBy())
}

func TestUnmarshalFromSecret_MissingPayloadFails(t *testing.T) {
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "bad", Namespace: "default"}}
	_, err := inventory.UnmarshalFromSecret(secret)
	assert.Error(t, err)
}
