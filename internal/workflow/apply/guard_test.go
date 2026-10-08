package apply

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"

	opmexit "github.com/open-platform-model/cli/internal/exit"
)

// otherIdentity is the identity of an instance that is not the one applied.
const otherIdentity = "uuid-of-another-instance"

func terminatingConfigMap(cm *unstructured.Unstructured) *unstructured.Unstructured {
	now := metav1.Now()
	cm.SetDeletionTimestamp(&now)
	cm.SetFinalizers([]string{"example.io/hold"})
	return cm
}

// The ownership guard runs on every apply. An instance with a record that
// starts to render an object the record does not list is refused, before any
// write, when that object exists and is someone else's or is being deleted.
func TestExecute_LaterApplyRefusesWhatTheInstanceDoesNotOwn(t *testing.T) {
	cases := []struct {
		name string
		live *unstructured.Unstructured
		want []string
	}{
		{
			name: "an object OPM does not manage",
			live: liveConfigMap("new", "", "", ""),
			want: []string{"ConfigMap/default/new exists and is not managed by OPM", opmlabels.AnnotationAdopt + "=" + renderIdentity},
		},
		{
			name: "an object of another instance",
			live: liveConfigMap("new", opmlabels.ManagedByCLI, otherIdentity, ""),
			want: []string{"ConfigMap/default/new belongs to module instance " + otherIdentity, opmlabels.AnnotationAdopt + "=" + renderIdentity},
		},
		{
			name: "an object that is being deleted",
			live: terminatingConfigMap(liveConfigMap("new", opmlabels.ManagedByCLI, renderIdentity, "")),
			want: []string{"ConfigMap/default/new is being deleted"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			withReleasedCLIVersion(t)
			logBuf := captureLog(t)
			cluster := newApplyCluster(
				recordWithIdentity(renderIdentity, "app"),
				liveConfigMap("app", opmlabels.ManagedByCLI, renderIdentity, ""),
				c.live,
			)

			err := Execute(context.Background(), cluster.request(Options{}, "app", "new"))

			requireExitCode(t, err, opmexit.ExitGeneralError)
			for _, s := range c.want {
				assert.Contains(t, err.Error(), s)
			}
			assert.Contains(t, err.Error(), "apply stopped before any change")
			assert.NotContains(t, err.Error(), "--force", "no flag lifts the guard")
			assert.NotContains(t, err.Error(), "earlier identity", "the first-apply line is not for an instance with a record")
			assert.Empty(t, cluster.writes(), "nothing is applied, pruned or recorded")
			assert.NotContains(t, logBuf.String(), "applying")
		})
	}
}

// A recorded object that is being deleted refuses every later apply.
func TestExecute_LaterApplyRefusesATerminatingRecordedObject(t *testing.T) {
	withReleasedCLIVersion(t)
	captureLog(t)
	cluster := newApplyCluster(
		recordWithIdentity(renderIdentity, "app"),
		terminatingConfigMap(liveConfigMap("app", opmlabels.ManagedByCLI, renderIdentity, "")),
	)

	err := Execute(context.Background(), cluster.request(Options{}, "app"))

	requireExitCode(t, err, opmexit.ExitGeneralError)
	assert.Contains(t, err.Error(), "ConfigMap/default/app is being deleted")
	assert.Empty(t, cluster.writes())
}

// Every refused object is named, not only the first.
func TestExecute_GuardNamesEveryRefusedObject(t *testing.T) {
	withReleasedCLIVersion(t)
	captureLog(t)
	cluster := newApplyCluster(liveConfigMap("a", "", "", ""), liveConfigMap("b", "", "", ""))

	err := Execute(context.Background(), cluster.request(Options{}, "a", "b"))

	requireExitCode(t, err, opmexit.ExitGeneralError)
	assert.Contains(t, err.Error(), "2 object(s)")
	assert.Contains(t, err.Error(), "ConfigMap/default/a exists")
	assert.Contains(t, err.Error(), "ConfigMap/default/b exists")
	assert.Empty(t, cluster.writes())
}

// A first apply no longer takes over an object of another instance. When the
// instance has no record, the refusal adds what to do if the objects are its
// own under an earlier identity.
func TestExecute_FirstApplyRefusesAnotherInstancesObject(t *testing.T) {
	withReleasedCLIVersion(t)
	captureLog(t)
	cluster := newApplyCluster(
		liveConfigMap("a", opmlabels.ManagedByCLI, otherIdentity, ""),
		liveConfigMap("b", opmlabels.ManagedByLegacy, otherIdentity, ""),
	)

	err := Execute(context.Background(), cluster.request(Options{WarnUnrecorded: true}, "a", "b"))

	requireExitCode(t, err, opmexit.ExitGeneralError)
	assert.Contains(t, err.Error(), "ConfigMap/default/a belongs to module instance "+otherIdentity)
	assert.Contains(t, err.Error(), "ConfigMap/default/b belongs to module instance "+otherIdentity)
	assert.Contains(t, err.Error(), opmlabels.AnnotationAdopt+"="+renderIdentity)
	assert.Contains(t, err.Error(), "own objects under an earlier identity")
	assert.Contains(t, err.Error(), "nothing has to be removed first")
	assert.Contains(t, err.Error(), "apply stopped before any change")
	assert.Empty(t, cluster.writes())
}

// The adopt annotation with this instance's identity is the one override: the
// object is applied and recorded, on a first apply and on a later one.
func TestExecute_AdoptAnnotationLetsTheInstanceTakeAnObject(t *testing.T) {
	cases := []struct {
		name    string
		cluster []*unstructured.Unstructured
	}{
		{
			name:    "foreign object on a first apply",
			cluster: []*unstructured.Unstructured{liveConfigMap("new", "", "", renderIdentity)},
		},
		{
			name: "another instance's object on a later apply",
			cluster: []*unstructured.Unstructured{
				recordWithIdentity(renderIdentity, "app"),
				liveConfigMap("app", opmlabels.ManagedByCLI, renderIdentity, ""),
				liveConfigMap("new", opmlabels.ManagedByCLI, otherIdentity, renderIdentity),
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			withReleasedCLIVersion(t)
			captureLog(t)
			cluster := newApplyCluster(c.cluster...)

			require.NoError(t, Execute(context.Background(), cluster.request(Options{}, "app", "new")))

			assert.Contains(t, cluster.writes(), "patch configmaps new", "the adopted object is applied")
			entries, written := cluster.writtenInventory(t)
			require.True(t, written)
			assert.ElementsMatch(t, []string{"app", "new"}, entryNames(entries), "and recorded")
		})
	}
}

// The annotation does not lift the refusal of an object that is being deleted.
func TestExecute_AdoptAnnotationDoesNotLiftTerminating(t *testing.T) {
	withReleasedCLIVersion(t)
	captureLog(t)
	cluster := newApplyCluster(terminatingConfigMap(liveConfigMap("new", "", "", renderIdentity)))

	err := Execute(context.Background(), cluster.request(Options{}, "new"))

	requireExitCode(t, err, opmexit.ExitGeneralError)
	assert.Empty(t, cluster.writes())
}

// A user's own objects are never refused. An older cli labeled them with the
// legacy managed-by value and the same instance identity; a record may list
// them under an identity the module no longer renders; and the oldest objects
// carry no identity label at all.
func TestExecute_OwnObjectsOfAnOlderCLIAreApplied(t *testing.T) {
	cases := []struct {
		name    string
		cluster []*unstructured.Unstructured
	}{
		{
			name:    "no record: legacy managed-by value and this instance's identity",
			cluster: []*unstructured.Unstructured{liveConfigMap("app", opmlabels.ManagedByLegacy, renderIdentity, "")},
		},
		{
			name:    "no record: legacy managed-by value and no identity label",
			cluster: []*unstructured.Unstructured{liveConfigMap("app", opmlabels.ManagedByLegacy, "", "")},
		},
		{
			name: "recorded: legacy managed-by value and this instance's identity",
			cluster: []*unstructured.Unstructured{
				recordWithIdentity(renderIdentity, "app"),
				liveConfigMap("app", opmlabels.ManagedByLegacy, renderIdentity, ""),
			},
		},
		{
			name: "recorded: the identity changed since the object was applied",
			cluster: []*unstructured.Unstructured{
				recordWithIdentity(oldIdentity, "app"),
				liveConfigMap("app", opmlabels.ManagedByLegacy, oldIdentity, ""),
			},
		},
		{
			name: "recorded: no OPM label left on the object",
			cluster: []*unstructured.Unstructured{
				recordWithIdentity(renderIdentity, "app"),
				liveConfigMap("app", "", "", ""),
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			withReleasedCLIVersion(t)
			captureLog(t)
			cluster := newApplyCluster(c.cluster...)

			require.NoError(t, Execute(context.Background(), cluster.request(Options{}, "app")))

			assert.Contains(t, cluster.writes(), "patch configmaps app")
			entries, written := cluster.writtenInventory(t)
			require.True(t, written)
			assert.Equal(t, []string{"app"}, entryNames(entries))
		})
	}
}

// With RefuseLetGo, which only `opm operator install` sets, an object adopted
// by another instance refuses the apply, and the refusal does not claim that
// nothing was changed when the caller wrote before the apply.
func TestExecute_RefuseLetGoRefusesAnAdoptedObject(t *testing.T) {
	withReleasedCLIVersion(t)
	captureLog(t)
	cluster := newApplyCluster(
		recordWithIdentity(renderIdentity, "app"),
		liveConfigMap("app", opmlabels.ManagedByCLI, renderIdentity, otherIdentity),
	)

	err := Execute(context.Background(), cluster.request(Options{RefuseLetGo: true, AfterCallerWrites: true}, "app"))

	requireExitCode(t, err, opmexit.ExitGeneralError)
	assert.Contains(t, err.Error(), "ConfigMap/default/app was adopted by module instance "+otherIdentity)
	assert.NotContains(t, err.Error(), "before any change")
	assert.Empty(t, cluster.writes())
}
