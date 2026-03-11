package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestComputeStaleSet(t *testing.T) {
	previous := []InventoryEntry{{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "web"}, {Group: "", Kind: "Service", Namespace: "ns", Name: "svc", Version: "v1", Component: "web"}}
	current := []InventoryEntry{{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v2", Component: "web"}}
	stale := ComputeStaleSet(previous, current)
	require.Len(t, stale, 1)
	assert.Equal(t, "svc", stale[0].Name)
	assert.Equal(t, "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", ComputeDigest(nil))
	assert.Equal(t, ComputeDigest(previous), ComputeDigest([]InventoryEntry{previous[1], previous[0]}))
}
