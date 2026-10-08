package operator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"

	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"
)

var widgetGVK = schema.GroupVersionKind{Group: "example.io", Version: "v1", Kind: "Widget"}

func widget() *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(widgetGVK)
	obj.SetNamespace("default")
	obj.SetName("main")
	return obj
}

// widgetClient is an empty fake cluster that resolves Widget as outcome says.
func widgetClient(outcome kubetest.Outcome) *kubernetes.Client {
	return &kubernetes.Client{
		Dynamic:   fakedynamic.NewSimpleDynamicClient(runtime.NewScheme()),
		Resources: kubetest.ResourcesWith(map[schema.GroupVersionKind]kubetest.Outcome{widgetGVK: outcome}),
	}
}

var (
	widgetNotServed = kubetest.Outcome{Err: &kubernetes.KindNotServedError{GVK: widgetGVK}}
	widgetDown      = kubetest.Outcome{Err: apierrors.NewServiceUnavailable("discovery is down")}
)

// Before the first install the cluster does not serve the kinds of the
// operator's own definitions: a rendered object of such a kind does not
// exist. A failed discovery request leaves the question open.
func TestTerminatingObjects_ResolvesByDiscovery(t *testing.T) {
	ctx := context.Background()

	terminating, err := terminatingObjects(ctx, widgetClient(widgetNotServed), []*unstructured.Unstructured{widget()})
	require.NoError(t, err)
	assert.Empty(t, terminating)

	_, err = terminatingObjects(ctx, widgetClient(widgetDown), []*unstructured.Unstructured{widget()})
	require.Error(t, err)
	assert.True(t, apierrors.IsServiceUnavailable(err))
}

func TestGetLive_ResolvesByDiscovery(t *testing.T) {
	ctx := context.Background()
	var readErr *MigrationReadError

	live, err := getLive(ctx, widgetClient(widgetNotServed), widgetGVK, "default", "main", true)
	require.NoError(t, err, "a rendered object of a kind that is not served yet does not exist")
	assert.Nil(t, live)

	_, err = getLive(ctx, widgetClient(widgetNotServed), widgetGVK, "default", "main", false)
	require.ErrorAs(t, err, &readErr, "a proof-list kind that is not served is a failed read, not an absent object")
	assert.True(t, kubernetes.IsKindNotServed(err))

	for _, unservedIsAbsent := range []bool{true, false} {
		_, err = getLive(ctx, widgetClient(widgetDown), widgetGVK, "default", "main", unservedIsAbsent)
		require.ErrorAs(t, err, &readErr)
		assert.True(t, apierrors.IsServiceUnavailable(err))
	}
}

// A delete the migration proved is never "done" when its kind cannot be
// resolved.
func TestDeleteProven_UnresolvedKindStops(t *testing.T) {
	for name, outcome := range map[string]kubetest.Outcome{"kind not served": widgetNotServed, "discovery down": widgetDown} {
		t.Run(name, func(t *testing.T) {
			err := deleteProven(context.Background(), widgetClient(outcome), widget())
			var stopped *MigrationStoppedError
			require.ErrorAs(t, err, &stopped)
		})
	}
}
