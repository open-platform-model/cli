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

	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"

	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/inventory"
)

// A dry run runs the guard of the real run: for an object the real run
// refuses it prints one "would refuse" line with the library's reason, exits
// with the code of the real refusal, and sends nothing to the server, not
// even the server-side dry run. The real run on the same cluster refuses
// with the same code, which is what makes the dry run a preview.
func TestExecute_DryRunPreviewsTheGuardRefusal(t *testing.T) {
	cases := map[string]struct {
		objs       []*unstructured.Unstructured
		wantReason string
		wantHint   bool
	}{
		"first apply, an object OPM does not manage": {
			objs:       []*unstructured.Unstructured{liveConfigMap("new", "", "", "")},
			wantReason: "ConfigMap/default/new exists and is not managed by OPM",
		},
		"first apply, an object of another instance": {
			objs:       []*unstructured.Unstructured{liveConfigMap("new", opmlabels.ManagedByCLI, otherIdentity, "")},
			wantReason: "ConfigMap/default/new belongs to module instance " + otherIdentity,
			wantHint:   true,
		},
		"later apply, an object of another instance": {
			objs:       []*unstructured.Unstructured{recordWithIdentity(renderIdentity, "app"), liveConfigMap("new", opmlabels.ManagedByCLI, otherIdentity, "")},
			wantReason: "ConfigMap/default/new belongs to module instance " + otherIdentity,
		},
		"later apply, a recorded object that is being deleted": {
			objs:       []*unstructured.Unstructured{recordWithIdentity(renderIdentity, "app", "new"), terminatingConfigMap(liveConfigMap("new", opmlabels.ManagedByCLI, renderIdentity, ""))},
			wantReason: "ConfigMap/default/new is being deleted",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			withReleasedCLIVersion(t)
			logBuf := captureLog(t)
			cluster := newApplyCluster(tc.objs...)

			err := Execute(context.Background(), cluster.request(Options{DryRun: true, WarnUnrecorded: true}, "app", "new"))

			requireExitCode(t, err, opmexit.ExitGeneralError)
			log := logBuf.String()
			line := logLine(log, "ConfigMap/default/new", "would refuse")
			require.NotEmpty(t, line, "one line names the refused object: %s", log)
			assert.Contains(t, line, tc.wantReason, "with the reason and the owner")
			assert.Equal(t, 1, strings.Count(log, "would refuse"), "the object that passes gets no line")
			assert.Contains(t, err.Error(), "a real apply would be refused: 1 object(s)")
			assert.Contains(t, err.Error(), "the dry run changed nothing")
			assert.NotContains(t, err.Error(), "apply stopped", "the words of the real refusal are not reused")
			assert.Equal(t, tc.wantHint, strings.Contains(err.Error(), "under an earlier identity"))
			var refusal *inventory.GuardRefusalError
			assert.ErrorAs(t, err, &refusal, "the refusal stays in the chain")
			assert.Empty(t, cluster.writes(), "nothing is sent, the server-side dry run included")
			assert.NotContains(t, log, "applying", "the dry run stops where the real run stops")

			cluster.dyn.ClearActions()
			realErr := Execute(context.Background(), cluster.request(Options{WarnUnrecorded: true}, "app", "new"))
			requireExitCode(t, realErr, opmexit.ExitGeneralError)
			assert.Contains(t, realErr.Error(), tc.wantReason, "the real run refuses for the same reason")
		})
	}
}

// Every refused object gets its own line, in one run.
func TestExecute_DryRunListsEveryRefusedObject(t *testing.T) {
	withReleasedCLIVersion(t)
	logBuf := captureLog(t)
	cluster := newApplyCluster(liveConfigMap("a", "", "", ""), liveConfigMap("b", "", "", ""))

	err := Execute(context.Background(), cluster.request(Options{DryRun: true}, "a", "b"))

	requireExitCode(t, err, opmexit.ExitGeneralError)
	assert.NotEmpty(t, logLine(logBuf.String(), "ConfigMap/default/a", "would refuse"))
	assert.NotEmpty(t, logLine(logBuf.String(), "ConfigMap/default/b", "would refuse"))
	assert.Contains(t, err.Error(), "2 object(s)")
	assert.Contains(t, logBuf.String(), opmlabels.AnnotationAdopt+"="+renderIdentity, "the line says how to adopt")
}

// An object the guard cannot read fails the dry run as it fails the real
// run: same cause in the chain, same exit code.
func TestExecute_DryRunFailsOnAnUnreadableObject(t *testing.T) {
	causes := map[string]struct {
		err  error
		code int
	}{
		"forbidden":      {apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "app", errors.New("no access")), opmexit.ExitPermissionDenied},
		"unavailable":    {apierrors.NewServiceUnavailable("down"), opmexit.ExitConnectivityError},
		"internal error": {apierrors.NewInternalError(errors.New("etcd leader changed")), opmexit.ExitGeneralError},
	}
	for name, cause := range causes {
		t.Run(name, func(t *testing.T) {
			withReleasedCLIVersion(t)
			captureLog(t)
			cluster := newApplyCluster(cliOwnedInstance("demo", "default", "app"))
			cluster.dyn.PrependReactor("get", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, cause.err
			})

			err := Execute(context.Background(), cluster.request(Options{DryRun: true}, "app"))

			requireExitCode(t, err, cause.code)
			assert.ErrorIs(t, err, cause.err)
			assert.Contains(t, err.Error(), "ConfigMap/app")
			assert.Contains(t, err.Error(), "the dry run changed nothing")
			assert.Empty(t, cluster.writes())
		})
	}
}

// An object another instance adopted is not a refusal on a dry run either:
// one "would skip" line, the object is left out of the server-side dry run,
// and the other objects are sent as before.
func TestExecute_DryRunSkipsAnObjectAdoptedByAnotherInstance(t *testing.T) {
	withReleasedCLIVersion(t)
	logBuf := captureLog(t)
	cluster := newApplyCluster(
		recordWithIdentity(renderIdentity, "app", "settings"),
		liveConfigMap("app", opmlabels.ManagedByCLI, renderIdentity, ""),
		liveConfigMap("settings", opmlabels.ManagedByCLI, renderIdentity, otherIdentity),
	)

	require.NoError(t, Execute(context.Background(), cluster.request(Options{DryRun: true}, "app", "settings")))

	log := logBuf.String()
	line := logLine(log, "ConfigMap/default/settings", "would skip")
	require.NotEmpty(t, line, log)
	assert.Contains(t, line, "was adopted by module instance "+otherIdentity, "the line names the adopting instance")
	assert.Equal(t, []string{"patch configmaps app"}, cluster.writes(), "only the other object goes to the server-side dry run")
	assert.Contains(t, log, "dry run complete: 1 resources would be applied")
	_, written := cluster.writtenInventory(t)
	assert.False(t, written, "a dry run writes no record")
}

// When every rendered object is adopted elsewhere the dry run sends nothing
// and says that nothing would be applied.
func TestExecute_DryRunLetsGoOfItsOnlyObject(t *testing.T) {
	withReleasedCLIVersion(t)
	logBuf := captureLog(t)
	cluster := newApplyCluster(
		recordWithIdentity(renderIdentity, "settings"),
		liveConfigMap("settings", opmlabels.ManagedByCLI, renderIdentity, otherIdentity),
	)

	require.NoError(t, Execute(context.Background(), cluster.request(Options{DryRun: true}, "settings")))

	assert.Empty(t, cluster.writes())
	assert.NotEmpty(t, logLine(logBuf.String(), "ConfigMap/default/settings", "would skip"))
	assert.Contains(t, logBuf.String(), "dry run complete: nothing would be applied: all 1 rendered resource(s) are adopted by another instance")
}
