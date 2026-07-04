package operator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedManifest_ParsesAllDocuments(t *testing.T) {
	objs, err := EmbeddedManifest()
	require.NoError(t, err)

	// The pinned dist/install.yaml: 1 Namespace, 3 CRDs, RBAC set, Service, Deployment.
	assert.Len(t, objs, 17)

	kindCounts := map[string]int{}
	for _, obj := range objs {
		kindCounts[obj.GetKind()]++
	}

	assert.Equal(t, 1, kindCounts["Namespace"])
	assert.Equal(t, 3, kindCounts["CustomResourceDefinition"])
	assert.Equal(t, 1, kindCounts["ServiceAccount"])
	assert.Equal(t, 1, kindCounts["Service"])
	assert.Equal(t, 1, kindCounts["Deployment"])
	assert.Positive(t, kindCounts["ClusterRole"])
	assert.Positive(t, kindCounts["ClusterRoleBinding"])
}

func TestEmbeddedManifest_CRDNamesAreExpected(t *testing.T) {
	objs, err := EmbeddedManifest()
	require.NoError(t, err)

	var crdNames []string
	for _, obj := range objs {
		if obj.GetKind() == "CustomResourceDefinition" {
			crdNames = append(crdNames, obj.GetName())
		}
	}

	assert.ElementsMatch(t, []string{
		"moduleinstances.opmodel.dev",
		"modulepackages.opmodel.dev",
		"platforms.opmodel.dev",
	}, crdNames)
}

func TestParseManifest_EmptyDocumentsAreSkipped(t *testing.T) {
	data := []byte("---\n---\napiVersion: v1\nkind: Namespace\nmetadata:\n  name: foo\n---\n")

	objs, err := ParseManifest(data)
	require.NoError(t, err)
	require.Len(t, objs, 1)
	assert.Equal(t, "Namespace", objs[0].GetKind())
	assert.Equal(t, "foo", objs[0].GetName())
}

func TestParseManifest_InvalidYAMLErrors(t *testing.T) {
	_, err := ParseManifest([]byte("foo: [unterminated"))
	assert.Error(t, err)
}
