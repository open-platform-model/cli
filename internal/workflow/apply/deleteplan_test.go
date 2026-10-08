package apply

import (
	"context"
	"errors"
	"strings"
	"testing"

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
	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"
)

var configMaps = schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}

// planCluster is a fake cluster holding the ModuleInstance apps/demo and the
// given OPM-managed ConfigMaps, with the record that lists entries.
func planCluster(entries []string, live ...string) (*dynamicfake.FakeDynamicClient, *inventory.Record, []*unstructured.Unstructured) {
	objs := make([]runtime.Object, 0, 1+len(live))
	objs = append(objs, &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": inventory.APIVersionModuleInstance, "kind": inventory.KindModuleInstance,
		"metadata": map[string]any{"name": "demo", "namespace": "apps"},
	}})
	liveObjs := make([]*unstructured.Unstructured, 0, len(live))
	for _, name := range live {
		cm := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1", "kind": "ConfigMap",
			"metadata": map[string]any{"name": name, "namespace": "apps", "uid": "uid-" + name, "labels": map[string]any{
				opmlabels.ManagedBy: opmlabels.ManagedByCLI,
			}},
		}}
		objs = append(objs, cm.DeepCopy())
		liveObjs = append(liveObjs, cm)
	}
	rec := &inventory.Record{Name: "demo", Namespace: "apps", Owner: inventory.OwnerCLI}
	for _, name := range entries {
		rec.Inventory.Entries = append(rec.Inventory.Entries, k8sinventory.Entry{Version: "v1", Kind: "ConfigMap", Namespace: "apps", Name: name})
	}
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{inventory.ModuleInstanceGVR: "ModuleInstanceList"}, objs...)
	return dyn, rec, liveObjs
}

func deleteRecorded(t *testing.T, dyn *dynamicfake.FakeDynamicClient, rec *inventory.Record, live []*unstructured.Unstructured, unreadable []kubernetes.UnreadableResource) *kubernetes.DeleteResult {
	t.Helper()
	result, err := DeleteRecorded(context.Background(), DeleteRequest{
		Client:       &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: dyn},
		InstanceName: "demo",
		Namespace:    "apps",
		Record:       rec,
		Live:         live,
		Unreadable:   unreadable,
		Log:          output.InstanceLogger("demo"),
	})
	require.NoError(t, err)
	return result
}

func recordExists(dyn *dynamicfake.FakeDynamicClient) bool {
	_, err := dyn.Tracker().Get(inventory.ModuleInstanceGVR, "apps", "demo")
	return err == nil
}

// The record is deleted on the plan's release verdict and on nothing else:
// one failed delete holds it although every other object went.
func TestDeleteRecorded_FailedStepHoldsTheRecord(t *testing.T) {
	captureLog(t)
	dyn, rec, live := planCluster([]string{"a", "b"}, "a", "b")
	dyn.PrependReactor("delete", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		if action.(k8stesting.DeleteAction).GetName() != "a" {
			return false, nil, nil
		}
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "a", errors.New("denied"))
	})

	result := deleteRecorded(t, dyn, rec, live, nil)

	require.Len(t, result.Errors, 1)
	assert.Equal(t, 1, result.Deleted)
	_, err := dyn.Tracker().Get(configMaps, "apps", "b")
	assert.True(t, apierrors.IsNotFound(err), "the other object is still deleted")
	assert.True(t, recordExists(dyn), "a failed object holds the record")
}

// A run with nothing failed releases the record, also when the plan left an
// object in place.
func TestDeleteRecorded_LeftBehindReleasesTheRecord(t *testing.T) {
	captureLog(t)
	dyn, rec, live := planCluster([]string{"a", "foreign"}, "a", "foreign")
	foreign, err := dyn.Tracker().Get(configMaps, "apps", "foreign")
	require.NoError(t, err)
	foreign.(*unstructured.Unstructured).SetLabels(nil)
	require.NoError(t, dyn.Tracker().Update(configMaps, foreign, "apps"))

	result := deleteRecorded(t, dyn, rec, live, nil)

	assert.Empty(t, result.Errors)
	require.Len(t, result.LeftBehind, 1)
	assert.Equal(t, "foreign", result.LeftBehind[0].Name)
	assert.False(t, recordExists(dyn), "left-behind objects do not hold the record")
}

// A recorded object that was already gone when the instance was read. The
// plan is built from the record, so it reads the entry and finds it absent.
//
// Before the plan: the entry was not in the live list, and nothing was
// printed for it at any level.
// With the plan: the debug line "resource already gone" names it. Nothing
// above debug level changes, it is not counted, and the record is deleted.
func TestDeleteRecorded_EntryGoneAtDiscovery(t *testing.T) {
	output.SetupLogging(output.LogConfig{Verbose: true})
	t.Cleanup(func() { output.SetupLogging(output.LogConfig{}) })
	logs := captureLog(t)
	dyn, rec, live := planCluster([]string{"web", "gone"}, "web")

	result := deleteRecorded(t, dyn, rec, live, nil)

	assert.Equal(t, 1, result.Deleted)
	assert.Empty(t, result.Errors)
	assert.Empty(t, result.LeftBehind)
	assert.False(t, recordExists(dyn))

	var goneLines []string
	for _, line := range strings.Split(logs.String(), "\n") {
		if strings.Contains(line, "name=gone") {
			goneLines = append(goneLines, line)
		}
	}
	require.Len(t, goneLines, 1, "one line for the absent entry:\n%s", logs.String())
	assert.Contains(t, goneLines[0], "DEBU")
	assert.Contains(t, goneLines[0], "resource already gone")
}

// An entry the discovery read could not read is not read again: it is
// reported first, before any deleted line, as before the plan, and it holds
// the record.
func TestDeleteRecorded_UnreadableIsReportedFirstAndHoldsTheRecord(t *testing.T) {
	logs := captureLog(t)
	dyn, rec, live := planCluster([]string{"web", "locked"}, "web", "locked")
	readErr := apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "locked", errors.New("denied"))
	unreadable := []kubernetes.UnreadableResource{{Kind: "ConfigMap", Namespace: "apps", Name: "locked", Err: readErr}}

	result := deleteRecorded(t, dyn, rec, live[:1], unreadable)

	require.Len(t, result.Errors, 1)
	assert.Equal(t, 1, result.Deleted)
	assert.True(t, recordExists(dyn), "an unreadable object holds the record")
	_, err := dyn.Tracker().Get(configMaps, "apps", "locked")
	assert.NoError(t, err, "the unreadable object is not deleted")
	for _, a := range dyn.Actions() {
		if g, ok := a.(k8stesting.GetAction); ok {
			assert.NotEqual(t, "locked", g.GetName(), "no second read")
		}
	}

	out := logs.String()
	reading := strings.Index(out, "reading ConfigMap/locked")
	deleted := strings.Index(out, "deleted")
	require.GreaterOrEqual(t, reading, 0, out)
	require.GreaterOrEqual(t, deleted, 0, out)
	assert.Less(t, reading, deleted, "the unreadable object's line keeps its place before the deleted lines:\n%s", out)
}

// A read that answers with another object than the one asked for.
//
// Before the plan: could not be told apart; the object read was judged as
// if it were the tracked one.
// With the plan: the step fails with the line below, the object is not
// deleted, and the record is kept.
func TestDeleteRecorded_ReadOfAnotherObject(t *testing.T) {
	logs := captureLog(t)
	dyn, rec, live := planCluster([]string{"web"}, "web")
	other := live[0].DeepCopy()
	other.SetName("other")
	dyn.PrependReactor("get", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, other, nil
	})

	result := deleteRecorded(t, dyn, rec, live, nil)

	require.Len(t, result.Errors, 1)
	assert.Contains(t, logs.String(), "reading ConfigMap/web: reading ConfigMap/apps/web returned ConfigMap/apps/other; not deleted")
	assert.True(t, recordExists(dyn))
	for _, a := range dyn.Actions() {
		_, isDelete := a.(k8stesting.DeleteAction)
		assert.False(t, isDelete, "nothing is deleted")
	}
}
