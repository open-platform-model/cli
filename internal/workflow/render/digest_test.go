package render

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	opmexit "github.com/open-platform-model/cli/internal/exit"
	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
	"github.com/open-platform-model/library/opm/k8s/object"
)

// digestResource compiles a CUE manifest source into an *object.Resource.
func digestResource(t *testing.T, src string) *object.Resource {
	t.Helper()
	v := cuecontext.New().CompileString(src)
	require.NoError(t, v.Err())
	return &object.Resource{Value: v, Instance: "demo", Component: "web", Transformer: "t"}
}

// digestResources is a typical three-object compiled set; each object carries
// the given managed-by value and team label.
func digestResources(t *testing.T, managedBy, team string) []*object.Resource {
	t.Helper()
	labels := `labels: {"app.kubernetes.io/managed-by": "` + managedBy + `", team: "` + team + `"}`
	return []*object.Resource{
		digestResource(t, `apiVersion: "apps/v1", kind: "Deployment", metadata: {name: "app", namespace: "ns", `+labels+`}, spec: replicas: 2`),
		digestResource(t, `apiVersion: "v1", kind: "Service", metadata: {name: "app", namespace: "ns", `+labels+`}, spec: port: 8080`),
		digestResource(t, `apiVersion: "v1", kind: "ConfigMap", metadata: {name: "config", namespace: "ns", `+labels+`}, data: key: "value"`),
	}
}

// retiredRenderDigest is the value the CLI's own render digest gave for the
// three-object set of TestExportAndDigest_DiffersFromTheRetiredDigest. The
// stored render digest changes once, in the release that first records the
// library's shared digest, on purpose (0012:D6, 0012:D7:R4).
const retiredRenderDigest = "sha256:0404868537e1114e9ea6e7d02af68ba6a4693531727c4be713212753de6840ae"

func TestExportAndDigest_IsTheLibraryDigestOfTheExport(t *testing.T) {
	resources := digestResources(t, "opm-cli", "a")
	exported, digest, err := exportAndDigest(resources)
	require.NoError(t, err)
	require.Len(t, exported, len(resources))
	for i, want := range []string{"Deployment", "Service", "ConfigMap"} {
		assert.Equal(t, want, exported[i].Object.GetKind(), "the export keeps the input order")
	}

	want, err := k8sinventory.RenderDigest(exported)
	require.NoError(t, err)
	assert.Equal(t, want, digest)
}

// The managed-by value is the runtime's name, so the CLI and the operator
// digest one render equally (0012:D6:R2).
func TestExportAndDigest_IgnoresTheManagedByValue(t *testing.T) {
	_, cli, err := exportAndDigest(digestResources(t, "opm-cli", "a"))
	require.NoError(t, err)
	_, operator, err := exportAndDigest(digestResources(t, "opm-controller", "a"))
	require.NoError(t, err)
	assert.Equal(t, cli, operator)
}

// Any other label moves the digest (0012:D6:R3).
func TestExportAndDigest_AnotherLabelMovesTheDigest(t *testing.T) {
	_, a, err := exportAndDigest(digestResources(t, "opm-cli", "a"))
	require.NoError(t, err)
	_, b, err := exportAndDigest(digestResources(t, "opm-cli", "b"))
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
}

func TestExportAndDigest_DiffersFromTheRetiredDigest(t *testing.T) {
	_, digest, err := exportAndDigest([]*object.Resource{
		digestResource(t, `apiVersion: "apps/v1", kind: "Deployment", metadata: {name: "app", namespace: "ns"}, spec: replicas: 2`),
		digestResource(t, `apiVersion: "v1", kind: "Service", metadata: {name: "app", namespace: "ns"}, spec: port: 8080`),
		digestResource(t, `apiVersion: "v1", kind: "ConfigMap", metadata: {name: "config", namespace: "ns"}, data: key: "value"`),
	})
	require.NoError(t, err)
	assert.NotEqual(t, retiredRenderDigest, digest)
}

// An object that cannot be exported fails the render with the general exit
// code and an error naming the conversion.
func TestExportAndDigest_ExportFailureIsAGeneralError(t *testing.T) {
	_, _, err := exportAndDigest([]*object.Resource{
		digestResource(t, `apiVersion: "v1", kind: "ConfigMap", metadata: {name: "config", namespace: "ns"}, data: key: string`),
	})
	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, opmexit.ExitGeneralError, exitErr.Code)
	assert.Contains(t, err.Error(), "converting rendered resources")
}
