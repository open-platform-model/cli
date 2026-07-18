package platform

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/library/opm/helper/synth"

	"github.com/open-platform-model/cli/internal/inventory"
)

// newFakeDynamic builds a fake dynamic client that knows the Platform GVR,
// pre-seeded with objs.
func newFakeDynamic(objs ...runtime.Object) *dynamicfake.FakeDynamicClient {
	scheme := runtime.NewScheme()
	gvrToKind := map[k8sschema.GroupVersionResource]string{
		inventory.PlatformGVR: inventory.KindPlatform + "List",
	}
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToKind, objs...)
}

func clusterPlatformObj(spec map[string]any) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": inventory.GroupOpmodel + "/" + inventory.VersionV1Alpha1,
		"kind":       inventory.KindPlatform,
		"metadata":   map[string]any{"name": inventory.PlatformSingletonName},
		"spec":       spec,
	}}
}

func testInput() synth.PlatformInput {
	return synth.PlatformInput{
		Name: "cluster",
		Type: "kubernetes",
		Subscriptions: map[string]synth.SubscriptionSpec{
			"opmodel.dev/catalogs/opm": {
				Filter: &synth.FilterSpec{Range: ">=1.0.0-0 <2.0.0-0"},
			},
		},
	}
}

func TestClusterSpecGetterFor_ReadsSingleton(t *testing.T) {
	dyn := newFakeDynamic(clusterPlatformObj(map[string]any{
		"type": "kubernetes",
	}))

	spec, name, unavailable, err := ClusterSpecGetterFor(dyn)(context.Background())
	require.NoError(t, err)
	assert.Empty(t, unavailable)
	assert.Equal(t, inventory.PlatformSingletonName, name)
	assert.Equal(t, "kubernetes", spec["type"])
}

func TestClusterSpecGetterFor_NotFoundIsFallback(t *testing.T) {
	dyn := newFakeDynamic()

	_, _, unavailable, err := ClusterSpecGetterFor(dyn)(context.Background())
	require.NoError(t, err)
	assert.Contains(t, unavailable, "no Platform CR")
}

func TestClusterSpecGetterFor_ForbiddenIsFallback(t *testing.T) {
	dyn := newFakeDynamic()
	dyn.PrependReactor("get", "platforms", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(
			k8sschema.GroupResource{Group: inventory.GroupOpmodel, Resource: inventory.ResourcePlatforms},
			inventory.PlatformSingletonName, nil)
	})

	_, _, unavailable, err := ClusterSpecGetterFor(dyn)(context.Background())
	require.NoError(t, err)
	assert.Contains(t, unavailable, "RBAC")
}

func TestEnsureClusterPlatform_CreatesWhenAbsent(t *testing.T) {
	dyn := newFakeDynamic()

	require.NoError(t, EnsureClusterPlatform(context.Background(), dyn, testInput()))

	created, err := dyn.Resource(inventory.PlatformGVR).Get(context.Background(),
		inventory.PlatformSingletonName, metav1.GetOptions{})
	require.NoError(t, err)
	spec, _, err := unstructured.NestedMap(created.Object, "spec")
	require.NoError(t, err)
	assert.Equal(t, "kubernetes", spec["type"])
	// The wire spec keeps name in metadata only.
	_, hasName := spec["name"]
	assert.False(t, hasName, "spec must not carry a name field")

	rng, _, err := unstructured.NestedString(created.Object,
		"spec", "registry", "opmodel.dev/catalogs/opm", "filter", "range")
	require.NoError(t, err)
	assert.Equal(t, ">=1.0.0-0 <2.0.0-0", rng)
}

func TestEnsureClusterPlatform_AlreadyExistsIsNoop(t *testing.T) {
	existing := clusterPlatformObj(map[string]any{
		"type": "pre-existing",
	})
	dyn := newFakeDynamic(existing)

	require.NoError(t, EnsureClusterPlatform(context.Background(), dyn, testInput()))

	// The existing Platform must be untouched — never overwritten (D22).
	after, err := dyn.Resource(inventory.PlatformGVR).Get(context.Background(),
		inventory.PlatformSingletonName, metav1.GetOptions{})
	require.NoError(t, err)
	typ, _, err := unstructured.NestedString(after.Object, "spec", "type")
	require.NoError(t, err)
	assert.Equal(t, "pre-existing", typ)
}

func TestEnsureClusterPlatform_ForbiddenDegradesToWarning(t *testing.T) {
	dyn := newFakeDynamic()
	dyn.PrependReactor("create", "platforms", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(
			k8sschema.GroupResource{Group: inventory.GroupOpmodel, Resource: inventory.ResourcePlatforms},
			inventory.PlatformSingletonName, nil)
	})

	// D17: forbidden create is a warning, not an error.
	require.NoError(t, EnsureClusterPlatform(context.Background(), dyn, testInput()))
}

func TestEnsureClusterPlatform_OtherErrorIsFatal(t *testing.T) {
	dyn := newFakeDynamic()
	dyn.PrependReactor("create", "platforms", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(assert.AnError)
	})

	require.Error(t, EnsureClusterPlatform(context.Background(), dyn, testInput()))
}
