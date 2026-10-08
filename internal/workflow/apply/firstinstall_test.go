package apply

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sfake "k8s.io/client-go/kubernetes/fake"

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

	require.NoError(t, Execute(context.Background(), cluster.request(Options{WarnUnrecorded: true}, "a", "b", "c")))

	log := logBuf.String()
	assert.Equal(t, 1, strings.Count(log, firstInstallWarning), "one warning")
	assert.Contains(t, log, "2 of 3 rendered resource(s)")
	assert.Contains(t, log, "no ModuleInstance record")
	assert.Contains(t, log, "prunes nothing")
	assert.Contains(t, log, `the Secret "opm.demo.uuid-1" in this namespace still lists what it owned: keep it`, "the warning names the Secret and says to keep it")
	assert.NotContains(t, log, "v1.0.0-beta.10", "this run writes the record, after which the migrating release deletes the Secret unread: the warning does not send the user there")
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

		require.NoError(t, Execute(context.Background(), cluster.request(Options{WarnUnrecorded: true}, "a")))
		assert.NotContains(t, logBuf.String(), firstInstallWarning)
	})
	t.Run("instance with a record", func(t *testing.T) {
		withReleasedCLIVersion(t)
		logBuf := captureLog(t)
		cluster := newApplyCluster(cliOwnedInstance("demo", "default", "a"), liveManagedConfigMap("a"))

		require.NoError(t, Execute(context.Background(), cluster.request(Options{WarnUnrecorded: true}, "a")))
		assert.NotContains(t, logBuf.String(), firstInstallWarning)
		for _, a := range cluster.client.Clientset.(*k8sfake.Clientset).Actions() {
			assert.NotEqual(t, "secrets", a.GetResource().Resource, "an apply with a record makes no request on Secrets: %s", a.GetVerb())
		}
	})
}

// A dry run of such a first install warns too, before anything is written,
// and names the release to apply with first.
func TestExecute_DryRunFirstInstallOverManagedResourcesWarns(t *testing.T) {
	withReleasedCLIVersion(t)
	logBuf := captureLog(t)
	cluster := newApplyCluster(liveManagedConfigMap("a"), liveManagedConfigMap("b"))

	require.NoError(t, Execute(context.Background(), cluster.request(Options{WarnUnrecorded: true, DryRun: true}, "a", "b", "c")))

	log := logBuf.String()
	assert.Equal(t, 1, strings.Count(log, firstInstallWarning), "one warning")
	assert.Contains(t, log, "2 of 3 rendered resource(s)")
	assert.Contains(t, log, "A real apply would update them in place")
	assert.Contains(t, log, "apply the instance once with opm v1.0.0-beta.10 before you apply it with this release")
	assert.Less(t, strings.Index(log, firstInstallWarning), strings.Index(log, "dry run - no changes will be made"), "the warning comes first")
	_, written := cluster.writtenInventory(t)
	assert.False(t, written, "a dry run writes no record")
}

// A dry run refuses nothing in the first-install look: an object a real run
// would refuse ends the look without a warning and without an error.
func TestExecute_DryRunFirstInstallLookRefusesNothing(t *testing.T) {
	withReleasedCLIVersion(t)
	logBuf := captureLog(t)
	foreign := renderedConfigMap("a")
	foreign.SetLabels(nil) // exists, no managed-by label
	cluster := newApplyCluster(foreign)

	require.NoError(t, Execute(context.Background(), cluster.request(Options{WarnUnrecorded: true, DryRun: true}, "a")))
	assert.NotContains(t, logBuf.String(), firstInstallWarning)
}

// A caller that does not ask for the warning gets none: `opm operator
// install` applies the render's CRDs before the workflow runs.
func TestExecute_NoFirstInstallWarningUnlessAsked(t *testing.T) {
	withReleasedCLIVersion(t)
	logBuf := captureLog(t)
	cluster := newApplyCluster(liveManagedConfigMap("a"))

	require.NoError(t, Execute(context.Background(), cluster.request(Options{}, "a")))
	assert.NotContains(t, logBuf.String(), firstInstallWarning)
}
