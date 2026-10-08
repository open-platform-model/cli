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
	"k8s.io/apimachinery/pkg/types"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"
	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"
	"github.com/open-platform-model/library/opm/k8s/lifecycle"
	"github.com/open-platform-model/library/opm/k8s/object"
	"github.com/open-platform-model/library/opm/k8s/ownership"
)

// deleteActions returns the delete calls the fake dynamic client received.
func deleteActions(dyn *dynamicfake.FakeDynamicClient) []k8stesting.DeleteAction {
	var out []k8stesting.DeleteAction
	for _, a := range dyn.Actions() {
		if d, ok := a.(k8stesting.DeleteAction); ok {
			out = append(out, d)
		}
	}
	return out
}

// planFor is the deletion plan of a user's explicit delete of entries, judged
// with the test instance's identity.
func planFor(entries ...k8sinventory.Entry) lifecycle.DeletionPlan {
	return lifecycle.NewDeletionPlan(entries, lifecycle.Policy{Prune: true}, testInstanceUUID)
}

var cmEntry = k8sinventory.Entry{Version: "v1", Kind: "ConfigMap", Namespace: "default", Name: "cm"}

func ownedCM(name string) *unstructured.Unstructured {
	return withUID(owned("v1", "ConfigMap", name, "default", opmlabels.ManagedByCLI, testInstanceUUID), "uid-"+name)
}

// TestRunDeletion covers every answer of the one function that deletes an
// instance's object, for a plan of one step: each skip of the library's
// delete verdict, the two ways an object is already gone, a failed read, a
// read that returns another object, and a delete refused on the UID
// precondition. A skip and a failed read send no delete.
func TestRunDeletion(t *testing.T) {
	mine := func() *unstructured.Unstructured { return ownedCM("cm") }
	answers := func(verb string, obj runtime.Object, err error) func(*dynamicfake.FakeDynamicClient) {
		return func(dyn *dynamicfake.FakeDynamicClient) {
			dyn.PrependReactor(verb, "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, obj, err
			})
		}
	}
	gr := schema.GroupResource{Resource: "configmaps"}

	tests := []struct {
		name        string
		entry       k8sinventory.Entry
		live        *unstructured.Unstructured
		reactor     func(*dynamicfake.FakeDynamicClient)
		wantResult  lifecycle.Result
		wantSkip    ownership.SkipReason
		wantFailed  lifecycle.ActionKind
		wantIs      error
		wantDeletes int
		noRequests  bool
	}{
		{name: "owned object is deleted", entry: cmEntry, live: mine(), wantResult: lifecycle.ResultDeleted, wantDeletes: 1},
		{name: "absent at the read", entry: cmEntry, wantResult: lifecycle.ResultSkipped, wantSkip: ownership.SkipAlreadyAbsent},
		{
			name: "not managed by OPM", entry: cmEntry,
			live:       withUID(owned("v1", "ConfigMap", "cm", "default", "", ""), "uid-read"),
			wantResult: lifecycle.ResultSkipped, wantSkip: ownership.SkipNotOPMManaged,
		},
		{
			name: "another instance's", entry: cmEntry,
			live:       withUID(owned("v1", "ConfigMap", "cm", "default", opmlabels.ManagedByController, "uuid-other"), "uid-read"),
			wantResult: lifecycle.ResultSkipped, wantSkip: ownership.SkipOwnerMismatch,
		},
		{
			name: "adopted by another instance", entry: cmEntry,
			live:       adoptedBy(mine(), "uuid-other"),
			wantResult: lifecycle.ResultSkipped, wantSkip: ownership.SkipAdoptedElsewhere,
		},
		{
			name:       "protected kind is skipped without a read",
			entry:      k8sinventory.Entry{Version: "v1", Kind: "Namespace", Name: "apps"},
			live:       owned("v1", "Namespace", "apps", "", opmlabels.ManagedByCLI, testInstanceUUID),
			wantResult: lifecycle.ResultSkipped, wantSkip: ownership.SkipSafetyExcluded, noRequests: true,
		},
		{
			name: "gone at the delete", entry: cmEntry, live: mine(),
			reactor:    answers("delete", nil, apierrors.NewNotFound(gr, "cm")),
			wantResult: lifecycle.ResultSkipped, wantSkip: ownership.SkipAlreadyAbsent, wantDeletes: 1,
		},
		{
			name: "replaced since the read", entry: cmEntry, live: mine(),
			reactor:    answers("delete", nil, apierrors.NewConflict(gr, "cm", errors.New("precondition failed"))),
			wantResult: lifecycle.ResultFailed, wantFailed: lifecycle.ActionDelete, wantIs: ErrReplaced, wantDeletes: 1,
		},
		{
			name: "delete denied", entry: cmEntry, live: mine(),
			reactor:    answers("delete", nil, apierrors.NewForbidden(gr, "cm", errors.New("denied"))),
			wantResult: lifecycle.ResultFailed, wantFailed: lifecycle.ActionDelete, wantDeletes: 1,
		},
		{
			name: "read denied sends no delete", entry: cmEntry, live: mine(),
			reactor:    answers("get", nil, apierrors.NewForbidden(gr, "cm", errors.New("denied"))),
			wantResult: lifecycle.ResultFailed, wantFailed: lifecycle.ActionRead,
		},
		{
			name: "read returns another object", entry: cmEntry, live: mine(),
			reactor:    answers("get", ownedCM("other"), nil),
			wantResult: lifecycle.ResultFailed, wantFailed: lifecycle.ActionRead,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var objs []runtime.Object
			if tc.live != nil {
				objs = append(objs, tc.live.DeepCopy())
			}
			dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), objs...)
			if tc.reactor != nil {
				tc.reactor(dyn)
			}
			client := &Client{Resources: kubetest.Resources(), Dynamic: dyn}

			var seen []StepResult
			run, err := RunDeletion(context.Background(), client, planFor(tc.entry),
				DeletionOptions{OnStep: func(s StepResult) { seen = append(seen, s) }})
			require.NoError(t, err)
			require.Len(t, run.Steps, 1)
			assert.Equal(t, run.Steps, seen, "OnStep sees every step")
			step := run.Steps[0]

			assert.Equal(t, tc.entry, step.Entry)
			assert.Equal(t, tc.wantResult, step.Outcome.Result)
			assert.Equal(t, tc.wantSkip, step.Outcome.Skip)
			assert.Equal(t, tc.wantFailed, step.Failed, "the caller can tell a failed read from a failed delete")
			if tc.wantSkip != "" {
				assert.NotEmpty(t, step.Outcome.Message, "a skip carries the library's message")
			}
			if tc.wantResult == lifecycle.ResultFailed {
				require.Error(t, step.Err)
				if tc.wantIs != nil {
					assert.ErrorIs(t, step.Err, tc.wantIs)
					assert.True(t, apierrors.IsConflict(step.Err), "the API error stays in the chain")
				} else {
					assert.NotErrorIs(t, step.Err, ErrReplaced)
				}
			} else {
				assert.NoError(t, step.Err)
			}
			assert.Len(t, deleteActions(dyn), tc.wantDeletes)
			if tc.noRequests {
				assert.Empty(t, dyn.Actions(), "no read for a kind OPM never deletes")
			}

			// The hold verdict follows the run: only a failed step holds.
			verdict := lifecycle.MayReleaseHold(run.Plan, run.State, lifecycle.HoldInput{})
			assert.Equal(t, tc.wantResult != lifecycle.ResultFailed, verdict.Release)
		})
	}
}

// A read that returns another object than the step's is refused by the plan.
// The message is new with the plan: before it the object was judged as read.
func TestRunDeletion_ReadOfAnotherObjectIsNamed(t *testing.T) {
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	dyn.PrependReactor("get", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, ownedCM("other"), nil
	})
	client := &Client{Resources: kubetest.Resources(), Dynamic: dyn}

	run, err := RunDeletion(context.Background(), client, planFor(cmEntry), DeletionOptions{})
	require.NoError(t, err)
	require.Len(t, run.Steps, 1)
	assert.EqualError(t, run.Steps[0].Err, "reading ConfigMap/default/cm returned ConfigMap/default/other; not deleted")
	assert.Empty(t, deleteActions(dyn))
}

// TestRunDeletion_SendsWhatThePlanNames pins the delete on the wire: the UID
// of the object that was read and judged, foreground propagation, and no
// resourceVersion.
func TestRunDeletion_SendsWhatThePlanNames(t *testing.T) {
	live := withUID(owned("v1", "ConfigMap", "cm", "default", opmlabels.ManagedByCLI, testInstanceUUID), "uid-on-the-wire")
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), live)
	client := &Client{Resources: kubetest.Resources(), Dynamic: dyn}

	run, err := RunDeletion(context.Background(), client, planFor(cmEntry), DeletionOptions{})
	require.NoError(t, err)
	require.Len(t, run.Steps, 1)
	require.Equal(t, lifecycle.ResultDeleted, run.Steps[0].Outcome.Result)

	deletes := deleteActions(dyn)
	require.Len(t, deletes, 1)
	opts := deletes[0].GetDeleteOptions()
	require.NotNil(t, opts.Preconditions)
	require.NotNil(t, opts.Preconditions.UID)
	assert.Equal(t, types.UID("uid-on-the-wire"), *opts.Preconditions.UID)
	assert.Nil(t, opts.Preconditions.ResourceVersion)
	require.NotNil(t, opts.PropagationPolicy)
	assert.Equal(t, "Foreground", string(*opts.PropagationPolicy))
}

// A dry run reads and judges, sends no delete, and records the step as a
// real run would after a successful delete.
func TestRunDeletion_DryRunSendsNoDelete(t *testing.T) {
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), ownedCM("cm"), ownedCM("b"))
	client := &Client{Resources: kubetest.Resources(), Dynamic: dyn}
	b := k8sinventory.Entry{Version: "v1", Kind: "ConfigMap", Namespace: "default", Name: "b"}

	run, err := RunDeletion(context.Background(), client, planFor(cmEntry, b), DeletionOptions{DryRun: true})
	require.NoError(t, err)
	require.Len(t, run.Steps, 2)
	for _, s := range run.Steps {
		assert.Equal(t, lifecycle.ResultDeleted, s.Outcome.Result)
	}
	assert.Empty(t, deleteActions(dyn))
	assert.Len(t, dyn.Actions(), 2, "one read per step")
}

// With StopOnDiscoveryFailure, the first failed discovery request ends the
// requests: that step and every later one fail with the discovery error. A
// kind that is not served does not stop the run, and without the option a
// discovery failure fails its own step only.
func TestRunDeletion_StopOnDiscoveryFailure(t *testing.T) {
	promGVK := schema.GroupVersionKind{Group: "monitoring.coreos.com", Version: "v1", Kind: "Prometheus"}
	widgetGVK := schema.GroupVersionKind{Group: "example.io", Version: "v1", Kind: "Widget"}
	prom := k8sinventory.Entry{Group: promGVK.Group, Version: "v1", Kind: "Prometheus", Namespace: "default", Name: "main"}
	widget := k8sinventory.Entry{Group: widgetGVK.Group, Version: "v1", Kind: "Widget", Namespace: "default", Name: "w"}
	forbidden := apierrors.NewForbidden(schema.GroupResource{}, "", errors.New("denied"))
	discovery := &DiscoveryError{GroupVersion: promGVK.GroupVersion(), Err: forbidden}

	newClient := func() (*Client, *dynamicfake.FakeDynamicClient) {
		dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), ownedCM("cm"))
		return &Client{
			Dynamic: dyn,
			Resources: kubetest.ResourcesWith(map[schema.GroupVersionKind]kubetest.Outcome{
				widgetGVK: {Err: &KindNotServedError{GVK: widgetGVK}},
				promGVK:   {Err: discovery},
			}),
		}, dyn
	}
	// Same weight for the two custom kinds, so plan order is input order;
	// the ConfigMap comes last.
	plan := planFor(widget, prom, cmEntry)

	t.Run("stop", func(t *testing.T) {
		client, dyn := newClient()
		run, err := RunDeletion(context.Background(), client, plan, DeletionOptions{StopOnDiscoveryFailure: true})
		require.NoError(t, err)
		require.Len(t, run.Steps, 3)

		assert.Equal(t, widget, run.Steps[0].Entry)
		assert.True(t, IsKindNotServed(run.Steps[0].Err))
		for _, s := range run.Steps[1:] {
			assert.Equal(t, lifecycle.ResultFailed, s.Outcome.Result)
			assert.Equal(t, lifecycle.ActionRead, s.Failed)
			assert.True(t, IsDiscoveryFailure(s.Err))
			assert.True(t, apierrors.IsForbidden(s.Err))
		}
		assert.Equal(t, cmEntry, run.Steps[2].Entry)
		assert.Empty(t, dyn.Actions(), "no request after the failed discovery")
	})

	t.Run("go on", func(t *testing.T) {
		client, dyn := newClient()
		run, err := RunDeletion(context.Background(), client, plan, DeletionOptions{})
		require.NoError(t, err)
		require.Len(t, run.Steps, 3)
		assert.True(t, IsDiscoveryFailure(run.Steps[1].Err))
		assert.Equal(t, lifecycle.ResultDeleted, run.Steps[2].Outcome.Result)
		assert.Len(t, deleteActions(dyn), 1)
	})
}

// An object the caller already failed to read is not read again: its step
// fails with the caller's error, and the other steps run.
func TestRunDeletion_UnreadableIsNotReadAgain(t *testing.T) {
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), ownedCM("cm"), ownedCM("b"))
	client := &Client{Resources: kubetest.Resources(), Dynamic: dyn}
	b := k8sinventory.Entry{Version: "v1", Kind: "ConfigMap", Namespace: "default", Name: "b"}
	readErr := apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "cm", errors.New("denied"))

	run, err := RunDeletion(context.Background(), client, planFor(cmEntry, b), DeletionOptions{
		Unreadable: map[ownership.Object]error{entryObject(cmEntry): readErr},
	})
	require.NoError(t, err)
	require.Len(t, run.Steps, 2)
	assert.Equal(t, lifecycle.ResultFailed, run.Steps[0].Outcome.Result)
	assert.Equal(t, lifecycle.ActionRead, run.Steps[0].Failed)
	assert.Same(t, error(readErr), run.Steps[0].Err)
	assert.Equal(t, lifecycle.ResultDeleted, run.Steps[1].Outcome.Result)

	for _, a := range dyn.Actions() {
		if g, ok := a.(k8stesting.GetAction); ok {
			assert.NotEqual(t, "cm", g.GetName(), "the unreadable object is not read again")
		}
	}
	assert.False(t, lifecycle.MayReleaseHold(run.Plan, run.State, lifecycle.HoldInput{}).Release)
}

// The plan deletes in the order the CLI's own descending sort gives, equal
// weights included, so moving the delete loops onto the plan moves no
// delete.
func TestRunDeletion_OrderIsTheDescendingSort(t *testing.T) {
	objs := []*unstructured.Unstructured{
		ownedCM("first"),
		owned("apps/v1", "Deployment", "web", "default", opmlabels.ManagedByCLI, testInstanceUUID),
		owned("v1", "Service", "web", "default", opmlabels.ManagedByCLI, testInstanceUUID),
		ownedCM("second"),
		owned("v1", "Secret", "creds", "default", opmlabels.ManagedByCLI, testInstanceUUID),
		ownedCM("third"),
	}
	seed := make([]runtime.Object, 0, len(objs))
	entries := make([]k8sinventory.Entry, 0, len(objs))
	for _, o := range objs {
		seed = append(seed, o.DeepCopy())
		gvk := o.GroupVersionKind()
		entries = append(entries, k8sinventory.Entry{Group: gvk.Group, Version: gvk.Version, Kind: gvk.Kind, Namespace: o.GetNamespace(), Name: o.GetName()})
	}
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), seed...)
	client := &Client{Resources: kubetest.Resources(), Dynamic: dyn}

	_, err := RunDeletion(context.Background(), client, planFor(entries...), DeletionOptions{})
	require.NoError(t, err)

	SortObjects(objs, object.Descending)
	want := make([]string, 0, len(objs))
	for _, o := range objs {
		want = append(want, kubetest.GVR(o).Resource+"/"+o.GetName())
	}
	deletes := deleteActions(dyn)
	got := make([]string, 0, len(deletes))
	for _, d := range deletes {
		got = append(got, d.GetResource().Resource+"/"+d.GetName())
	}
	assert.Equal(t, want, got)
}

// TestIsProtectedKind_IsTheLibraryRule: the CLI's protected-kind test and the
// library's safety exclusion are one rule.
func TestIsProtectedKind_IsTheLibraryRule(t *testing.T) {
	for _, tc := range []struct{ group, kind string }{
		{"", "Namespace"},
		{"apiextensions.k8s.io", "CustomResourceDefinition"},
		{"example.io", "Namespace"},
		{"", "CustomResourceDefinition"},
		{"apps", "Deployment"},
		{"", "PersistentVolumeClaim"},
	} {
		assert.Equal(t, ownership.SafetyExcluded(tc.group, tc.kind), IsProtectedKind(tc.group, tc.kind), "%s/%s", tc.group, tc.kind)
	}
}
