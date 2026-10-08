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
	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"
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

// TestJudgedDelete covers every answer of the one function that deletes an
// instance's object: each skip of the library's delete verdict, the two ways
// an object is already gone, a failed read, and a delete refused on the UID
// precondition. A skip and a failed read send no delete.
func TestJudgedDelete(t *testing.T) {
	cm := ownership.Object{Kind: "ConfigMap", Namespace: "default", Name: "cm"}
	mine := func() *unstructured.Unstructured {
		return withUID(owned("v1", "ConfigMap", "cm", "default", opmlabels.ManagedByCLI, testInstanceUUID), "uid-read")
	}
	deleteAnswers := func(err error) func(*dynamicfake.FakeDynamicClient) {
		return func(dyn *dynamicfake.FakeDynamicClient) {
			dyn.PrependReactor("delete", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, err
			})
		}
	}
	gr := schema.GroupResource{Resource: "configmaps"}

	tests := []struct {
		name        string
		obj         ownership.Object
		version     string
		live        *unstructured.Unstructured
		reactor     func(*dynamicfake.FakeDynamicClient)
		wantSkip    ownership.SkipReason
		wantDeleted bool
		wantErr     bool
		wantIs      error
		wantDeletes int
	}{
		{name: "owned object is deleted", obj: cm, version: "v1", live: mine(), wantDeleted: true, wantDeletes: 1},
		{name: "absent at the read", obj: cm, version: "v1", wantSkip: ownership.SkipAlreadyAbsent},
		{
			name: "not managed by OPM", obj: cm, version: "v1",
			live:     withUID(owned("v1", "ConfigMap", "cm", "default", "", ""), "uid-read"),
			wantSkip: ownership.SkipNotOPMManaged,
		},
		{
			name: "another instance's", obj: cm, version: "v1",
			live:     withUID(owned("v1", "ConfigMap", "cm", "default", opmlabels.ManagedByController, "uuid-other"), "uid-read"),
			wantSkip: ownership.SkipOwnerMismatch,
		},
		{
			name: "adopted by another instance", obj: cm, version: "v1",
			live:     adoptedBy(mine(), "uuid-other"),
			wantSkip: ownership.SkipAdoptedElsewhere,
		},
		{
			name:     "protected kind is skipped without a read",
			obj:      ownership.Object{Kind: "Namespace", Name: "apps"},
			version:  "v1",
			live:     owned("v1", "Namespace", "apps", "", opmlabels.ManagedByCLI, testInstanceUUID),
			wantSkip: ownership.SkipSafetyExcluded,
		},
		{
			name: "gone at the delete", obj: cm, version: "v1", live: mine(),
			reactor:  deleteAnswers(apierrors.NewNotFound(gr, "cm")),
			wantSkip: ownership.SkipAlreadyAbsent, wantDeletes: 1,
		},
		{
			name: "replaced since the read", obj: cm, version: "v1", live: mine(),
			reactor: deleteAnswers(apierrors.NewConflict(gr, "cm", errors.New("precondition failed"))),
			wantErr: true, wantIs: ErrReplaced, wantDeletes: 1,
		},
		{
			name: "delete denied", obj: cm, version: "v1", live: mine(),
			reactor: deleteAnswers(apierrors.NewForbidden(gr, "cm", errors.New("denied"))),
			wantErr: true, wantDeletes: 1,
		},
		{
			name: "read denied sends no delete", obj: cm, version: "v1", live: mine(),
			reactor: func(dyn *dynamicfake.FakeDynamicClient) {
				dyn.PrependReactor("get", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
					return true, nil, apierrors.NewForbidden(gr, "cm", errors.New("denied"))
				})
			},
			wantErr: true,
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

			got, err := JudgedDelete(context.Background(), client, tc.obj, tc.version, testInstanceUUID, false)

			if tc.wantErr {
				require.Error(t, err)
				if tc.wantIs != nil {
					assert.ErrorIs(t, err, tc.wantIs)
				} else {
					assert.NotErrorIs(t, err, ErrReplaced)
				}
				assert.False(t, got.Deleted, "a failed delete is never reported as deleted")
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.wantSkip, got.Skip)
			assert.Equal(t, tc.wantDeleted, got.Deleted)
			if tc.wantSkip != "" {
				assert.NotEmpty(t, got.Message, "a skip carries the library's message")
			}
			assert.Len(t, deleteActions(dyn), tc.wantDeletes)
			if tc.name == "protected kind is skipped without a read" {
				assert.Empty(t, dyn.Actions(), "no read for a kind OPM never deletes")
			}
		})
	}
}

// TestJudgedDelete_SendsUIDPrecondition pins the precondition on the wire:
// the DELETE names the UID of the object that was read and judged, with
// foreground propagation, and no resourceVersion.
func TestJudgedDelete_SendsUIDPrecondition(t *testing.T) {
	live := withUID(owned("v1", "ConfigMap", "cm", "default", opmlabels.ManagedByCLI, testInstanceUUID), "uid-on-the-wire")
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), live)
	client := &Client{Resources: kubetest.Resources(), Dynamic: dyn}

	got, err := JudgedDelete(context.Background(), client,
		ownership.Object{Kind: "ConfigMap", Namespace: "default", Name: "cm"}, "v1", testInstanceUUID, false)
	require.NoError(t, err)
	require.True(t, got.Deleted)

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

// TestJudgedDelete_DryRunStopsAfterTheVerdict: a dry run reads and judges,
// and sends no delete.
func TestJudgedDelete_DryRunStopsAfterTheVerdict(t *testing.T) {
	live := withUID(owned("v1", "ConfigMap", "cm", "default", opmlabels.ManagedByCLI, testInstanceUUID), "uid-read")
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), live)
	client := &Client{Resources: kubetest.Resources(), Dynamic: dyn}

	got, err := JudgedDelete(context.Background(), client,
		ownership.Object{Kind: "ConfigMap", Namespace: "default", Name: "cm"}, "v1", testInstanceUUID, true)
	require.NoError(t, err)
	assert.Equal(t, DeleteOutcome{}, got)
	assert.Empty(t, deleteActions(dyn))
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
