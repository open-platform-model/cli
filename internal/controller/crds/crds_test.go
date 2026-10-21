package crds

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func crd(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apiextensions.k8s.io/v1",
		"kind":       Kind,
		"metadata":   map[string]any{"name": name, "labels": map[string]any{"b": "2", "a": "1"}},
		"spec":       map[string]any{"group": "opmodel.dev", "versions": []any{map[string]any{"name": "v1alpha1", "served": true}}},
	}}
}

// Manifest writes the header and sorted keys, and Parse reads it back.
func TestManifest_RoundTrips(t *testing.T) {
	data, err := Manifest(crd("widgets.opmodel.dev"))
	require.NoError(t, err)
	assert.Equal(t, Header+`apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  labels:
    a: "1"
    b: "2"
  name: widgets.opmodel.dev
spec:
  group: opmodel.dev
  versions:
  - name: v1alpha1
    served: true
`, string(data))

	obj, err := Parse(data)
	require.NoError(t, err)
	assert.Equal(t, crd("widgets.opmodel.dev").Object, obj.Object)
}

func TestManifest_RefusesWhatIsNotANamedCRD(t *testing.T) {
	notCRD := crd("x.opmodel.dev")
	notCRD.SetKind("Deployment")
	_, err := Manifest(notCRD)
	assert.Error(t, err)
	_, err = Manifest(crd("../escape"))
	assert.Error(t, err)
}

// Read returns the manifests in file-name order and refuses a file that
// holds another CRD than its name says, or nothing at all.
func TestRead(t *testing.T) {
	a, err := Manifest(crd("a.opmodel.dev"))
	require.NoError(t, err)
	b, err := Manifest(crd("b.opmodel.dev"))
	require.NoError(t, err)

	objs, err := Read(fstest.MapFS{
		"b.opmodel.dev.yaml": {Data: b},
		"a.opmodel.dev.yaml": {Data: a},
		"crds.go":            {Data: []byte("package crds\n")},
	})
	require.NoError(t, err)
	require.Len(t, objs, 2)
	assert.Equal(t, "a.opmodel.dev", objs[0].GetName())
	assert.Equal(t, "b.opmodel.dev", objs[1].GetName())

	_, err = Read(fstest.MapFS{"c.opmodel.dev.yaml": {Data: a}})
	assert.ErrorContains(t, err, "belongs in a.opmodel.dev.yaml")

	_, err = Read(fstest.MapFS{"x.yaml": {Data: []byte("kind: ConfigMap\nmetadata: {name: x}\n")}})
	assert.ErrorContains(t, err, "not a named CustomResourceDefinition")

	_, err = Read(fstest.MapFS{})
	assert.ErrorContains(t, err, "no CRD manifests")
}

// The embedded set parses.
func TestObjects(t *testing.T) {
	objs, err := Objects()
	require.NoError(t, err)
	assert.NotEmpty(t, objs)
}
