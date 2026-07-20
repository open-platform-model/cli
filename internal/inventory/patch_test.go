package inventory

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/cli/internal/kubernetes"
)

// applyPatchRecorder captures the body of the last server-side-apply patch.
type applyPatchRecorder struct {
	body map[string]any
}

// newApplyPatchClient installs a reactor that intercepts apply patches, records
// the applied document, and answers with a CR carrying the given generation.
//
// These tests assert the COMPLETENESS of the outgoing document. Under
// server-side apply, a field the opm-cli manager owns but omits is released and
// pruned by the API server — so "what the payload leaves out" is what deletes
// data. The fake dynamic client does not implement apply-patch merge semantics,
// which is exactly why an earlier version of these tests asserted the opposite
// invariant (a minimal single-field document) and passed while the real
// behavior was broken. The merge semantics themselves are covered against a
// real API server in tests/integration/ssa-ownership.
func newApplyPatchClient(t *testing.T, generation int64) (*kubernetes.Client, *applyPatchRecorder) {
	t.Helper()

	rec := &applyPatchRecorder{}
	client := newDynamicClient()

	fake, ok := client.Dynamic.(*dynamicfake.FakeDynamicClient)
	require.True(t, ok, "expected a fake dynamic client")

	fake.PrependReactor("patch", ResourceModuleInstances, func(action k8stesting.Action) (bool, runtime.Object, error) {
		patch, ok := action.(k8stesting.PatchAction)
		require.True(t, ok)

		body := map[string]any{}
		require.NoError(t, json.Unmarshal(patch.GetPatch(), &body))
		rec.body = body

		return true, &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": APIVersionModuleInstance,
			"kind":       KindModuleInstance,
			"metadata": map[string]any{
				"name":       patch.GetName(),
				"namespace":  patch.GetNamespace(),
				"generation": generation,
			},
		}}, nil
	})

	return client, rec
}

// specOf returns the "spec" object of the captured patch body.
func (r *applyPatchRecorder) specOf(t *testing.T) map[string]any {
	t.Helper()
	require.NotNil(t, r.body, "no patch was captured")
	spec, ok := r.body["spec"].(map[string]any)
	require.True(t, ok, "captured patch has no spec object")
	return spec
}

// The ownership flip must carry the module reference and values alongside the
// new owner. A document with only spec.owner releases opm-cli's claim on
// spec.module — a required CRD field — and the API server rejects the whole
// apply, so handoff would fail on every real cluster.
func TestApplySpec_OwnershipFlipCarriesTheWholeOwnedSpec(t *testing.T) {
	client, rec := newApplyPatchClient(t, 9)

	generation, err := ApplySpec(context.Background(), client, SpecInput{
		Name:          "podinfo",
		Namespace:     "demo",
		Owner:         OwnerOperator,
		ModulePath:    "opmodel.dev/modules/podinfo@v0",
		ModuleVersion: "0.1.0",
		Values:        map[string]any{"replicas": 3},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(9), generation, "the flip must report the resulting generation for the reconcile wait")

	spec := rec.specOf(t)
	assert.Equal(t, OwnerOperator, spec["owner"])

	module, ok := spec["module"].(map[string]any)
	require.True(t, ok, "spec.module must be present — omitting it prunes a required field")
	assert.Equal(t, "opmodel.dev/modules/podinfo@v0", module["path"])
	assert.Equal(t, "0.1.0", module["version"])

	values, ok := spec["values"].(map[string]any)
	require.True(t, ok, "spec.values must be present — omitting it silently deletes the instance's config")
	assert.InDelta(t, 3.0, values["replicas"], 0.0001)
}

// The thin editor restates the owner it read rather than omitting it. Omitting
// would release the field and let the API server prune the operator's
// ownership marker.
func TestApplySpec_ThinEditorPreservesTheOperatorOwner(t *testing.T) {
	client, rec := newApplyPatchClient(t, 4)

	_, err := ApplySpec(context.Background(), client, SpecInput{
		Name:          "podinfo",
		Namespace:     "demo",
		Owner:         OwnerOperator, // the value read from the live CR
		ModulePath:    "opmodel.dev/modules/podinfo@v0",
		ModuleVersion: "0.2.0",
		Values:        map[string]any{"replicas": 2},
	})
	require.NoError(t, err)

	spec := rec.specOf(t)
	assert.Equal(t, OwnerOperator, spec["owner"],
		"the edit must restate the operator's ownership, not drop it")
	assert.NotContains(t, rec.body, "status", "the CLI never writes status in thin-editor mode")
}

// An absent owner stays absent: the CRD reads a missing spec.owner as
// operator-managed, so omitting the key leaves that meaning intact rather than
// writing an empty string the enum would reject.
func TestApplySpec_EmptyOwnerOmitsTheField(t *testing.T) {
	client, rec := newApplyPatchClient(t, 1)

	_, err := ApplySpec(context.Background(), client, SpecInput{
		Name:          "podinfo",
		Namespace:     "demo",
		Owner:         "",
		ModulePath:    "opmodel.dev/modules/podinfo@v0",
		ModuleVersion: "0.2.0",
	})
	require.NoError(t, err)

	assert.NotContains(t, rec.specOf(t), "owner")
}

// Provenance is stamped only for a local render; a registry render omits the
// annotation so SSA clears any stale one.
func TestApplySpec_ProvenanceAnnotation(t *testing.T) {
	client, rec := newApplyPatchClient(t, 1)
	_, err := ApplySpec(context.Background(), client, SpecInput{
		Name: "podinfo", Namespace: "demo", Owner: OwnerCLI,
		ModulePath: "p", ModuleVersion: "v", SourceLocal: true,
	})
	require.NoError(t, err)

	metadata, ok := rec.body["metadata"].(map[string]any)
	require.True(t, ok)
	annotations, ok := metadata["annotations"].(map[string]any)
	require.True(t, ok, "a local render must stamp the provenance annotation")
	assert.Equal(t, SourceLocal, annotations[AnnotationSource])

	client, rec = newApplyPatchClient(t, 1)
	_, err = ApplySpec(context.Background(), client, SpecInput{
		Name: "podinfo", Namespace: "demo", Owner: OwnerCLI,
		ModulePath: "p", ModuleVersion: "v", SourceLocal: false,
	})
	require.NoError(t, err)

	metadata, _ = rec.body["metadata"].(map[string]any)
	assert.NotContains(t, metadata, "annotations", "a registry render must clear the annotation")
}
