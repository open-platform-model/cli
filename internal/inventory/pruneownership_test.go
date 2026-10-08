package inventory

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

	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"

	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"
)

const recordedUUID = "uuid-recorded"

// staleCM is a live ConfigMap "old" in "default" with the given managed-by
// value, UUID label and adopt annotation; an empty value leaves that one off.
func staleCM(managedBy, uuid, adoptedBy string) *unstructured.Unstructured {
	obj := liveObject("v1", "ConfigMap", "default", "old")
	labels := map[string]string{}
	if managedBy != "" {
		labels[opmlabels.ManagedBy] = managedBy
	}
	if uuid != "" {
		labels[opmlabels.ModuleInstanceUUID] = uuid
	}
	obj.SetLabels(labels)
	if adoptedBy != "" {
		obj.SetAnnotations(map[string]string{opmlabels.AnnotationAdopt: adoptedBy})
	}
	obj.SetUID("uid-old")
	return obj
}

// TestPruneStaleResources_AsksTheDeleteVerdict covers what the prune does
// with each answer for one stale entry: it deletes only the object the
// instance still owns, with the UID it read; it returns as left behind an
// object the verdict skips, without a delete; and it reports as failed an
// entry it could not read or whose object was replaced since the read.
func TestPruneStaleResources_AsksTheDeleteVerdict(t *testing.T) {
	entry := k8sinventory.Entry{Version: "v1", Kind: "ConfigMap", Namespace: "default", Name: "old"}
	gr := schema.GroupResource{Resource: "configmaps"}
	answer := func(verb string, err error) func(*dynamicfake.FakeDynamicClient) {
		return func(dyn *dynamicfake.FakeDynamicClient) {
			dyn.PrependReactor(verb, "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, err
			})
		}
	}

	tests := map[string]struct {
		live        *unstructured.Unstructured
		reactor     func(*dynamicfake.FakeDynamicClient)
		wantDeletes int
		wantLeft    string // non-empty: left behind, and the reason holds this
		wantFailed  bool
		wantIs      error
	}{
		"owned object is deleted": {
			live: staleCM(opmlabels.ManagedByCLI, recordedUUID, ""), wantDeletes: 1,
		},
		"object without a UUID label is deleted on managed-by": {
			live: staleCM(opmlabels.ManagedByLegacy, "", ""), wantDeletes: 1,
		},
		"name taken by an object OPM does not manage": {
			live: staleCM("", "", ""), wantLeft: "is not managed by OPM",
		},
		"another instance's object": {
			live: staleCM(opmlabels.ManagedByController, "uuid-other", ""), wantLeft: "belongs to module instance uuid-other",
		},
		"object adopted by another instance": {
			live: staleCM(opmlabels.ManagedByCLI, recordedUUID, "uuid-other"), wantLeft: "is being adopted by module instance uuid-other",
		},
		"already gone": {},
		"read denied": {
			live:       staleCM(opmlabels.ManagedByCLI, recordedUUID, ""),
			reactor:    answer("get", apierrors.NewForbidden(gr, "old", errors.New("denied"))),
			wantFailed: true,
		},
		"replaced since the read": {
			live:        staleCM(opmlabels.ManagedByCLI, recordedUUID, ""),
			reactor:     answer("delete", apierrors.NewConflict(gr, "old", errors.New("precondition failed"))),
			wantDeletes: 1, wantFailed: true, wantIs: kubernetes.ErrReplaced,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var objs []runtime.Object
			if tc.live != nil {
				objs = append(objs, tc.live)
			}
			dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), objs...)
			if tc.reactor != nil {
				tc.reactor(dyn)
			}
			client := &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: dyn}

			leftBehind, err := PruneStaleResources(context.Background(), client, []k8sinventory.Entry{entry}, recordedUUID)

			var deletes []k8stesting.DeleteAction
			for _, a := range dyn.Actions() {
				if d, ok := a.(k8stesting.DeleteAction); ok {
					deletes = append(deletes, d)
				}
			}
			require.Len(t, deletes, tc.wantDeletes)
			for _, d := range deletes {
				pre := d.GetDeleteOptions().Preconditions
				require.NotNil(t, pre, "every delete carries the UID that was read")
				require.NotNil(t, pre.UID)
				assert.Equal(t, "uid-old", string(*pre.UID))
			}

			if tc.wantLeft != "" {
				require.Len(t, leftBehind, 1)
				assert.Equal(t, entry, leftBehind[0].Entry)
				assert.Contains(t, leftBehind[0].Reason, tc.wantLeft)
			} else {
				assert.Empty(t, leftBehind)
			}

			if !tc.wantFailed {
				require.NoError(t, err)
				return
			}
			var pruneErr *PruneError
			require.ErrorAs(t, err, &pruneErr)
			assert.Equal(t, []k8sinventory.Entry{entry}, pruneErr.Failed)
			if tc.wantIs != nil {
				assert.ErrorIs(t, err, tc.wantIs)
			}
		})
	}
}
