package apply

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"
)

// setAdopt rewrites the adopt annotation of the live ConfigMap in "default".
func (c *applyCluster) setAdopt(t *testing.T, name, identity string) {
	t.Helper()
	res := c.dyn.Resource(schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}).Namespace("default")
	cm, err := res.Get(context.Background(), name, metav1.GetOptions{})
	require.NoError(t, err)
	cm.SetAnnotations(map[string]string{opmlabels.AnnotationAdopt: identity})
	_, err = res.Update(context.Background(), cm, metav1.UpdateOptions{})
	require.NoError(t, err)
	c.dyn.ClearActions()
}

// A recorded object whose adopt annotation names another instance is let go:
// it is not applied, not deleted and not recorded, the apply warns once and
// succeeds, and the other objects are applied as before.
func TestExecute_LetsGoOfAnObjectAdoptedByAnotherInstance(t *testing.T) {
	withReleasedCLIVersion(t)
	logBuf := captureLog(t)
	cluster := newApplyCluster(
		recordWithIdentity(renderIdentity, "app", "settings"),
		liveConfigMap("app", opmlabels.ManagedByCLI, renderIdentity, ""),
		liveConfigMap("settings", opmlabels.ManagedByCLI, renderIdentity, otherIdentity),
	)

	require.NoError(t, Execute(context.Background(), cluster.request(Options{}, "app", "settings")))

	writes := cluster.writes()
	assert.Contains(t, writes, "patch configmaps app", "the instance's other object is applied")
	assert.NotContains(t, writes, "patch configmaps settings", "the adopted object is not applied")
	assert.NotContains(t, writes, "delete configmaps settings", "and not deleted")
	entries, written := cluster.writtenInventory(t)
	require.True(t, written)
	assert.Equal(t, []string{"app"}, entryNames(entries), "the record no longer holds it")

	log := logBuf.String()
	assert.Equal(t, 1, strings.Count(log, "ConfigMap/default/settings was adopted by module instance "+otherIdentity), "one warning")
	assert.Contains(t, log, "drops it from its inventory")
	assert.Contains(t, log, opmlabels.AnnotationAdopt+"="+renderIdentity, "the warning says how to take it back")
	assert.Contains(t, log, "applying 1 resources")
	assert.NotContains(t, log, "left behind", "the prune never sees it")
}

// After the hand-over the object carries the adopting instance's identity
// and is in no record of this instance. The instance still renders it, does
// not take it back and does not fail. Setting the annotation to this
// instance's identity reverses the hand-over: the object is applied and
// recorded again.
func TestExecute_HandOverStaysUntilTheAnnotationNamesThisInstance(t *testing.T) {
	withReleasedCLIVersion(t)
	captureLog(t)
	cluster := newApplyCluster(
		recordWithIdentity(renderIdentity, "app"),
		liveConfigMap("app", opmlabels.ManagedByCLI, renderIdentity, ""),
		liveConfigMap("settings", opmlabels.ManagedByCLI, otherIdentity, otherIdentity),
	)

	require.NoError(t, Execute(context.Background(), cluster.request(Options{}, "app", "settings")))
	assert.NotContains(t, cluster.writes(), "patch configmaps settings")
	entries, _ := cluster.writtenInventory(t)
	assert.Equal(t, []string{"app"}, entryNames(entries))

	cluster.setAdopt(t, "settings", renderIdentity)

	require.NoError(t, Execute(context.Background(), cluster.request(Options{}, "app", "settings")))
	assert.Contains(t, cluster.writes(), "patch configmaps settings", "applied again")
	entries, _ = cluster.writtenInventory(t)
	assert.ElementsMatch(t, []string{"app", "settings"}, entryNames(entries), "and recorded again")
}
