package apply

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"
)

// firstInstallWarning is the text only the first-install warning carries.
const firstInstallWarning = "already exist and are managed by OPM"

// liveManagedConfigMap is a ConfigMap an earlier OPM apply left in the
// cluster: it carries the cli's managed-by label.
func liveManagedConfigMap(name string) *unstructured.Unstructured {
	cm := renderedConfigMap(name)
	labels := cm.GetLabels()
	labels[opmlabels.ManagedBy] = opmlabels.ManagedByCLI
	cm.SetLabels(labels)
	return cm
}

// A first install that finds rendered resources already in the cluster under
// OPM management says so once, before it applies, and goes on: the resources
// are applied and recorded, and nothing is deleted.
func TestExecute_FirstInstallOverManagedResourcesWarns(t *testing.T) {
	withReleasedCLIVersion(t)
	logBuf := captureLog(t)
	cluster := newApplyCluster(liveManagedConfigMap("a"), liveManagedConfigMap("b")) // no record

	require.NoError(t, Execute(context.Background(), cluster.request(Options{}, "a", "b", "c")))

	log := logBuf.String()
	assert.Equal(t, 1, strings.Count(log, firstInstallWarning), "one warning")
	assert.Contains(t, log, "2 of 3 rendered resource(s)")
	assert.Contains(t, log, "no ModuleInstance record")
	assert.Contains(t, log, "prunes nothing")
	assert.Contains(t, log, "v1.0.0-beta.10", "the warning names the release that migrates")
	assert.Less(t, strings.Index(log, firstInstallWarning), strings.Index(log, "applying 3 resources"), "the warning comes before the apply")

	entries, written := cluster.writtenInventory(t)
	require.True(t, written, "the record is written")
	assert.ElementsMatch(t, []string{"a", "b", "c"}, entryNames(entries))
	for _, w := range cluster.writes() {
		assert.NotContains(t, w, "delete", "nothing is deleted")
	}
}

// No rendered resource exists yet, or the instance has a record: no warning.
func TestExecute_NoFirstInstallWarningWithoutCause(t *testing.T) {
	t.Run("clean first install", func(t *testing.T) {
		withReleasedCLIVersion(t)
		logBuf := captureLog(t)
		cluster := newApplyCluster()

		require.NoError(t, Execute(context.Background(), cluster.request(Options{}, "a")))
		assert.NotContains(t, logBuf.String(), firstInstallWarning)
	})
	t.Run("instance with a record", func(t *testing.T) {
		withReleasedCLIVersion(t)
		logBuf := captureLog(t)
		cluster := newApplyCluster(cliOwnedInstance("demo", "default", "a"), liveManagedConfigMap("a"))

		require.NoError(t, Execute(context.Background(), cluster.request(Options{}, "a")))
		assert.NotContains(t, logBuf.String(), firstInstallWarning)
	})
}
