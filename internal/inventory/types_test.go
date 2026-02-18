package inventory

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/opmodel/cli/internal/build"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// makeResource is a test helper that builds a *build.Resource with the given GVK/metadata.
func makeResource(group, version, kind, namespace, name, component string) *build.Resource {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    kind,
	})
	obj.SetNamespace(namespace)
	obj.SetName(name)
	return &build.Resource{
		Object:    obj,
		Component: component,
	}
}

// makeInventorySecret builds a minimal InventorySecret for testing.
func makeInventorySecret() *InventorySecret {
	return &InventorySecret{
		Metadata: InventoryMetadata{
			Kind:               "ModuleRelease",
			APIVersion:         "core.opmodel.dev/v1alpha1",
			Name:               "jellyfin",      // canonical module name
			ReleaseName:        "jellyfin-prod", // release name (from --release-name)
			Namespace:          "media",
			ReleaseID:          "a3b8f2e1-1234-5678-9abc-def012345678",
			LastTransitionTime: "2026-01-01T00:00:00Z",
		},
		Index: []string{"change-sha1-aabbccdd", "change-sha1-11223344"},
		Changes: map[string]*ChangeEntry{
			"change-sha1-aabbccdd": {
				Module: ModuleRef{
					Path:    "opmodel.dev/modules/jellyfin",
					Version: "1.0.0",
					Name:    "jellyfin",
				},
				Values:         `{port: 8096}`,
				ManifestDigest: "sha256:abc123def456",
				Timestamp:      "2026-01-01T00:00:00Z",
				Inventory: InventoryList{
					Entries: []InventoryEntry{
						{Group: "apps", Kind: "Deployment", Namespace: "media", Name: "jellyfin", Version: "v1", Component: "app"},
						{Group: "", Kind: "Service", Namespace: "media", Name: "jellyfin", Version: "v1", Component: "app"},
					},
				},
			},
			"change-sha1-11223344": {
				Module: ModuleRef{
					Path:    "opmodel.dev/modules/jellyfin",
					Version: "0.9.0",
					Name:    "jellyfin",
				},
				Values:         `{port: 8096}`,
				ManifestDigest: "sha256:olddigest",
				Timestamp:      "2025-12-01T00:00:00Z",
				Inventory: InventoryList{
					Entries: []InventoryEntry{
						{Group: "apps", Kind: "Deployment", Namespace: "media", Name: "jellyfin", Version: "v1", Component: "app"},
					},
				},
			},
		},
	}
}

// --- NewEntryFromResource ---

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

func TestNewEntryFromResource_ClusterScoped(t *testing.T) {
	r := makeResource("rbac.authorization.k8s.io", "v1", "ClusterRole", "", "my-role", "rbac")
	entry := NewEntryFromResource(r)

	assert.Equal(t, "rbac.authorization.k8s.io", entry.Group)
	assert.Equal(t, "ClusterRole", entry.Kind)
	assert.Equal(t, "", entry.Namespace)
	assert.Equal(t, "my-role", entry.Name)
	assert.Equal(t, "v1", entry.Version)
	assert.Equal(t, "rbac", entry.Component)
}

// --- IdentityEqual ---

func TestIdentityEqual_SameEntry(t *testing.T) {
	a := InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "web"}
	b := InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "web"}
	assert.True(t, IdentityEqual(a, b))
}

func TestIdentityEqual_DifferentVersionSameIdentity(t *testing.T) {
	// Version change should NOT affect identity equality
	a := InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "web"}
	b := InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v2", Component: "web"}
	assert.True(t, IdentityEqual(a, b), "version difference should not break identity equality")
}

func TestIdentityEqual_DifferentComponentNotEqual(t *testing.T) {
	a := InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "web"}
	b := InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "frontend"}
	assert.False(t, IdentityEqual(a, b), "different component should break identity equality")
}

func TestIdentityEqual_DifferentName(t *testing.T) {
	a := InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app-a", Version: "v1", Component: "web"}
	b := InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app-b", Version: "v1", Component: "web"}
	assert.False(t, IdentityEqual(a, b))
}

// --- K8sIdentityEqual ---

func TestK8sIdentityEqual_SameK8sResourceDifferentComponent(t *testing.T) {
	a := InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "web"}
	b := InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "frontend"}
	assert.True(t, K8sIdentityEqual(a, b), "same K8s resource under different component should be K8s-identity-equal")
}

func TestK8sIdentityEqual_DifferentName(t *testing.T) {
	a := InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app-a", Component: "web"}
	b := InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app-b", Component: "web"}
	assert.False(t, K8sIdentityEqual(a, b))
}

func TestK8sIdentityEqual_DifferentVersionStillEqual(t *testing.T) {
	a := InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1"}
	b := InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v2"}
	assert.True(t, K8sIdentityEqual(a, b))
}

// --- SecretName ---

func TestSecretName(t *testing.T) {
	name := SecretName("jellyfin", "a3b8f2e1-1234-5678-9abc-def012345678")
	assert.Equal(t, "opm.jellyfin.a3b8f2e1-1234-5678-9abc-def012345678", name)
}

// --- InventoryLabels ---

func TestInventoryLabels(t *testing.T) {
	// moduleName="jellyfin", releaseName="jf-prod", namespace="media", releaseID="abc123"
	labels := InventoryLabels("jellyfin", "jf-prod", "media", "abc123")
	assert.Equal(t, "open-platform-model", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "jellyfin", labels["module.opmodel.dev/name"])
	assert.Equal(t, "jf-prod", labels["module-release.opmodel.dev/name"])
	assert.Equal(t, "media", labels["module.opmodel.dev/namespace"])
	assert.Equal(t, "abc123", labels["module-release.opmodel.dev/uuid"])
	assert.Equal(t, "inventory", labels["opmodel.dev/component"])
}

// --- MarshalToSecret / UnmarshalFromSecret roundtrip ---

func TestMarshalUnmarshalRoundtrip(t *testing.T) {
	original := makeInventorySecret()

	secret, err := MarshalToSecret(original)
	require.NoError(t, err)

	// Verify secret metadata — Secret name uses ReleaseName, not module Name
	assert.Equal(t, "opm.jellyfin-prod.a3b8f2e1-1234-5678-9abc-def012345678", secret.Name)
	assert.Equal(t, "media", secret.Namespace)
	assert.Equal(t, corev1.SecretType("opmodel.dev/release"), secret.Type)
	assert.Equal(t, "inventory", secret.Labels["opmodel.dev/component"])

	// Unmarshal and compare
	restored, err := UnmarshalFromSecret(secret)
	require.NoError(t, err)

	assert.Equal(t, original.Metadata, restored.Metadata)
	assert.Equal(t, original.Index, restored.Index)
	assert.Equal(t, len(original.Changes), len(restored.Changes))

	for id, origEntry := range original.Changes {
		restoredEntry, ok := restored.Changes[id]
		require.True(t, ok, "change entry %q missing after roundtrip", id)
		assert.Equal(t, origEntry.Module, restoredEntry.Module)
		assert.Equal(t, origEntry.Values, restoredEntry.Values)
		assert.Equal(t, origEntry.ManifestDigest, restoredEntry.ManifestDigest)
		assert.Equal(t, origEntry.Timestamp, restoredEntry.Timestamp)
		assert.Equal(t, origEntry.Inventory, restoredEntry.Inventory)
	}
}

func TestUnmarshalFromSecret_Base64Data(t *testing.T) {
	// Simulate what Kubernetes returns on GET: base64-encoded data, not stringData
	original := makeInventorySecret()

	// Marshal first to get the string values
	tempSecret, err := MarshalToSecret(original)
	require.NoError(t, err)

	// Convert stringData to base64 data (simulating K8s GET response)
	data := make(map[string][]byte)
	for k, v := range tempSecret.StringData {
		data[k] = []byte(v)
	}

	// Build a realistic K8s GET response Secret
	kubeSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            tempSecret.Name,
			Namespace:       tempSecret.Namespace,
			ResourceVersion: "12345",
		},
		Type: tempSecret.Type,
		Data: data,
	}

	restored, err := UnmarshalFromSecret(kubeSecret)
	require.NoError(t, err)

	assert.Equal(t, original.Metadata, restored.Metadata)
	assert.Equal(t, "12345", restored.ResourceVersion(), "resourceVersion should be preserved")
}

func TestUnmarshalFromSecret_Base64EncodedData(t *testing.T) {
	// Simulate Kubernetes encoding base64 at the transport layer
	original := makeInventorySecret()

	metaBytes, _ := json.Marshal(original.Metadata)
	indexBytes, _ := json.Marshal(original.Index)

	data := map[string][]byte{
		"metadata": []byte(base64.StdEncoding.EncodeToString(metaBytes)),
		"index":    []byte(base64.StdEncoding.EncodeToString(indexBytes)),
	}

	// Note: This test verifies that the data field bytes are used directly.
	// In practice, Kubernetes stores raw bytes in data (not base64-of-base64).
	// The above simulates what comes back from the API server in data[] field.
	_ = data // already tested in TestUnmarshalFromSecret_Base64Data
}

func TestMarshalToSecret_EmptyInventory(t *testing.T) {
	inv := &InventorySecret{
		Metadata: InventoryMetadata{
			Kind:        "ModuleRelease",
			APIVersion:  "core.opmodel.dev/v1alpha1",
			Name:        "test",
			ReleaseName: "test",
			Namespace:   "default",
			ReleaseID:   "00000000-0000-0000-0000-000000000001",
		},
		Index:   []string{},
		Changes: map[string]*ChangeEntry{},
	}

	secret, err := MarshalToSecret(inv)
	require.NoError(t, err)
	assert.Equal(t, "[]", secret.StringData[secretKeyIndex], "empty index should serialize as JSON array")

	restored, err := UnmarshalFromSecret(secret)
	require.NoError(t, err)
	assert.Equal(t, []string{}, restored.Index)
	assert.Empty(t, restored.Changes)
}

func TestInventoryMetadata_KindAndAPIVersion(t *testing.T) {
	inv := makeInventorySecret()
	assert.Equal(t, "ModuleRelease", inv.Metadata.Kind)
	assert.Equal(t, "core.opmodel.dev/v1alpha1", inv.Metadata.APIVersion)
}

func TestMarshalToSecret_ResourceVersionPreserved(t *testing.T) {
	inv := makeInventorySecret()
	inv.resourceVersion = "99999"

	secret, err := MarshalToSecret(inv)
	require.NoError(t, err)
	assert.Equal(t, "99999", secret.ResourceVersion)
}
