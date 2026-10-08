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
	k8stesting "k8s.io/client-go/testing"

	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/output"
)

// claimEntry is the raw inventory entry of a PersistentVolumeClaim in
// "default".
func claimEntry(name string) map[string]any {
	return map[string]any{"group": "", "kind": "PersistentVolumeClaim", "namespace": "default", "name": name, "v": "v1", "component": "app"}
}

func configMapEntry(name string) map[string]any {
	return map[string]any{"group": "", "kind": "ConfigMap", "namespace": "default", "name": name, "v": "v1", "component": "app"}
}

func liveClaim(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "PersistentVolumeClaim",
		"metadata": map[string]any{"name": name, "namespace": "default"},
	}}
}

// deletes lists the delete calls the cluster received, as "<resource>/<name>".
func (c *applyCluster) deletes() []string {
	var out []string
	for _, a := range c.dyn.Actions() {
		if d, ok := a.(k8stesting.DeleteAction); ok {
			out = append(out, a.GetResource().Resource+"/"+d.GetName())
		}
	}
	return out
}

// claimCluster is the instance "demo" whose record holds the ConfigMaps
// "keep" and "gone" and the claim "data". An apply that renders only "keep"
// finds "gone" and "data" stale.
func claimCluster() *applyCluster {
	return newApplyCluster(
		cliOwnedInstanceWith("demo", "default", configMapEntry("keep"), configMapEntry("gone"), claimEntry("data")),
		renderedConfigMap("gone"), liveClaim("data"),
	)
}

// logLine returns the first log line that holds every part.
func logLine(log string, parts ...string) string {
	for _, line := range strings.Split(log, "\n") {
		ok := true
		for _, p := range parts {
			ok = ok && strings.Contains(line, p)
		}
		if ok {
			return line
		}
	}
	return ""
}

// By default prune keeps a stale claim: it is not deleted, it stays in the
// written inventory beside the current entries, the stale ConfigMap goes, the
// apply passes with its success line, and the claim is listed as kept on
// informational lines only.
func TestExecute_PruneKeepsStaleClaims(t *testing.T) {
	withReleasedCLIVersion(t)
	logBuf := captureLog(t)
	cluster := claimCluster()

	var err error
	stdout := captureStdout(t, func() {
		err = Execute(context.Background(), cluster.request(Options{}, "keep"))
	})
	require.NoError(t, err, "a kept claim does not fail the apply")
	assert.Contains(t, stdout, "applied", "the success line prints")

	assert.Equal(t, []string{"configmaps/gone"}, cluster.deletes(), "only the stale ConfigMap is deleted")
	entries, written := cluster.writtenInventory(t)
	require.True(t, written)
	assert.ElementsMatch(t, []string{"keep", "data"}, entryNames(entries), "the kept claim stays in the inventory")

	log := logBuf.String()
	keptLine := logLine(log, "PersistentVolumeClaim/default/data", output.StatusKept)
	require.NotEmpty(t, keptLine, "the claim is listed as kept:\n%s", log)
	assert.Contains(t, keptLine, "INFO")
	countLine := logLine(log, "keeping 1 stale PersistentVolumeClaim(s)")
	require.NotEmpty(t, countLine, log)
	assert.Contains(t, countLine, "INFO")
	assert.Contains(t, countLine, "--delete-data")
	assert.NotContains(t, log, "WARN", "keeping a claim prints no warning")
	assert.NotContains(t, log, "ERRO", "keeping a claim prints no error")

	// The kept entry is stale again on the next apply of the same render.
	stale := ComputeStaleInventorySet(entries, CurrentInventoryEntries(cluster.request(Options{}, "keep").Result.Resources))
	require.Len(t, stale, 1)
	assert.Equal(t, "data", stale[0].Name)
}

// With --delete-data a stale claim is pruned like any stale resource and
// leaves the inventory.
func TestExecute_DeleteDataPrunesStaleClaims(t *testing.T) {
	withReleasedCLIVersion(t)
	logBuf := captureLog(t)
	cluster := claimCluster()

	var err error
	captureStdout(t, func() {
		err = Execute(context.Background(), cluster.request(Options{DeleteData: true}, "keep"))
	})
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{"configmaps/gone", "persistentvolumeclaims/data"}, cluster.deletes())
	entries, written := cluster.writtenInventory(t)
	require.True(t, written)
	assert.Equal(t, []string{"keep"}, entryNames(entries))
	assert.Empty(t, logLine(logBuf.String(), output.StatusKept), "nothing is listed as kept")
}

// A claim an earlier apply kept is removed by a later apply with the flag:
// the record still lists it, so it is stale again.
func TestExecute_LaterApplyWithDeleteDataRemovesAKeptClaim(t *testing.T) {
	withReleasedCLIVersion(t)
	captureLog(t)
	// The record the first apply wrote: the current entry and the kept claim.
	cluster := newApplyCluster(
		cliOwnedInstanceWith("demo", "default", configMapEntry("keep"), claimEntry("data")),
		liveClaim("data"),
	)

	var err error
	captureStdout(t, func() {
		err = Execute(context.Background(), cluster.request(Options{DeleteData: true}, "keep"))
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"persistentvolumeclaims/data"}, cluster.deletes())
	entries, _ := cluster.writtenInventory(t)
	assert.Equal(t, []string{"keep"}, entryNames(entries))
}

// --force on an empty render prunes every tracked resource except claims.
func TestExecute_ForcedEmptyRenderKeepsClaims(t *testing.T) {
	withReleasedCLIVersion(t)
	captureLog(t)
	cluster := newApplyCluster(
		cliOwnedInstanceWith("demo", "default", configMapEntry("gone"), claimEntry("data")),
		renderedConfigMap("gone"), liveClaim("data"),
	)

	var err error
	captureStdout(t, func() {
		err = Execute(context.Background(), cluster.request(Options{Force: true}))
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"configmaps/gone"}, cluster.deletes(), "--force does not delete data")
	entries, written := cluster.writtenInventory(t)
	require.True(t, written)
	assert.Equal(t, []string{"data"}, entryNames(entries))
}

// --no-prune is unchanged: nothing is deleted, nothing is listed as kept, and
// the record holds the current entries only.
func TestExecute_NoPruneListsNoKeptClaims(t *testing.T) {
	withReleasedCLIVersion(t)
	logBuf := captureLog(t)
	cluster := claimCluster()

	var err error
	captureStdout(t, func() {
		err = Execute(context.Background(), cluster.request(Options{NoPrune: true}, "keep"))
	})
	require.NoError(t, err)
	assert.Empty(t, cluster.deletes())
	assert.Empty(t, logLine(logBuf.String(), output.StatusKept))
	entries, _ := cluster.writtenInventory(t)
	assert.Equal(t, []string{"keep"}, entryNames(entries))
}

// The dry-run preview lists a stale claim as kept, or as would prune with
// --delete-data, and deletes and writes nothing.
func TestExecute_DryRunPreviewsKeptClaims(t *testing.T) {
	withReleasedCLIVersion(t)
	for _, deleteData := range []bool{false, true} {
		logBuf := captureLog(t)
		cluster := claimCluster()
		require.NoError(t, Execute(context.Background(), cluster.request(Options{DryRun: true, DeleteData: deleteData}, "keep")))

		log := logBuf.String()
		assert.Empty(t, cluster.deletes(), "a dry run deletes nothing")
		_, written := cluster.writtenInventory(t)
		assert.False(t, written, "a dry run writes no record")
		assert.NotEmpty(t, logLine(log, "ConfigMap/default/gone", "would prune"), log)
		if deleteData {
			assert.NotEmpty(t, logLine(log, "PersistentVolumeClaim/default/data", "would prune"), log)
			assert.Empty(t, logLine(log, output.StatusKept), log)
			continue
		}
		assert.NotEmpty(t, logLine(log, "PersistentVolumeClaim/default/data", output.StatusKept), log)
		assert.Empty(t, logLine(log, "PersistentVolumeClaim/default/data", "would prune"), log)
		assert.NotEmpty(t, logLine(log, "would keep 1 stale PersistentVolumeClaim(s)", "--delete-data"), log)
		assert.NotContains(t, log, "WARN")
	}
}

// A stale claim that is no longer in the cluster is not kept: nothing is
// reported, and the entry leaves the record without --delete-data. A claim
// that is still there is kept beside it.
func TestExecute_StaleClaimAlreadyGoneLeavesTheRecord(t *testing.T) {
	withReleasedCLIVersion(t)
	logBuf := captureLog(t)
	cluster := newApplyCluster(
		cliOwnedInstanceWith("demo", "default", configMapEntry("keep"), claimEntry("gone"), claimEntry("library")),
		liveClaim("library"),
	)

	var err error
	stdout := captureStdout(t, func() {
		err = Execute(context.Background(), cluster.request(Options{}, "keep"))
	})
	require.NoError(t, err)
	assert.Contains(t, stdout, "applied")
	assert.Empty(t, cluster.deletes())

	entries, written := cluster.writtenInventory(t)
	require.True(t, written)
	assert.ElementsMatch(t, []string{"keep", "library"}, entryNames(entries), "the absent claim leaves the record, the present one stays")
	log := logBuf.String()
	assert.Empty(t, logLine(log, "PersistentVolumeClaim/default/gone"), "an absent claim is not reported as kept:\n%s", log)
	assert.NotEmpty(t, logLine(log, "keeping 1 stale PersistentVolumeClaim(s)"), log)
}

// With every stale claim gone, the apply reports nothing about claims.
func TestExecute_OnlyAbsentStaleClaimsReportNothing(t *testing.T) {
	withReleasedCLIVersion(t)
	for _, dryRun := range []bool{false, true} {
		logBuf := captureLog(t)
		cluster := newApplyCluster(cliOwnedInstanceWith("demo", "default", configMapEntry("keep"), claimEntry("gone")))
		captureStdout(t, func() {
			require.NoError(t, Execute(context.Background(), cluster.request(Options{DryRun: dryRun}, "keep")))
		})
		assert.Empty(t, logLine(logBuf.String(), "PersistentVolumeClaim"), logBuf.String())
		if !dryRun {
			entries, _ := cluster.writtenInventory(t)
			assert.Equal(t, []string{"keep"}, entryNames(entries))
		}
	}
}

// A stale claim that cannot be read may still hold data: it is kept and
// stays in the record.
func TestExecute_UnreadableStaleClaimIsKept(t *testing.T) {
	withReleasedCLIVersion(t)
	logBuf := captureLog(t)
	cluster := claimCluster()
	cluster.dyn.PrependReactor("get", "persistentvolumeclaims", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "persistentvolumeclaims"}, "data", errors.New("no read access"))
	})

	captureStdout(t, func() {
		require.NoError(t, Execute(context.Background(), cluster.request(Options{}, "keep")))
	})
	entries, _ := cluster.writtenInventory(t)
	assert.ElementsMatch(t, []string{"keep", "data"}, entryNames(entries))
	assert.NotEmpty(t, logLine(logBuf.String(), "PersistentVolumeClaim/default/data", output.StatusKept))
	assert.Equal(t, []string{"configmaps/gone"}, cluster.deletes())
}

func TestSplitDataClaims(t *testing.T) {
	prunable, claims := inventory.SplitDataClaims([]k8sinventory.Entry{
		{Kind: "ConfigMap", Name: "a"},
		{Kind: "PersistentVolumeClaim", Name: "data"},
		{Group: "example.io", Kind: "PersistentVolumeClaim", Name: "lookalike"},
		{Kind: "PersistentVolumeClaim", Name: "cache"},
	})
	assert.Equal(t, []string{"a", "lookalike"}, entryNames(prunable), "a kind of the same name in another group is prunable")
	assert.Equal(t, []string{"data", "cache"}, entryNames(claims), "input order is kept")

	prunable, claims = inventory.SplitDataClaims(nil)
	assert.Empty(t, prunable)
	assert.Empty(t, claims)
}
