package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func manifestObject(apiVersion, kind, namespace, name string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetAPIVersion(apiVersion)
	u.SetKind(kind)
	u.SetName(name)
	if namespace != "" {
		u.SetNamespace(namespace)
	}
	return u
}

// manifestInput mixes kinds of different weights with equal-weight objects
// that differ only in namespace or name, in an order that matches none of
// the sort keys.
func manifestInput() []*unstructured.Unstructured {
	return []*unstructured.Unstructured{
		manifestObject("apps/v1", "Deployment", "b", "web"),
		manifestObject("v1", "ConfigMap", "b", "cfg"),
		manifestObject("apps/v1", "Deployment", "a", "web"),
		manifestObject("v1", "Namespace", "", "b"),
		manifestObject("v1", "ConfigMap", "a", "zeta"),
		manifestObject("v1", "ConfigMap", "a", "alpha"),
	}
}

// wantManifestOrder is ascending weight (Namespace, ConfigMap, Deployment),
// then namespace, then name.
var wantManifestOrder = []string{
	"Namespace//b",
	"ConfigMap/a/alpha",
	"ConfigMap/a/zeta",
	"ConfigMap/b/cfg",
	"Deployment/a/web",
	"Deployment/b/web",
}

func manifestKeys(objs []*unstructured.Unstructured) []string {
	out := make([]string, len(objs))
	for i, o := range objs {
		out[i] = o.GetKind() + "/" + o.GetNamespace() + "/" + o.GetName()
	}
	return out
}

func TestSortResources_WeightThenNamespaceThenName(t *testing.T) {
	objs := manifestInput()
	sortResources(objs)
	assert.Equal(t, wantManifestOrder, manifestKeys(objs))
}

// TestWriteManifests_OrdersOutput pins the order of the `module build`
// output itself: the YAML documents follow sortResources.
func TestWriteManifests_OrdersOutput(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, WriteManifests(manifestInput(), ManifestOptions{Format: FormatYAML, Writer: &buf}))

	docs := strings.Split(buf.String(), "---\n")
	require.Len(t, docs, len(wantManifestOrder))
	for i, want := range wantManifestOrder {
		parts := strings.SplitN(want, "/", 3)
		assert.Contains(t, docs[i], "kind: "+parts[0], "document %d", i)
		assert.Contains(t, docs[i], "name: "+parts[2], "document %d", i)
		if parts[1] != "" {
			assert.Contains(t, docs[i], "namespace: "+parts[1], "document %d", i)
		} else {
			assert.NotContains(t, docs[i], "namespace:", "document %d", i)
		}
	}
}
