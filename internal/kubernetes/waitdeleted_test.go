package kubernetes

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"
	"github.com/open-platform-model/cli/internal/output"
	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"
	"github.com/open-platform-model/library/opm/k8s/lifecycle"
)

var configMapsGVR = schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}

// deletedCM is the deleted ConfigMap ownedCM(name) in "default".
func deletedCM(name string) DeletedObject {
	return DeletedObject{
		Entry: k8sinventory.Entry{Version: "v1", Kind: "ConfigMap", Namespace: "default", Name: name},
		UID:   types.UID("uid-" + name),
	}
}

// terminatingCM is ownedCM(name) as the API server shows it after an
// accepted delete that finalizers hold.
func terminatingCM(name string, finalizers ...string) *unstructured.Unstructured {
	cm := ownedCM(name)
	now := metav1.Now()
	cm.SetDeletionTimestamp(&now)
	cm.SetFinalizers(finalizers)
	return cm
}

// getsOf counts the reads of the ConfigMap name the fake received.
func getsOf(dyn *dynamicfake.FakeDynamicClient, resource, name string) int {
	n := 0
	for _, a := range dyn.Actions() {
		if g, ok := a.(k8stesting.GetAction); ok && a.GetVerb() == "get" && a.GetResource().Resource == resource && g.GetName() == name {
			n++
		}
	}
	return n
}

// waitDeleted runs WaitDeleted with a budget of timeout.
func waitDeleted(t *testing.T, dyn *dynamicfake.FakeDynamicClient, timeout time.Duration, objs ...DeletedObject) error {
	t.Helper()
	shortWaitPoll(t)
	start := time.Now()
	ctx, cancel := context.WithDeadline(context.Background(), start.Add(timeout))
	defer cancel()
	return WaitDeleted(ctx, &Client{Resources: kubetest.Resources(), Dynamic: dyn}, objs, start)
}

// The run records, per step, the UID an accepted delete was sent with, and
// nothing for a step that sent no delete or whose delete was refused.
func TestRunDeletion_RecordsTheUIDOfAnAcceptedDelete(t *testing.T) {
	unmanaged := withUID(owned("v1", "ConfigMap", "b", "default", "", ""), "uid-b")
	b := k8sinventory.Entry{Version: "v1", Kind: "ConfigMap", Namespace: "default", Name: "b"}

	t.Run("accepted delete, then a skip", func(t *testing.T) {
		dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), ownedCM("cm"), unmanaged.DeepCopy())
		run, err := RunDeletion(context.Background(), &Client{Resources: kubetest.Resources(), Dynamic: dyn}, planFor(cmEntry, b), DeletionOptions{})
		require.NoError(t, err)
		require.Len(t, run.Steps, 2)
		assert.Equal(t, types.UID("uid-cm"), run.Steps[0].UID)
		assert.Equal(t, lifecycle.ResultSkipped, run.Steps[1].Outcome.Result)
		assert.Empty(t, run.Steps[1].UID, "a skipped step carries no UID, also right after a delete")
		assert.Equal(t, []DeletedObject{deletedCM("cm")}, DeletedObjects(run), "only the accepted delete is listed")
	})

	t.Run("dry run", func(t *testing.T) {
		dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), ownedCM("cm"))
		run, err := RunDeletion(context.Background(), &Client{Resources: kubetest.Resources(), Dynamic: dyn}, planFor(cmEntry), DeletionOptions{DryRun: true})
		require.NoError(t, err)
		require.Len(t, run.Steps, 1)
		assert.Empty(t, run.Steps[0].UID, "no delete was sent")
	})

	t.Run("refused delete", func(t *testing.T) {
		dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), ownedCM("cm"))
		dyn.PrependReactor("delete", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "cm", nil)
		})
		run, err := RunDeletion(context.Background(), &Client{Resources: kubetest.Resources(), Dynamic: dyn}, planFor(cmEntry), DeletionOptions{})
		require.NoError(t, err)
		require.Len(t, run.Steps, 1)
		assert.Equal(t, lifecycle.ResultFailed, run.Steps[0].Outcome.Result)
		assert.Empty(t, run.Steps[0].UID)
		assert.Empty(t, DeletedObjects(run))
	})
}

// An object that reads NotFound is gone: the wait returns on its first poll.
func TestWaitDeleted_ReturnsAtOnceWhenEveryObjectIsGone(t *testing.T) {
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())

	require.NoError(t, waitDeleted(t, dyn, time.Minute, deletedCM("a"), deletedCM("b")))

	assert.Equal(t, 1, getsOf(dyn, "configmaps", "a"))
	assert.Equal(t, 1, getsOf(dyn, "configmaps", "b"))
}

// Another object under the same name is not the deleted object: the deleted
// one is gone, and the wait does not wait for the new one.
func TestWaitDeleted_AnotherUIDUnderTheNameIsGone(t *testing.T) {
	recreated := withUID(owned("v1", "ConfigMap", "cm", "default", opmlabels.ManagedByCLI, testInstanceUUID), "uid-new")
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), recreated)

	require.NoError(t, waitDeleted(t, dyn, time.Minute, deletedCM("cm")))
}

// Without a recorded UID only NotFound counts: an object under the name is
// still there, whatever its UID.
func TestWaitDeleted_WithoutUIDOnlyNotFoundIsGone(t *testing.T) {
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), ownedCM("cm"))
	obj := deletedCM("cm")
	obj.UID = ""

	err := waitDeleted(t, dyn, 40*time.Millisecond, obj)

	var terminating *TerminatingError
	require.ErrorAs(t, err, &terminating)
	require.Len(t, terminating.Objects, 1)
}

// A terminating object keeps the wait polling; the wait ends when it goes.
func TestWaitDeleted_PollsUntilTheObjectIsGone(t *testing.T) {
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), terminatingCM("cm", "example.io/hold"))
	var gets atomic.Int32
	dyn.PrependReactor("get", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
		if gets.Add(1) == 3 {
			// The finalizer is removed: the object goes before this read.
			require.NoError(t, dyn.Tracker().Delete(configMapsGVR, "default", "cm"))
		}
		return false, nil, nil
	})

	require.NoError(t, waitDeleted(t, dyn, time.Minute, deletedCM("cm")))

	assert.Equal(t, int32(3), gets.Load(), "two reads saw it terminating, the third saw it gone")
}

// At the deadline the error names each object that is left, with its
// finalizers, and not the ones that went.
func TestWaitDeleted_DeadlineNamesWhatIsLeft(t *testing.T) {
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(),
		terminatingCM("held", "example.io/hold", "example.io/other"), terminatingCM("plain"))

	err := waitDeleted(t, dyn, 40*time.Millisecond, deletedCM("gone"), deletedCM("held"), deletedCM("plain"))

	var terminating *TerminatingError
	require.ErrorAs(t, err, &terminating)
	require.Len(t, terminating.Objects, 2)
	held, plain := terminating.Objects[0], terminating.Objects[1]
	assert.Equal(t, "held", held.Entry.Name)
	assert.Equal(t, []string{"example.io/hold", "example.io/other"}, held.Finalizers)
	assert.NoError(t, held.Err)
	assert.Equal(t, "plain", plain.Entry.Name)
	assert.Empty(t, plain.Finalizers)
	assert.Contains(t, err.Error(), "2 deleted resource(s)")
	assert.Greater(t, getsOf(dyn, "configmaps", "held"), 1, "it polled")
	assert.Equal(t, 1, getsOf(dyn, "configmaps", "gone"), "an object that went is not read again")
}

// A read that fails is not an object that is gone: it stays pending and the
// deadline reports the error of the read.
func TestWaitDeleted_FailedReadIsNotGone(t *testing.T) {
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	dyn.PrependReactor("get", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "cm", nil)
	})

	err := waitDeleted(t, dyn, 40*time.Millisecond, deletedCM("cm"))

	var terminating *TerminatingError
	require.ErrorAs(t, err, &terminating)
	require.Len(t, terminating.Objects, 1)
	require.Error(t, terminating.Objects[0].Err)
	assert.True(t, apierrors.IsForbidden(terminating.Objects[0].Err), "the cause is reachable: %v", terminating.Objects[0].Err)
}

// A kind the cluster does not serve cannot be read, so its object is not
// taken as gone either.
func TestWaitDeleted_UnservedKindIsNotGone(t *testing.T) {
	shortWaitPoll(t)
	client := &Client{
		Dynamic:   dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		Resources: kubetest.ResourcesWith(map[schema.GroupVersionKind]kubetest.Outcome{promGVK: {Err: &KindNotServedError{GVK: promGVK}}}),
	}
	obj := DeletedObject{Entry: k8sinventory.Entry{Group: promGVK.Group, Version: promGVK.Version, Kind: promGVK.Kind, Namespace: "default", Name: "main"}}
	start := time.Now()
	ctx, cancel := context.WithDeadline(context.Background(), start.Add(40*time.Millisecond))
	defer cancel()

	err := WaitDeleted(ctx, client, []DeletedObject{obj}, start)

	var terminating *TerminatingError
	require.ErrorAs(t, err, &terminating)
	require.Len(t, terminating.Objects, 1)
	assert.True(t, IsKindNotServed(terminating.Objects[0].Err), "%v", terminating.Objects[0].Err)
}

// Cancellation is not a timeout: the context's error comes back, with no
// claim about what is left.
func TestWaitDeleted_CancellationReturnsTheContextError(t *testing.T) {
	shortWaitPoll(t)
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), terminatingCM("cm", "example.io/hold"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := WaitDeleted(ctx, &Client{Resources: kubetest.Resources(), Dynamic: dyn}, []DeletedObject{deletedCM("cm")}, time.Now())

	require.ErrorIs(t, err, context.Canceled)
	var terminating *TerminatingError
	assert.NotErrorAs(t, err, &terminating)
}

// waitCluster holds the objects of one instance: "held" stays after its
// delete, "gone" goes, "foreign" belongs to another instance and "data" is a
// claim.
func waitCluster() (*dynamicfake.FakeDynamicClient, DeleteOptions) {
	held := terminatingCM("held", "example.io/hold")
	held.SetDeletionTimestamp(nil)
	gone := ownedCM("gone")
	foreign := withUID(owned("v1", "ConfigMap", "foreign", "default", opmlabels.ManagedByCLI, "uuid-other"), "uid-foreign")
	claim := withUID(owned("v1", "PersistentVolumeClaim", "data", "default", opmlabels.ManagedByCLI, testInstanceUUID), "uid-data")

	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), held.DeepCopy(), gone.DeepCopy(), foreign.DeepCopy(), claim.DeepCopy())
	// "held" stays after its accepted delete; every other delete goes through.
	dyn.PrependReactor("delete", "configmaps", func(a k8stesting.Action) (bool, runtime.Object, error) {
		d, ok := a.(k8stesting.DeleteAction)
		return ok && d.GetName() == "held", nil, nil
	})
	return dyn, DeleteOptions{
		InstanceName:          "demo",
		Namespace:             "default",
		InstanceUUID:          testInstanceUUID,
		InventoryLive:         []*unstructured.Unstructured{held, gone, foreign, claim},
		InventoryRecordExists: true,
	}
}

// WaitUntilGone waits for the objects whose delete was accepted and for
// nothing else: the kept claim is never read, the object left behind is read
// once, for its verdict, and the object that went is not listed.
func TestDeleteResult_WaitUntilGoneListsOnlyDeletedObjectsThatAreStillThere(t *testing.T) {
	shortWaitPoll(t)
	dyn, opts := waitCluster()
	client := &Client{Resources: kubetest.Resources(), Dynamic: dyn}

	result, err := Delete(context.Background(), client, opts)
	require.NoError(t, err)
	assert.Empty(t, result.Terminating, "Delete itself does not wait")
	assert.Equal(t, 1, getsOf(dyn, "configmaps", "held"), "and reads nothing after a delete")

	require.NoError(t, result.WaitUntilGone(context.Background(), client, 40*time.Millisecond, output.InstanceLogger("demo")))

	assert.Equal(t, 2, result.Deleted)
	assert.Len(t, result.Kept, 1)
	assert.Len(t, result.LeftBehind, 1)
	assert.Empty(t, result.Errors)
	require.Len(t, result.Terminating, 1)
	assert.Equal(t, "held", result.Terminating[0].Entry.Name)
	assert.Equal(t, []string{"example.io/hold"}, result.Terminating[0].Finalizers)

	assert.Zero(t, getsOf(dyn, "persistentvolumeclaims", "data"), "a kept claim is not waited for")
	assert.Equal(t, 1, getsOf(dyn, "configmaps", "foreign"), "an object left behind is read for its verdict only")
	assert.Greater(t, getsOf(dyn, "configmaps", "held"), 1)
}

// When every deleted object went, nothing is listed and the wait returns on
// its first poll.
func TestDeleteResult_WaitUntilGoneWhenEverythingWent(t *testing.T) {
	shortWaitPoll(t)
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), ownedCM("cm"))
	client := &Client{Resources: kubetest.Resources(), Dynamic: dyn}
	result, err := Delete(context.Background(), client, DeleteOptions{
		InstanceName: "demo", Namespace: "default", InstanceUUID: testInstanceUUID,
		InventoryLive: []*unstructured.Unstructured{ownedCM("cm")}, InventoryRecordExists: true,
	})
	require.NoError(t, err)

	require.NoError(t, result.WaitUntilGone(context.Background(), client, time.Minute, output.InstanceLogger("demo")))

	assert.Empty(t, result.Terminating)
	assert.Equal(t, 2, getsOf(dyn, "configmaps", "cm"), "the plan's read and one poll")
}

// A canceled wait is an error, not a list of terminating objects.
func TestDeleteResult_WaitUntilGoneCanceled(t *testing.T) {
	shortWaitPoll(t)
	dyn, opts := waitCluster()
	client := &Client{Resources: kubetest.Resources(), Dynamic: dyn}
	result, err := Delete(context.Background(), client, opts)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = result.WaitUntilGone(ctx, client, time.Minute, output.InstanceLogger("demo"))

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, result.Terminating)
}
