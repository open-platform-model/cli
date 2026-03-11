package inventory

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	pkgcore "github.com/opmodel/cli/pkg/core"
)

func makeResource(group, version, kind, namespace, name, component string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{Group: group, Version: version, Kind: kind})
	obj.SetNamespace(namespace)
	obj.SetName(name)
	obj.SetLabels(map[string]string{pkgcore.LabelComponentName: component})
	return obj
}

func makeInventorySecret() *InventorySecret {
	return &InventorySecret{
		ReleaseMetadata: ReleaseMetadata{
			Kind:               "ModuleRelease",
			APIVersion:         "core.opmodel.dev/v1alpha1",
			ReleaseName:        "jellyfin-prod",
			ReleaseNamespace:   "media",
			ReleaseID:          "a3b8f2e1-1234-5678-9abc-def012345678",
			LastTransitionTime: "2026-01-01T00:00:00Z",
			CreatedBy:          CreatedByCLI,
		},
		ModuleMetadata: ModuleMetadata{
			Kind:       "Module",
			APIVersion: "core.opmodel.dev/v1alpha1",
			Name:       "jellyfin",
			UUID:       "b1c2d3e4-5678-90ab-cdef-012345678901",
		},
		Index: []string{"change-sha1-aabbccdd", "change-sha1-11223344"},
		Changes: map[string]*ChangeEntry{
			"change-sha1-aabbccdd": {
				Source:         ChangeSource{Path: "opmodel.dev/modules/jellyfin", Version: "1.0.0", ReleaseName: "jellyfin-prod"},
				Values:         `{port: 8096}`,
				ManifestDigest: "sha256:abc123def456",
				Timestamp:      "2026-01-01T00:00:00Z",
				Inventory:      InventoryList{Entries: []InventoryEntry{{Group: "apps", Kind: "Deployment", Namespace: "media", Name: "jellyfin", Version: "v1", Component: "app"}, {Group: "", Kind: "Service", Namespace: "media", Name: "jellyfin", Version: "v1", Component: "app"}}},
			},
			"change-sha1-11223344": {
				Source:         ChangeSource{Path: "opmodel.dev/modules/jellyfin", Version: "0.9.0", ReleaseName: "jellyfin-prod"},
				Values:         `{port: 8096}`,
				ManifestDigest: "sha256:olddigest",
				Timestamp:      "2025-12-01T00:00:00Z",
				Inventory:      InventoryList{Entries: []InventoryEntry{{Group: "apps", Kind: "Deployment", Namespace: "media", Name: "jellyfin", Version: "v1", Component: "app"}}},
			},
		},
	}
}

func TestNewEntryFromResource_Namespaced(t *testing.T) {
	r := makeResource("apps", "v1", "Deployment", "production", "my-app", "app")
	entry := NewEntryFromResource(r)
	assert.Equal(t, "apps", entry.Group)
	assert.Equal(t, "Deployment", entry.Kind)
	assert.Equal(t, "production", entry.Namespace)
	assert.Equal(t, "my-app", entry.Name)
	assert.Equal(t, "v1", entry.Version)
	assert.Equal(t, "app", entry.Component)
}

func TestIdentityHelpers(t *testing.T) {
	a := InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "web"}
	b := InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v2", Component: "web"}
	c := InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "frontend"}
	assert.True(t, IdentityEqual(a, b))
	assert.False(t, IdentityEqual(a, c))
	assert.True(t, K8sIdentityEqual(a, c))
}

func TestSecretName(t *testing.T) {
	name := SecretName("jellyfin", "a3b8f2e1-1234-5678-9abc-def012345678")
	assert.Equal(t, "opm.jellyfin.a3b8f2e1-1234-5678-9abc-def012345678", name)
}

func TestInventoryLabels(t *testing.T) {
	labels := InventoryLabels("jf-prod", "media", "abc123")
	assert.Equal(t, "open-platform-model", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "jf-prod", labels["module-release.opmodel.dev/name"])
	assert.Equal(t, "media", labels["module-release.opmodel.dev/namespace"])
	assert.Equal(t, "abc123", labels["module-release.opmodel.dev/uuid"])
	assert.Equal(t, "inventory", labels["opmodel.dev/component"])
	assert.Len(t, labels, 5)
}

func TestMarshalUnmarshalRoundtrip(t *testing.T) {
	original := makeInventorySecret()
	secret, err := MarshalToSecret(original)
	require.NoError(t, err)
	assert.Equal(t, "opm.jellyfin-prod.a3b8f2e1-1234-5678-9abc-def012345678", secret.Name)
	assert.Equal(t, "media", secret.Namespace)
	assert.Equal(t, corev1.SecretType("opmodel.dev/release"), secret.Type)
	restored, err := UnmarshalFromSecret(secret)
	require.NoError(t, err)
	assert.Equal(t, original.ReleaseMetadata, restored.ReleaseMetadata)
	assert.Equal(t, original.ModuleMetadata, restored.ModuleMetadata)
	assert.Equal(t, original.Index, restored.Index)
	assert.Equal(t, len(original.Changes), len(restored.Changes))
}

func TestUnmarshalFromSecret_Base64Data(t *testing.T) {
	original := makeInventorySecret()
	tempSecret, err := MarshalToSecret(original)
	require.NoError(t, err)
	data := make(map[string][]byte)
	for k, v := range tempSecret.StringData {
		data[k] = []byte(v)
	}
	kubeSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: tempSecret.Name, Namespace: tempSecret.Namespace, ResourceVersion: "12345"}, Type: tempSecret.Type, Data: data}
	restored, err := UnmarshalFromSecret(kubeSecret)
	require.NoError(t, err)
	assert.Equal(t, original.ReleaseMetadata, restored.ReleaseMetadata)
	assert.Equal(t, original.ModuleMetadata, restored.ModuleMetadata)
	assert.Equal(t, "12345", restored.ResourceVersion())
}

func TestMarshalToSecret_EmptyInventory(t *testing.T) {
	inv := &InventorySecret{ReleaseMetadata: ReleaseMetadata{Kind: "ModuleRelease", APIVersion: "core.opmodel.dev/v1alpha1", ReleaseName: "test", ReleaseNamespace: "default", ReleaseID: "00000000-0000-0000-0000-000000000001", CreatedBy: CreatedByCLI}, ModuleMetadata: ModuleMetadata{Kind: "Module", APIVersion: "core.opmodel.dev/v1alpha1", Name: "test-module"}, Index: []string{}, Changes: map[string]*ChangeEntry{}}
	secret, err := MarshalToSecret(inv)
	require.NoError(t, err)
	assert.Equal(t, "[]", secret.StringData[secretKeyIndex])
	restored, err := UnmarshalFromSecret(secret)
	require.NoError(t, err)
	assert.Equal(t, []string{}, restored.Index)
	assert.Empty(t, restored.Changes)
}

func TestMarshalToSecret_ResourceVersionPreserved(t *testing.T) {
	inv := makeInventorySecret()
	inv.SetResourceVersion("99999")
	secret, err := MarshalToSecret(inv)
	require.NoError(t, err)
	assert.Equal(t, "99999", secret.ResourceVersion)
}

func TestReleaseMetadata_JSONFieldNames(t *testing.T) {
	rel := ReleaseMetadata{Kind: "ModuleRelease", APIVersion: "core.opmodel.dev/v1alpha1", ReleaseName: "mc", ReleaseNamespace: "default", ReleaseID: "abc-123", LastTransitionTime: "2026-01-01T00:00:00Z", CreatedBy: CreatedByCLI}
	data, err := json.Marshal(rel)
	require.NoError(t, err)
	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, "mc", raw["name"])
	assert.Equal(t, "abc-123", raw["uuid"])
	assert.Equal(t, "cli", raw["createdBy"])
	_, hasReleaseName := raw["releaseName"]
	assert.False(t, hasReleaseName)
}

func TestModuleMetadata_UUIDOmittedWhenEmpty(t *testing.T) {
	mod := ModuleMetadata{Kind: "Module", APIVersion: "core.opmodel.dev/v1alpha1", Name: "minecraft"}
	data, err := json.Marshal(mod)
	require.NoError(t, err)
	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	_, hasUUID := raw["uuid"]
	assert.False(t, hasUUID)
}

func TestUnmarshalFromSecret_MissingModuleMetadata_NotAnError(t *testing.T) {
	inv := makeInventorySecret()
	secret, err := MarshalToSecret(inv)
	require.NoError(t, err)
	delete(secret.StringData, secretKeyModuleMetadata)
	restored, err := UnmarshalFromSecret(secret)
	require.NoError(t, err)
	assert.Equal(t, ModuleMetadata{}, restored.ModuleMetadata)
}

func TestUnmarshalFromSecret_MissingCreatedBy_IsLegacyCLI(t *testing.T) {
	inv := makeInventorySecret()
	inv.ReleaseMetadata.CreatedBy = ""
	secret, err := MarshalToSecret(inv)
	require.NoError(t, err)
	restored, err := UnmarshalFromSecret(secret)
	require.NoError(t, err)
	assert.Equal(t, CreatedByCLI, restored.ReleaseMetadata.NormalizedCreatedBy())
}

func TestNormalizeCreatedBy_DefaultsToCLI(t *testing.T) {
	assert.Equal(t, CreatedByCLI, NormalizeCreatedBy(""))
	assert.Equal(t, CreatedByCLI, NormalizeCreatedBy("bogus"))
	assert.Equal(t, CreatedByController, NormalizeCreatedBy(CreatedByController))
}
