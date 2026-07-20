package operator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestReadinessTargets_SelectsCRDsAndTheControllerDeployment(t *testing.T) {
	manifest, err := EmbeddedManifest()
	require.NoError(t, err)

	targets := readinessTargets(manifest)
	require.NotEmpty(t, targets)

	var crds, deployments int
	for _, obj := range targets {
		switch obj.GetKind() {
		case kindCustomResourceDefinition:
			crds++
		case kindDeployment:
			deployments++
		default:
			t.Fatalf("unexpected readiness target kind %q — supporting objects add noise to the refusal message", obj.GetKind())
		}
	}

	assert.Positive(t, crds, "the ModuleInstance and Platform CRDs define whether the operator can serve")
	assert.Equal(t, 1, deployments, "expected exactly the controller Deployment")
}

func TestCheckReady_AbsentOperatorIsNotReady(t *testing.T) {
	// An empty cluster: nothing the readiness targets name exists.
	client := fakeClientWith()

	err := CheckReady(context.Background(), client)
	require.Error(t, err)

	var notReady *NotReadyError
	require.ErrorAs(t, err, &notReady)
	assert.NotEmpty(t, notReady.Pending)
	assert.Contains(t, err.Error(), "opm operator install")
}

func TestNotReadyError_IncludesTheCallerHint(t *testing.T) {
	err := &NotReadyError{
		Pending: []string{"Deployment/opm-operator-controller-manager in opm-operator-system"},
		Hint:    "deleting now would wedge the instance",
	}

	assert.Contains(t, err.Error(), "not ready")
	assert.Contains(t, err.Error(), "opm-operator-controller-manager")
	assert.Contains(t, err.Error(), "deleting now would wedge the instance")
	assert.Contains(t, err.Error(), "opm operator install")
}

func TestDescribeObjectList_NamespacedAndClusterScoped(t *testing.T) {
	objs := []*unstructured.Unstructured{
		{Object: map[string]any{"kind": "Deployment", "metadata": map[string]any{"name": "mgr", "namespace": "sys"}}},
		{Object: map[string]any{"kind": "CustomResourceDefinition", "metadata": map[string]any{"name": "moduleinstances.opmodel.dev"}}},
	}

	described := describeObjectList(objs)
	assert.Equal(t, []string{"Deployment/mgr in sys", "CustomResourceDefinition/moduleinstances.opmodel.dev"}, described)
}
