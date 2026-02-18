package inventory

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/opmodel/cli/internal/build"
)

// makeResourceWithContent builds a *build.Resource with the given object fields.
func makeResourceWithContent(group, version, kind, namespace, name, component string, extra map[string]interface{}) *build.Resource {
	obj := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": group + "/" + version,
		"kind":       kind,
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
		},
	}}
	if group == "" {
		obj.Object["apiVersion"] = version
	}
	for k, v := range extra {
		obj.Object[k] = v
	}
	obj.SetGroupVersionKind(schema.GroupVersionKind{Group: group, Version: version, Kind: kind})
	obj.SetNamespace(namespace)
	obj.SetName(name)
	return &build.Resource{Object: obj, Component: component}
}

// standardResources returns a typical 3-resource set.
func standardResources() []*build.Resource {
	return []*build.Resource{
		makeResourceWithContent("apps", "v1", "Deployment", "ns", "app", "web", map[string]interface{}{
			"spec": map[string]interface{}{"replicas": int64(2)},
		}),
		makeResourceWithContent("", "v1", "Service", "ns", "app", "web", map[string]interface{}{
			"spec": map[string]interface{}{"port": int64(8080)},
		}),
		makeResourceWithContent("", "v1", "ConfigMap", "ns", "config", "web", map[string]interface{}{
			"data": map[string]interface{}{"key": "value"},
		}),
	}
}

func TestComputeManifestDigest_Format(t *testing.T) {
	resources := standardResources()
	digest := ComputeManifestDigest(resources)
	require.True(t, strings.HasPrefix(digest, "sha256:"), "digest should start with sha256:")
	assert.Len(t, digest, len("sha256:")+64, "SHA256 hex should be 64 chars")
}

func TestComputeManifestDigest_Deterministic_InputOrder(t *testing.T) {
	// Three orderings of the same resources should produce the same digest
	r1 := standardResources()                    // Deployment, Service, ConfigMap
	r2 := []*build.Resource{r1[2], r1[0], r1[1]} // ConfigMap, Deployment, Service
	r3 := []*build.Resource{r1[1], r1[2], r1[0]} // Service, ConfigMap, Deployment

	d1 := ComputeManifestDigest(r1)
	d2 := ComputeManifestDigest(r2)
	d3 := ComputeManifestDigest(r3)

	assert.Equal(t, d1, d2, "different input order should produce same digest (order 2)")
	assert.Equal(t, d1, d3, "different input order should produce same digest (order 3)")
}

func TestComputeManifestDigest_ContentChange(t *testing.T) {
	resources := standardResources()
	original := ComputeManifestDigest(resources)

	// Modify a field in one resource
	resources[0].Object.Object["spec"] = map[string]interface{}{"replicas": int64(5)}
	modified := ComputeManifestDigest(resources)

	assert.NotEqual(t, original, modified, "content change should produce different digest")
}

func TestComputeManifestDigest_AddedResource(t *testing.T) {
	resources := standardResources()
	original := ComputeManifestDigest(resources)

	extra := makeResourceWithContent("", "v1", "Secret", "ns", "my-secret", "web", nil)
	resources = append(resources, extra)
	withExtra := ComputeManifestDigest(resources)

	assert.NotEqual(t, original, withExtra, "adding a resource should change digest")
}

func TestComputeManifestDigest_RemovedResource(t *testing.T) {
	resources := standardResources()
	original := ComputeManifestDigest(resources)

	withoutLast := ComputeManifestDigest(resources[:2])
	assert.NotEqual(t, original, withoutLast, "removing a resource should change digest")
}

func TestComputeManifestDigest_EmptySet(t *testing.T) {
	d := ComputeManifestDigest([]*build.Resource{})
	assert.True(t, strings.HasPrefix(d, "sha256:"), "empty set should still produce a valid digest")
	// SHA256 of empty bytes
	assert.Equal(t, "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", d)
}

func TestComputeManifestDigest_SameWeightTiebreaking(t *testing.T) {
	// Deployment and StatefulSet both have weight 100 — verify tiebreaking by kind
	deployment := makeResourceWithContent("apps", "v1", "Deployment", "ns", "app", "web", nil)
	statefulSet := makeResourceWithContent("apps", "v1", "StatefulSet", "ns", "app", "db", nil)

	d1 := ComputeManifestDigest([]*build.Resource{deployment, statefulSet})
	d2 := ComputeManifestDigest([]*build.Resource{statefulSet, deployment})

	assert.Equal(t, d1, d2, "same-weight resources should be tiebroken deterministically")
}

func TestComputeManifestDigest_DoesNotMutateInput(t *testing.T) {
	resources := standardResources()
	original := make([]*build.Resource, len(resources))
	copy(original, resources)

	ComputeManifestDigest(resources)

	for i, r := range resources {
		assert.Equal(t, original[i], r, "ComputeManifestDigest should not mutate input slice")
	}
}

func BenchmarkComputeManifestDigest_20Resources(b *testing.B) {
	resources := make([]*build.Resource, 20)
	for i := range resources {
		resources[i] = makeResourceWithContent("apps", "v1", "Deployment", "ns",
			strings.Repeat("a", i+1), "web",
			map[string]interface{}{"spec": map[string]interface{}{"replicas": int64(i + 1)}})
	}

	b.ResetTimer()
	for range b.N {
		ComputeManifestDigest(resources)
	}
}
