package apply

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"
	"github.com/open-platform-model/cli/internal/output"
)

// deleteRecordedWaiting runs DeleteRecorded with Wait set and a short budget.
func deleteRecordedWaiting(t *testing.T, dyn *dynamicfake.FakeDynamicClient, rec *inventory.Record, live []*unstructured.Unstructured) *kubernetes.DeleteResult {
	t.Helper()
	return deleteRecordedWith(t, dyn, rec, live, false)
}

func deleteRecordedWith(t *testing.T, dyn *dynamicfake.FakeDynamicClient, rec *inventory.Record, live []*unstructured.Unstructured, dryRun bool) *kubernetes.DeleteResult {
	t.Helper()
	prev := kubernetes.WaitPollInterval
	kubernetes.WaitPollInterval = 5 * time.Millisecond
	t.Cleanup(func() { kubernetes.WaitPollInterval = prev })

	result, err := DeleteRecorded(context.Background(), DeleteRequest{
		Client:       &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: dyn},
		InstanceName: "demo",
		Namespace:    "apps",
		Record:       rec,
		Live:         live,
		DryRun:       dryRun,
		Wait:         true,
		Timeout:      40 * time.Millisecond,
		Log:          output.InstanceLogger("demo"),
	})
	require.NoError(t, err)
	return result
}

// With Wait, a deleted object that is still there when the budget ends holds
// the record: it goes on tracking what still exists, and the result names
// the object for the caller's report.
func TestDeleteRecorded_WaitHoldsTheRecordWhileAnObjectIsTerminating(t *testing.T) {
	captureLog(t)
	dyn, rec, live := planCluster([]string{"held", "plain"}, "held", "plain")
	dyn.PrependReactor("delete", "configmaps", func(a k8stesting.Action) (bool, runtime.Object, error) {
		d, ok := a.(k8stesting.DeleteAction)
		return ok && d.GetName() == "held", nil, nil // accepted; "held" stays, as under a finalizer
	})

	result := deleteRecordedWaiting(t, dyn, rec, live)

	assert.Equal(t, 2, result.Deleted)
	assert.Empty(t, result.Errors)
	require.Len(t, result.Terminating, 1)
	assert.Equal(t, "held", result.Terminating[0].Entry.Name)
	assert.True(t, recordExists(dyn), "the record is kept while a deleted object still exists")
}

// With Wait, the record is deleted once every deleted object is gone.
func TestDeleteRecorded_WaitReleasesTheRecordWhenTheObjectsAreGone(t *testing.T) {
	logBuf := captureLog(t)
	dyn, rec, live := planCluster([]string{"a", "b"}, "a", "b")

	result := deleteRecordedWaiting(t, dyn, rec, live)

	assert.Equal(t, 2, result.Deleted)
	assert.Empty(t, result.Terminating)
	assert.False(t, recordExists(dyn))
	assert.Contains(t, logBuf.String(), "waiting for 2 deleted resource(s) to be gone")
}

// gets counts the reads of ConfigMaps the cluster received.
func gets(dyn *dynamicfake.FakeDynamicClient) int {
	n := 0
	for _, a := range dyn.Actions() {
		if a.GetVerb() == "get" && a.GetResource().Resource == "configmaps" {
			n++
		}
	}
	return n
}

// The kept, left-behind and error lines come before the wait starts, so the
// line that says the command is waiting is the last one before it goes quiet.
func TestDeleteRecorded_WaitLineComesAfterTheResourceLines(t *testing.T) {
	logBuf := captureLog(t)
	dyn, rec, live := planCluster([]string{"a", "foreign"}, "a", "foreign")
	live[1].SetLabels(map[string]string{}) // no longer OPM-managed: left behind
	require.NoError(t, dyn.Tracker().Update(configMaps, live[1].DeepCopy(), "apps"))

	result := deleteRecordedWaiting(t, dyn, rec, live)

	require.Len(t, result.LeftBehind, 1)
	log := logBuf.String()
	leftBehind := strings.Index(log, output.StatusLeftBehind)
	waiting := strings.Index(log, "waiting for 1 deleted resource(s) to be gone")
	require.NotEqual(t, -1, leftBehind, log)
	require.NotEqual(t, -1, waiting, log)
	assert.Less(t, leftBehind, waiting)
}

// A dry run sends no delete, so Wait reads nothing more and holds nothing.
func TestDeleteRecorded_WaitDoesNothingOnADryRun(t *testing.T) {
	logBuf := captureLog(t)
	dyn, rec, live := planCluster([]string{"held"}, "held")

	result := deleteRecordedWith(t, dyn, rec, live, true)

	assert.Empty(t, result.Terminating)
	assert.Equal(t, 1, gets(dyn), "the plan's read only")
	assert.NotContains(t, logBuf.String(), "waiting for")
}

// A run in which an object failed does not wait: it fails already, keeps the
// record, and its re-run waits.
func TestDeleteRecorded_WaitDoesNothingAfterAFailedObject(t *testing.T) {
	logBuf := captureLog(t)
	dyn, rec, live := planCluster([]string{"bad", "held"}, "bad", "held")
	dyn.PrependReactor("delete", "configmaps", func(a k8stesting.Action) (bool, runtime.Object, error) {
		if d, ok := a.(k8stesting.DeleteAction); ok && d.GetName() == "bad" {
			return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "bad", nil)
		}
		return true, nil, nil // "held" is accepted and stays
	})

	result := deleteRecordedWaiting(t, dyn, rec, live)

	assert.Len(t, result.Errors, 1)
	assert.Empty(t, result.Terminating)
	assert.Equal(t, 2, gets(dyn), "one read per object for its verdict, none after")
	assert.NotContains(t, logBuf.String(), "waiting for")
	assert.True(t, recordExists(dyn))
}

// A canceled wait is an error of the delete: no result, and the record stays.
func TestDeleteRecorded_WaitCanceledKeepsTheRecord(t *testing.T) {
	captureLog(t)
	dyn, rec, live := planCluster([]string{"held"}, "held")
	ctx, cancel := context.WithCancel(context.Background())
	dyn.PrependReactor("delete", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
		cancel() // interrupted while the delete is sent
		return true, nil, nil
	})

	result, err := DeleteRecorded(ctx, DeleteRequest{
		Client:       &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: dyn},
		InstanceName: "demo", Namespace: "apps", Record: rec, Live: live,
		Wait: true, Timeout: time.Minute, Log: output.InstanceLogger("demo"),
	})

	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, result)
	assert.True(t, recordExists(dyn))
}
