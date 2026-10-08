package kubernetes

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
)

func diffConfigMap(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]any{"name": name, "namespace": "apps"},
		"data":     map[string]any{"key": "value"},
	}}
}

// A live read that fails with anything but NotFound is an error on the
// result, not a skipped resource: the resource is in no difference count, and
// the readable and the absent resources are still compared.
func TestDiff_FailedReadIsAnError(t *testing.T) {
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), diffConfigMap("same"), diffConfigMap("denied"))
	dyn.PrependReactor("get", "configmaps", func(a k8stesting.Action) (bool, runtime.Object, error) {
		if a.(k8stesting.GetAction).GetName() == "denied" {
			return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "denied", errors.New("no read access"))
		}
		return false, nil, nil
	})

	result, err := Diff(context.Background(), &Client{Dynamic: dyn},
		[]*unstructured.Unstructured{diffConfigMap("same"), diffConfigMap("denied"), diffConfigMap("new")},
		"demo", NewComparer())
	require.NoError(t, err)

	require.Len(t, result.Errors, 1)
	failed := result.Errors[0]
	assert.Equal(t, "ConfigMap", failed.Kind)
	assert.Equal(t, "apps", failed.Namespace)
	assert.Equal(t, "denied", failed.Name)
	assert.True(t, apierrors.IsForbidden(failed), "the read error is reachable through unwrapping")
	assert.Contains(t, failed.Error(), "ConfigMap/apps/denied")

	assert.Equal(t, 1, result.Unchanged)
	assert.Equal(t, 1, result.Added, "NotFound still means a new resource")
	assert.Equal(t, 0, result.Modified)
	assert.Len(t, result.Resources, 2, "the unreadable resource is in no state")
}

type failingComparer struct{ err error }

func (c failingComparer) Compare(_, _ *unstructured.Unstructured) (string, error) { return "", c.err }

// A comparison that fails is an error on the result too.
func TestDiff_FailedComparisonIsAnError(t *testing.T) {
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), diffConfigMap("web"))
	cause := errors.New("cannot marshal")

	result, err := Diff(context.Background(), &Client{Dynamic: dyn},
		[]*unstructured.Unstructured{diffConfigMap("web")}, "demo", failingComparer{err: cause})
	require.NoError(t, err)

	require.Len(t, result.Errors, 1)
	assert.Equal(t, "web", result.Errors[0].Name)
	assert.ErrorIs(t, result.Errors[0], cause)
	assert.Empty(t, result.Resources)
}
