package operator

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/cli/internal/kubernetes"
)

func moduleInstanceFixture(namespace, name string, finalizers ...string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "opmodel.dev/v1alpha1",
		"kind":       "ModuleInstance",
		"metadata": map[string]any{
			"name":      name,
			"namespace": namespace,
		},
	}}
	if len(finalizers) > 0 {
		fs := make([]any, len(finalizers))
		for i, f := range finalizers {
			fs[i] = f
		}
		_ = unstructured.SetNestedSlice(obj.Object, fs, "metadata", "finalizers")
	}
	return obj
}

func TestCheckFinalizerGuard_FindsOnlyArmedInstances(t *testing.T) {
	armedInst := moduleInstanceFixture("default", "jellyfin", cleanupFinalizer, "example.com/foreign")
	cleanInst := moduleInstanceFixture("default", "redis")
	client := fakeClientWith(armedInst, cleanInst)

	armed, err := CheckFinalizerGuard(context.Background(), client)
	require.NoError(t, err)
	require.Len(t, armed, 1)
	assert.Equal(t, ArmedInstance{Namespace: "default", Name: "jellyfin"}, armed[0])
}

func TestCheckFinalizerGuard_NoInstancesReturnsEmpty(t *testing.T) {
	client := fakeClientWith()
	armed, err := CheckFinalizerGuard(context.Background(), client)
	require.NoError(t, err)
	assert.Empty(t, armed)
}

func TestCheckFinalizerGuard_ListFailureFailsClosed(t *testing.T) {
	client := fakeClientWith()
	fake, ok := client.Dynamic.(interface {
		PrependReactor(verb, resource string, reaction k8stesting.ReactionFunc)
	})
	require.True(t, ok)
	fake.PrependReactor("list", "moduleinstances", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("forbidden: user cannot list moduleinstances")
	})

	_, err := CheckFinalizerGuard(context.Background(), client)
	require.Error(t, err)
	assert.ErrorContains(t, err, "listing moduleinstances")
}

func TestFinalizerGuardError_NamesArmedInstances(t *testing.T) {
	err := &FinalizerGuardError{Armed: []ArmedInstance{{Namespace: "default", Name: "jellyfin"}}}
	assert.ErrorContains(t, err, "default/jellyfin")
	assert.ErrorContains(t, err, "--remove-finalizers")
	assert.ErrorContains(t, err, cleanupFinalizer)
}

func TestRemoveCleanupFinalizer_RemovesOnlyTheCleanupFinalizer(t *testing.T) {
	inst := moduleInstanceFixture("default", "jellyfin", cleanupFinalizer, "example.com/foreign")
	client := fakeClientWith(inst)

	armed := []ArmedInstance{{Namespace: "default", Name: "jellyfin"}}
	err := RemoveCleanupFinalizer(context.Background(), client, armed)
	require.NoError(t, err)

	live, err := client.Dynamic.Resource(moduleInstanceGVR).Namespace("default").Get(context.Background(), "jellyfin", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com/foreign"}, live.GetFinalizers())
}

func TestRemoveCleanupFinalizer_MissingFinalizerIsANoop(t *testing.T) {
	inst := moduleInstanceFixture("default", "jellyfin", "example.com/foreign")
	client := fakeClientWith(inst)

	armed := []ArmedInstance{{Namespace: "default", Name: "jellyfin"}}
	err := RemoveCleanupFinalizer(context.Background(), client, armed)
	require.NoError(t, err)

	live, err := client.Dynamic.Resource(moduleInstanceGVR).Namespace("default").Get(context.Background(), "jellyfin", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com/foreign"}, live.GetFinalizers())
}

func TestRemoveCleanupFinalizer_MultipleInstances(t *testing.T) {
	instA := moduleInstanceFixture("default", "jellyfin", cleanupFinalizer)
	instB := moduleInstanceFixture("media", "seerr", cleanupFinalizer, "example.com/foreign")
	client := fakeClientWith(instA, instB)

	armed := []ArmedInstance{
		{Namespace: "default", Name: "jellyfin"},
		{Namespace: "media", Name: "seerr"},
	}
	require.NoError(t, RemoveCleanupFinalizer(context.Background(), client, armed))

	liveA, err := client.Dynamic.Resource(moduleInstanceGVR).Namespace("default").Get(context.Background(), "jellyfin", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Empty(t, liveA.GetFinalizers())

	liveB, err := client.Dynamic.Resource(moduleInstanceGVR).Namespace("media").Get(context.Background(), "seerr", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com/foreign"}, liveB.GetFinalizers())
}

func TestRemoveCleanupFinalizer_ContinuesPastAFailureAndReturnsCombinedError(t *testing.T) {
	instA := moduleInstanceFixture("default", "jellyfin", cleanupFinalizer)
	instB := moduleInstanceFixture("media", "seerr", cleanupFinalizer)
	client := fakeClientWith(instA, instB)

	fake, ok := client.Dynamic.(interface {
		PrependReactor(verb, resource string, reaction k8stesting.ReactionFunc)
	})
	require.True(t, ok)
	fake.PrependReactor("patch", "moduleinstances", func(action k8stesting.Action) (bool, runtime.Object, error) {
		patchAction, ok := action.(k8stesting.PatchAction)
		if ok && patchAction.GetName() == "jellyfin" {
			return true, nil, errors.New("transient conflict")
		}
		return false, nil, nil
	})

	armed := []ArmedInstance{
		{Namespace: "default", Name: "jellyfin"},
		{Namespace: "media", Name: "seerr"},
	}
	err := RemoveCleanupFinalizer(context.Background(), client, armed)
	require.Error(t, err)
	assert.ErrorContains(t, err, "default/jellyfin")

	// The failing instance is untouched.
	liveA, getErr := client.Dynamic.Resource(moduleInstanceGVR).Namespace("default").Get(context.Background(), "jellyfin", metav1.GetOptions{})
	require.NoError(t, getErr)
	assert.Equal(t, []string{cleanupFinalizer}, liveA.GetFinalizers())

	// The later instance still got processed despite the earlier failure —
	// the loop no longer aborts on the first error.
	liveB, getErr := client.Dynamic.Resource(moduleInstanceGVR).Namespace("media").Get(context.Background(), "seerr", metav1.GetOptions{})
	require.NoError(t, getErr)
	assert.Empty(t, liveB.GetFinalizers())
}

func TestUninstall_RemoveFinalizersPartialFailureDoesNotDeleteResources(t *testing.T) {
	instA := moduleInstanceFixture("default", "jellyfin", cleanupFinalizer)
	instB := moduleInstanceFixture("media", "seerr", cleanupFinalizer)
	manifest, err := EmbeddedManifest()
	require.NoError(t, err)
	plan := UninstallPlan(manifest)
	client := fakeClientWith(append([]*unstructured.Unstructured{instA, instB}, plan...)...)

	fake, ok := client.Dynamic.(interface {
		PrependReactor(verb, resource string, reaction k8stesting.ReactionFunc)
	})
	require.True(t, ok)
	fake.PrependReactor("patch", "moduleinstances", func(action k8stesting.Action) (bool, runtime.Object, error) {
		patchAction, ok := action.(k8stesting.PatchAction)
		if ok && patchAction.GetName() == "jellyfin" {
			return true, nil, errors.New("transient conflict")
		}
		return false, nil, nil
	})

	result, err := Uninstall(context.Background(), client, UninstallOptions{RemoveFinalizers: true})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "default/jellyfin")

	// seerr was stripped — processing continued past jellyfin's failure.
	liveB, getErr := client.Dynamic.Resource(moduleInstanceGVR).Namespace("media").Get(context.Background(), "seerr", metav1.GetOptions{})
	require.NoError(t, getErr)
	assert.Empty(t, liveB.GetFinalizers())

	// The operator's own resources were never touched: Uninstall must not
	// proceed to its delete loop while any armed instance failed to strip.
	_, err = client.Dynamic.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}).
		Namespace("opm-operator-system").Get(context.Background(), "opm-operator-controller-manager", metav1.GetOptions{})
	require.NoError(t, err, "Deployment should still exist")
}

func TestUninstall_IsIdempotentWhenNothingLeftToDelete(t *testing.T) {
	client := fakeClientWith() // Nothing pre-seeded — simulates "already uninstalled".

	result, err := Uninstall(context.Background(), client, UninstallOptions{})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Errors)
}

func TestUninstall_RefusesWhenArmedAndRemoveFinalizersFalse(t *testing.T) {
	inst := moduleInstanceFixture("default", "jellyfin", cleanupFinalizer)
	client := fakeClientWith(inst)

	result, err := Uninstall(context.Background(), client, UninstallOptions{RemoveFinalizers: false})
	require.Error(t, err)
	assert.Nil(t, result)
	var guardErr *FinalizerGuardError
	require.ErrorAs(t, err, &guardErr)
	assert.Equal(t, []ArmedInstance{{Namespace: "default", Name: "jellyfin"}}, guardErr.Armed)

	// Nothing else was touched: the instance still exists with the finalizer intact.
	live, err := client.Dynamic.Resource(moduleInstanceGVR).Namespace("default").Get(context.Background(), "jellyfin", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, []string{cleanupFinalizer}, live.GetFinalizers())
}

func TestUninstall_RemoveFinalizersStripsAndProceeds(t *testing.T) {
	inst := moduleInstanceFixture("default", "jellyfin", cleanupFinalizer)
	manifest, err := EmbeddedManifest()
	require.NoError(t, err)
	plan := UninstallPlan(manifest)
	client := fakeClientWith(append([]*unstructured.Unstructured{inst}, plan...)...)

	result, err := Uninstall(context.Background(), client, UninstallOptions{RemoveFinalizers: true})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, len(plan), result.Deleted)
	assert.Empty(t, result.Errors)

	live, err := client.Dynamic.Resource(moduleInstanceGVR).Namespace("default").Get(context.Background(), "jellyfin", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Empty(t, live.GetFinalizers())

	_, err = client.Dynamic.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}).
		Namespace("opm-operator-system").Get(context.Background(), "opm-operator-controller-manager", metav1.GetOptions{})
	assert.True(t, apierrors.IsNotFound(err))
}

func TestUninstall_NoArmedInstancesDeletesEverythingInPlan(t *testing.T) {
	manifest, err := EmbeddedManifest()
	require.NoError(t, err)
	plan := UninstallPlan(manifest)
	client := fakeClientWith(plan...)

	result, err := Uninstall(context.Background(), client, UninstallOptions{})
	require.NoError(t, err)
	assert.Equal(t, len(plan), result.Deleted)
	assert.Empty(t, result.Errors)
}

func TestUninstall_NeverTargetsCRDsOrNamespace(t *testing.T) {
	manifest, err := EmbeddedManifest()
	require.NoError(t, err)
	// Seed the full manifest, including CRDs and the Namespace, to prove
	// Uninstall leaves them alone even though they're present on the "cluster".
	client := fakeClientWith(manifest...)

	result, err := Uninstall(context.Background(), client, UninstallOptions{})
	require.NoError(t, err)
	assert.Empty(t, result.Errors)

	for _, obj := range manifest {
		if obj.GetKind() != kindCustomResourceDefinition && obj.GetKind() != kindNamespace {
			continue
		}
		live, err := client.Dynamic.Resource(kubernetes.GVRFromUnstructured(obj)).Namespace(obj.GetNamespace()).Get(context.Background(), obj.GetName(), metav1.GetOptions{})
		require.NoError(t, err, "CRD/Namespace should not have been deleted")
		assert.NotNil(t, live)
	}
}
