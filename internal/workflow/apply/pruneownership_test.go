package apply

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8stesting "k8s.io/client-go/testing"

	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"

	opmexit "github.com/open-platform-model/cli/internal/exit"
)

// The identities of these tests. The request renders the instance with
// renderIdentity (applyCluster.request); recordedIdentity is what an earlier
// apply stored in the record before the module moved to a new path.
const (
	renderIdentity = "uuid-1"
	oldIdentity    = "uuid-before-the-move"
)

// recordWithIdentity is the CLI-owned record of "demo" whose inventory holds
// the named ConfigMaps and whose status stores the given instance identity.
func recordWithIdentity(identity string, configMaps ...string) *unstructured.Unstructured {
	rec := cliOwnedInstance("demo", "default", configMaps...)
	if err := unstructured.SetNestedField(rec.Object, identity, "status", "instanceUUID"); err != nil {
		panic(err)
	}
	return rec
}

// liveConfigMap is a ConfigMap in the cluster with the given managed-by
// value, instance identity label and adopt annotation; an empty value leaves
// that one off.
func liveConfigMap(name, managedBy, identity, adoptedBy string) *unstructured.Unstructured {
	cm := renderedConfigMap(name)
	labels := map[string]string{}
	if managedBy != "" {
		labels[opmlabels.ManagedBy] = managedBy
	}
	if identity != "" {
		labels[opmlabels.ModuleInstanceUUID] = identity
	}
	cm.SetLabels(labels)
	if adoptedBy != "" {
		cm.SetAnnotations(map[string]string{opmlabels.AnnotationAdopt: adoptedBy})
	}
	cm.SetUID(types.UID("uid-" + name))
	return cm
}

// writtenIdentity is the instance identity of the last status write.
func (c *applyCluster) writtenIdentity(t *testing.T) string {
	t.Helper()
	identity := ""
	for _, a := range c.dyn.Actions() {
		patch, ok := a.(k8stesting.PatchAction)
		if !ok || a.GetResource().Resource != "moduleinstances" || a.GetSubresource() != "status" {
			continue
		}
		var body struct {
			Status struct {
				InstanceUUID string `json:"instanceUUID"`
			} `json:"status"`
		}
		require.NoError(t, json.Unmarshal(patch.GetPatch(), &body))
		identity = body.Status.InstanceUUID
	}
	return identity
}

// A stale entry is deleted only when the live object under its name is still
// the instance's. An object that OPM does not manage, that belongs to another
// instance or that another instance is adopting is left in the cluster,
// reported with the reason, and dropped from the record; the apply succeeds.
// Before prune read the live object it deleted every one of these.
func TestExecute_PruneLeavesBehindWhatTheInstanceDoesNotOwn(t *testing.T) {
	tests := map[string]struct {
		live       *unstructured.Unstructured
		wantReason string
	}{
		"a user's object took the recorded name": {
			live:       liveConfigMap("old", "", "", ""),
			wantReason: "ConfigMap/default/old is not managed by OPM; left in place",
		},
		"another instance owns it now": {
			live:       liveConfigMap("old", opmlabels.ManagedByController, "uuid-other", ""),
			wantReason: "ConfigMap/default/old belongs to module instance uuid-other, not this one; left in place",
		},
		"another instance is adopting it": {
			live:       liveConfigMap("old", opmlabels.ManagedByCLI, renderIdentity, "uuid-other"),
			wantReason: "ConfigMap/default/old is being adopted by module instance uuid-other, not this one; left in place",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			withReleasedCLIVersion(t)
			logBuf := captureLog(t)
			cluster := newApplyCluster(recordWithIdentity(renderIdentity, "keep", "old"), tc.live)

			var err error
			stdout := captureStdout(t, func() {
				err = Execute(context.Background(), cluster.request(Options{}, "keep"))
			})

			require.NoError(t, err, "a left-behind object is not a failure")
			assert.Contains(t, stdout, "applied")
			assert.Empty(t, cluster.deletes(), "the object is not the instance's to delete")
			assert.Contains(t, logBuf.String(), "left behind")
			assert.Contains(t, logBuf.String(), tc.wantReason)
			entries, written := cluster.writtenInventory(t)
			require.True(t, written)
			assert.Equal(t, []string{"keep"}, entryNames(entries), "the object leaves the record")
		})
	}
}

// The stale object the instance still owns is deleted, with a precondition
// on the UID of the object that was read.
func TestExecute_PruneDeletesItsOwnWithTheUIDItRead(t *testing.T) {
	withReleasedCLIVersion(t)
	captureLog(t)
	cluster := newApplyCluster(recordWithIdentity(renderIdentity, "keep", "old"),
		liveConfigMap("old", opmlabels.ManagedByCLI, renderIdentity, ""))

	require.NoError(t, Execute(context.Background(), cluster.request(Options{}, "keep")))

	require.Equal(t, []string{"configmaps/old"}, cluster.deletes())
	for _, a := range cluster.dyn.Actions() {
		if d, ok := a.(k8stesting.DeleteAction); ok {
			pre := d.GetDeleteOptions().Preconditions
			require.NotNil(t, pre)
			require.NotNil(t, pre.UID)
			assert.Equal(t, "uid-old", string(*pre.UID))
		}
	}
}

// After the module moved to a new path the render's identity differs from
// the one in the record. The prune judges with the recorded identity, the
// one that applied the stale objects: it deletes the stale object that
// carries it, leaves one that carries a third identity, and only the write
// that follows moves the record to the render's identity. Judged with the
// render's identity, the instance's own stale object would be left behind.
func TestExecute_PruneAfterAnIdentityChangeJudgesWithTheRecordedIdentity(t *testing.T) {
	withReleasedCLIVersion(t)
	logBuf := captureLog(t)
	cluster := newApplyCluster(
		recordWithIdentity(oldIdentity, "keep", "mine", "theirs"),
		liveConfigMap("mine", opmlabels.ManagedByCLI, oldIdentity, ""),
		liveConfigMap("theirs", opmlabels.ManagedByCLI, "uuid-third", ""),
	)

	require.NoError(t, Execute(context.Background(), cluster.request(Options{}, "keep")))

	assert.Equal(t, []string{"configmaps/mine"}, cluster.deletes(),
		"the stale object of the recorded identity is pruned; the third identity's is not")
	assert.Contains(t, logBuf.String(), "ConfigMap/default/theirs belongs to module instance uuid-third, not this one")
	entries, written := cluster.writtenInventory(t)
	require.True(t, written)
	assert.Equal(t, []string{"keep"}, entryNames(entries))
	assert.Equal(t, renderIdentity, cluster.writtenIdentity(t), "the record takes the render's identity in the write after the prune")
}

// The documented limit of judging with the recorded identity: a prune that
// fails on the apply right after an identity change keeps its entry, and the
// record written by that apply already holds the new identity. The retry
// judges the entry with that new identity, so the object, which still carries
// the old one, is left behind with a warning and leaves the record. Nothing
// is deleted that should stay; the user removes the object by hand.
func TestExecute_FailedPruneAfterAnIdentityChangeIsLeftBehindOnTheRetry(t *testing.T) {
	withReleasedCLIVersion(t)

	// The apply right after the move: the delete of "stuck" is denied.
	captureLog(t)
	first := newApplyCluster(recordWithIdentity(oldIdentity, "keep", "stuck"),
		liveConfigMap("stuck", opmlabels.ManagedByCLI, oldIdentity, ""))
	first.dyn.PrependReactor("delete", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "stuck", errors.New("no delete access"))
	})
	requireExitCode(t, Execute(context.Background(), first.request(Options{}, "keep")), opmexit.ExitGeneralError)
	entries, written := first.writtenInventory(t)
	require.True(t, written)
	assert.ElementsMatch(t, []string{"keep", "stuck"}, entryNames(entries), "the failed entry stays in the record")
	require.Equal(t, renderIdentity, first.writtenIdentity(t), "and the record already holds the new identity")

	// The retry, on the state the first apply left.
	logBuf := captureLog(t)
	retry := newApplyCluster(recordWithIdentity(renderIdentity, "keep", "stuck"),
		liveConfigMap("stuck", opmlabels.ManagedByCLI, oldIdentity, ""))
	require.NoError(t, Execute(context.Background(), retry.request(Options{}, "keep")))

	assert.Empty(t, retry.deletes(), "the retry does not delete the object")
	assert.Contains(t, logBuf.String(), "left behind")
	assert.Contains(t, logBuf.String(), "ConfigMap/default/stuck belongs to module instance "+oldIdentity+", not this one")
	entries, written = retry.writtenInventory(t)
	require.True(t, written)
	assert.Equal(t, []string{"keep"}, entryNames(entries), "and drops it from the record")
}

// A stale object the prune cannot judge or cannot delete safely is a failed
// prune: the entry stays in the record and the command exits 1 after the
// write. A denied read sends no delete; a delete the API server refuses on
// the UID precondition means the object was replaced since the read.
func TestExecute_PruneThatCannotJudgeOrWasOvertakenKeepsTheEntry(t *testing.T) {
	gr := schema.GroupResource{Resource: "configmaps"}
	tests := map[string]struct {
		verb        string
		answer      error
		wantDeletes int
		wantInLog   string
	}{
		"the live read is denied": {
			verb: "get", answer: apierrors.NewForbidden(gr, "old", errors.New("no read access")),
			wantInLog: "no read access",
		},
		"the object was replaced since the read": {
			verb: "delete", answer: apierrors.NewConflict(gr, "old", errors.New("the UID in the precondition does not match")),
			wantDeletes: 1, wantInLog: "was replaced after it was read",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			withReleasedCLIVersion(t)
			logBuf := captureLog(t)
			cluster := newApplyCluster(recordWithIdentity(renderIdentity, "keep", "old"),
				liveConfigMap("old", opmlabels.ManagedByCLI, renderIdentity, ""))
			cluster.dyn.PrependReactor(tc.verb, "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
				if named, ok := action.(interface{ GetName() string }); ok && named.GetName() == "old" {
					return true, nil, tc.answer
				}
				return false, nil, nil
			})

			var err error
			stdout := captureStdout(t, func() {
				err = Execute(context.Background(), cluster.request(Options{}, "keep"))
			})

			requireExitCode(t, err, opmexit.ExitGeneralError)
			assert.NotContains(t, stdout, "applied", "no success line")
			assert.Len(t, cluster.deletes(), tc.wantDeletes)
			assert.Contains(t, logBuf.String(), "prune failed")
			assert.Contains(t, logBuf.String(), tc.wantInLog)
			entries, written := cluster.writtenInventory(t)
			require.True(t, written)
			assert.ElementsMatch(t, []string{"keep", "old"}, entryNames(entries), "the entry stays in the record")
		})
	}
}
