package apply

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stesting "k8s.io/client-go/testing"

	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"

	opmexit "github.com/open-platform-model/cli/internal/exit"
)

// captureStdout runs fn and returns what it wrote to standard output, where
// the success line goes.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()

	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}

// A stale resource whose delete fails stays in the written inventory, is
// named in the output, and fails the command; the stale resource that was
// deleted leaves the inventory. The kept entry is stale again on the next
// apply, so a re-run retries it.
func TestExecute_FailedPruneKeepsTheEntryAndFails(t *testing.T) {
	withReleasedCLIVersion(t)
	logBuf := captureLog(t)

	cluster := newApplyCluster(
		cliOwnedInstance("demo", "default", "keep", "stuck", "gone"),
		renderedConfigMap("stuck"), renderedConfigMap("gone"),
	)
	denied := apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "stuck", errors.New("no delete access"))
	cluster.dyn.PrependReactor("delete", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		if action.(k8stesting.DeleteAction).GetName() == "stuck" {
			return true, nil, denied
		}
		return false, nil, nil
	})

	var err error
	stdout := captureStdout(t, func() {
		err = Execute(context.Background(), cluster.request(Options{}, "keep"))
	})

	requireExitCode(t, err, opmexit.ExitGeneralError)
	assert.NotContains(t, stdout, "applied", "no success line after a failed prune")
	assert.NotContains(t, stdout, "up to date", "no success line after a failed prune")

	names, written := cluster.writtenInventory(t)
	require.True(t, written, "the record is still written")
	assert.ElementsMatch(t, []string{"keep", "stuck"}, names,
		"the record holds the current entry and the entry prune failed to delete, not the deleted one")

	var failureLine string
	for _, line := range strings.Split(logBuf.String(), "\n") {
		if strings.Contains(line, "ConfigMap/default/stuck") {
			failureLine = line
		}
	}
	require.NotEmpty(t, failureLine, "the output names the resource that was not pruned:\n%s", logBuf.String())
	assert.Contains(t, failureLine, "prune failed")
	assert.Contains(t, failureLine, "no delete access", "the line carries the delete error")
	for _, line := range strings.Split(logBuf.String(), "\n") {
		if strings.Contains(line, "ConfigMap/default/gone") {
			assert.NotContains(t, line, "prune failed", "the pruned resource is not reported as failed")
		}
	}
	assert.Contains(t, logBuf.String(), "run apply again", "the output says a re-run retries the prune")

	written2 := make([]k8sinventory.Entry, 0, len(names))
	for _, n := range names {
		written2 = append(written2, k8sinventory.Entry{Kind: "ConfigMap", Namespace: "default", Name: n, Version: "v1"})
	}
	stale := ComputeStaleInventorySet(written2, CurrentInventoryEntries(cluster.request(Options{}, "keep").Result.Resources))
	require.Len(t, stale, 1, "the kept entry is stale again on the next apply")
	assert.Equal(t, "stuck", stale[0].Name)
}

// A prune with no failure keeps the earlier outcome: the stale entry leaves
// the record, the success line prints, and the command passes.
func TestExecute_SuccessfulPruneDropsTheEntry(t *testing.T) {
	withReleasedCLIVersion(t)
	captureLog(t)
	cluster := newApplyCluster(cliOwnedInstance("demo", "default", "keep", "gone"), renderedConfigMap("gone"))

	var err error
	stdout := captureStdout(t, func() {
		err = Execute(context.Background(), cluster.request(Options{}, "keep"))
	})

	require.NoError(t, err)
	assert.Contains(t, stdout, "applied")
	names, written := cluster.writtenInventory(t)
	require.True(t, written)
	assert.Equal(t, []string{"keep"}, names)
}
