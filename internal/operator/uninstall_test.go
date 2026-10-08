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

// Uninstall builds the error with no Action or Remedy; its message stays as it
// always was.
func TestFinalizerGuardError_DefaultsToUninstall(t *testing.T) {
	err := &FinalizerGuardError{Armed: []ArmedInstance{{Namespace: "default", Name: "jellyfin"}, {Namespace: "media", Name: "sonarr"}}}
	assert.EqualError(t, err,
		"refusing to uninstall: 2 instance(s) still carry the opmodel.dev/cleanup finalizer: default/jellyfin, media/sonarr (use --remove-finalizers to proceed; this orphans their workloads)")
}

func TestFinalizerGuardError_NamesActionTargetAndRemedy(t *testing.T) {
	target := ArmedInstance{Namespace: "opm-operator-system", Name: "opm-operator"}
	err := &FinalizerGuardError{
		Armed:  []ArmedInstance{{Namespace: "default", Name: "hello"}, target},
		Action: "delete opm-operator-system/opm-operator, which deploys the operator",
		Target: target,
		Remedy: "run 'opm operator uninstall --remove-finalizers' to remove that finalizer, orphaning their workloads, then retry",
	}
	assert.EqualError(t, err,
		"refusing to delete opm-operator-system/opm-operator, which deploys the operator: 2 instance(s) still carry the opmodel.dev/cleanup finalizer: "+
			"default/hello, opm-operator-system/opm-operator (the instance being deleted) "+
			"(run 'opm operator uninstall --remove-finalizers' to remove that finalizer, orphaning their workloads, then retry)")
}

func TestOwnInstanceOwnerError_NamesSignalAndRemedy(t *testing.T) {
	err := &OwnInstanceOwnerError{Namespace: "opm-operator-system", Name: "opm-operator", Signal: SignalCoordinates}
	msg := err.Error()
	assert.Contains(t, msg, "refusing to delete opm-operator-system/opm-operator")
	assert.Contains(t, msg, "matched by coordinates")
	assert.Contains(t, msg, "operator-owned")
	assert.Contains(t, msg, "set spec.owner to cli")
	assert.Contains(t, msg, `kubectl patch moduleinstance opm-operator -n opm-operator-system --type=merge -p '{"spec":{"owner":"cli"}}'`)
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

// "Uninstall after a module install": every recorded object except the CRDs
// and the Namespace, then the record.
func TestUninstall_DeletesTheRecordedInventory(t *testing.T) {
	releasedCLI(t)
	fastPolling(t)
	fc := newFakeCluster(t)
	_, err := install(t, fc, &fakeRender{objs: moduleObjects(renderOpts{})}, PlanOptions{})
	require.NoError(t, err)

	result, err := Uninstall(context.Background(), fc.client, UninstallOptions{})
	require.NoError(t, err)
	assert.Empty(t, result.Errors)
	assert.Equal(t, 3, result.Deleted, "ClusterRole, ServiceAccount, Deployment")
	assert.Equal(t, 5, result.LeftBehind, "four CRDs and the Namespace")

	assert.False(t, fc.exists(deploymentGVR, OperatorNamespace, ControllerDeploymentName))
	assert.False(t, fc.exists(clusterRoleGVR, "", "opm-operator-manager-role"))
	assert.True(t, fc.exists(namespaceGVR, "", OperatorNamespace))
	for _, crd := range CRDNames() {
		assert.True(t, fc.exists(crdGVR, "", crd))
	}
	assert.Nil(t, fc.record(), "the record is deleted last")
}

// "Object an older release installed is removed".
func TestUninstall_RemovesWhatAnOlderReleaseRecorded(t *testing.T) {
	releasedCLI(t)
	fastPolling(t)
	fc := newFakeCluster(t)
	_, err := install(t, fc, &fakeRender{objs: moduleObjects(renderOpts{extraRole: "opm-operator-old-role"})}, PlanOptions{})
	require.NoError(t, err)

	_, err = Uninstall(context.Background(), fc.client, UninstallOptions{})
	require.NoError(t, err)
	assert.False(t, fc.exists(clusterRoleGVR, "", "opm-operator-old-role"))
}

// "No record": nothing is deleted.
func TestUninstall_NoRecordDeletesNothing(t *testing.T) {
	fc := newFakeCluster(t, deploymentFixture(true))

	result, err := Uninstall(context.Background(), fc.client, UninstallOptions{})
	require.Error(t, err)
	assert.Nil(t, result)
	var noRecord *NoRecordError
	require.ErrorAs(t, err, &noRecord)
	assert.Contains(t, err.Error(), "opm operator install")
	assert.Empty(t, fc.Writes())
	assert.True(t, fc.exists(deploymentGVR, OperatorNamespace, ControllerDeploymentName))
}

// "Re-running uninstall": absent objects count as deleted, then the record.
func TestUninstall_ReRunDeletesTheRecord(t *testing.T) {
	releasedCLI(t)
	fastPolling(t)
	fc := newFakeCluster(t)
	r := &fakeRender{objs: moduleObjects(renderOpts{})}
	_, err := install(t, fc, r, PlanOptions{})
	require.NoError(t, err)
	for _, obj := range r.objs {
		if obj.GetKind() == kindCustomResourceDefinition || obj.GetKind() == kindNamespace {
			continue
		}
		require.NoError(t, fc.client.ResourceClient(kubernetes.GVRFromUnstructured(obj), obj.GetNamespace()).
			Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{}))
	}

	result, err := Uninstall(context.Background(), fc.client, UninstallOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.Deleted)
	assert.Empty(t, result.Errors)
	assert.Nil(t, fc.record())
}

func TestUninstall_RefusesWhenArmedAndRemoveFinalizersFalse(t *testing.T) {
	inst := moduleInstanceFixture("default", "jellyfin", cleanupFinalizer)
	fc := newFakeCluster(t, inst, operatorRecord("cli", nil))

	result, err := Uninstall(context.Background(), fc.client, UninstallOptions{RemoveFinalizers: false})
	require.Error(t, err)
	assert.Nil(t, result)
	var guardErr *FinalizerGuardError
	require.ErrorAs(t, err, &guardErr)
	assert.Equal(t, []ArmedInstance{{Namespace: "default", Name: "jellyfin"}}, guardErr.Armed)
	assert.Empty(t, fc.Writes())
}

func TestUninstall_RemoveFinalizersPartialFailureDoesNotDeleteResources(t *testing.T) {
	instA := moduleInstanceFixture("default", "jellyfin", cleanupFinalizer)
	instB := moduleInstanceFixture("media", "seerr", cleanupFinalizer)
	fc := newFakeCluster(t, instA, instB, operatorRecord("cli", nil), deploymentFixture(true))
	fc.fake.PrependReactor("patch", "moduleinstances", func(action k8stesting.Action) (bool, runtime.Object, error) {
		patchAction, ok := action.(k8stesting.PatchAction)
		if ok && patchAction.GetName() == "jellyfin" {
			return true, nil, errors.New("transient conflict")
		}
		return false, nil, nil
	})

	result, err := Uninstall(context.Background(), fc.client, UninstallOptions{RemoveFinalizers: true})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "default/jellyfin")

	liveB, getErr := fc.client.Dynamic.Resource(moduleInstanceGVR).Namespace("media").Get(context.Background(), "seerr", metav1.GetOptions{})
	require.NoError(t, getErr)
	assert.Empty(t, liveB.GetFinalizers())
	assert.NotNil(t, fc.record(), "nothing of the operator was deleted")
	assert.True(t, fc.exists(deploymentGVR, OperatorNamespace, ControllerDeploymentName))
}

func TestUninstall_RemoveFinalizersStripsAndProceeds(t *testing.T) {
	releasedCLI(t)
	fastPolling(t)
	fc := newFakeCluster(t, moduleInstanceFixture("default", "jellyfin", cleanupFinalizer))
	_, err := install(t, fc, &fakeRender{objs: moduleObjects(renderOpts{})}, PlanOptions{})
	require.NoError(t, err)

	result, err := Uninstall(context.Background(), fc.client, UninstallOptions{RemoveFinalizers: true})
	require.NoError(t, err)
	assert.Equal(t, 3, result.Deleted)

	live, err := fc.client.Dynamic.Resource(moduleInstanceGVR).Namespace("default").Get(context.Background(), "jellyfin", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Empty(t, live.GetFinalizers())
	assert.False(t, fc.exists(deploymentGVR, OperatorNamespace, ControllerDeploymentName))
}

// "Unreadable recorded object": a recorded object whose read fails with an
// error other than NotFound is a per-object error, the readable objects are
// still deleted, and the record is kept so the uninstall is not reported as
// done and a re-run retries.
func TestUninstall_UnreadableObjectKeepsTheRecord(t *testing.T) {
	releasedCLI(t)
	fastPolling(t)
	fc := newFakeCluster(t)
	_, err := install(t, fc, &fakeRender{objs: moduleObjects(renderOpts{})}, PlanOptions{})
	require.NoError(t, err)

	forbid := true
	fc.fake.PrependReactor("get", "clusterroles", func(action k8stesting.Action) (bool, runtime.Object, error) {
		if !forbid {
			return false, nil, nil
		}
		get := action.(k8stesting.GetAction)
		return true, nil, apierrors.NewForbidden(clusterRoleGVR.GroupResource(), get.GetName(), errors.New("no RBAC"))
	})

	result, err := Uninstall(context.Background(), fc.client, UninstallOptions{})
	forbid = false
	require.NoError(t, err)
	require.Len(t, result.Errors, 1)
	assert.ErrorContains(t, result.Errors[0], "opm-operator-manager-role")
	assert.Equal(t, 2, result.Deleted, "ServiceAccount, Deployment")

	assert.False(t, fc.exists(deploymentGVR, OperatorNamespace, ControllerDeploymentName))
	assert.True(t, fc.exists(clusterRoleGVR, "", "opm-operator-manager-role"), "the unreadable object is not deleted")
	assert.NotNil(t, fc.record(), "the record is kept and still tracks the unreadable object")
}

// "Forbidden record delete": every recorded object is deleted and the delete
// of the record then fails. Uninstall returns the error, with the cause
// reachable, so the command reports no uninstall; the record stays.
func TestUninstall_FailedRecordDeleteIsAnError(t *testing.T) {
	releasedCLI(t)
	fastPolling(t)
	fc := newFakeCluster(t)
	_, err := install(t, fc, &fakeRender{objs: moduleObjects(renderOpts{})}, PlanOptions{})
	require.NoError(t, err)

	fc.fake.PrependReactor("delete", moduleInstanceGVR.Resource, func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(moduleInstanceGVR.GroupResource(), OperatorInstanceName, errors.New("no RBAC"))
	})

	result, err := Uninstall(context.Background(), fc.client, UninstallOptions{})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apierrors.IsForbidden(err), "the cause is reachable through unwrapping")
	assert.Contains(t, err.Error(), OperatorNamespace+"/"+OperatorInstanceName)
	assert.Contains(t, err.Error(), "the record remains")
	assert.Contains(t, err.Error(), "re-running is safe")

	assert.False(t, fc.exists(deploymentGVR, OperatorNamespace, ControllerDeploymentName), "the recorded objects are deleted")
	assert.False(t, fc.exists(clusterRoleGVR, "", "opm-operator-manager-role"))
	assert.NotNil(t, fc.record(), "the record is still there")
}
