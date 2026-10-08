package apply

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	opmexit "github.com/open-platform-model/cli/internal/exit"
)

// On a first install, an object the existence check cannot read refuses the
// apply before any change: the check cannot tell whether a foreign object
// holds the name, and the forced apply that follows would take it over.
func TestExecute_FirstInstallRefusesUnreadableObject(t *testing.T) {
	causes := []struct {
		name string
		err  error
		code int
	}{
		{"forbidden", apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "app", errors.New("no access")), opmexit.ExitPermissionDenied},
		{"internal error", apierrors.NewInternalError(errors.New("etcd leader changed")), opmexit.ExitGeneralError},
	}
	for _, cause := range causes {
		t.Run(cause.name, func(t *testing.T) {
			withReleasedCLIVersion(t)
			logBuf := captureLog(t)
			cluster := newApplyCluster() // no record: a first install
			cluster.dyn.PrependReactor("get", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, cause.err
			})

			err := Execute(context.Background(), cluster.request(Options{}, "app"))

			requireExitCode(t, err, cause.code)
			assert.Contains(t, err.Error(), "ConfigMap/app", "the error names the resource")
			assert.ErrorIs(t, err, cause.err, "the cause stays in the chain")
			assert.Empty(t, cluster.writes(), "nothing is applied and no record is written")
			assert.NotContains(t, logBuf.String(), "applying", "the apply never starts")
		})
	}
}

// With a record, the check does not run: an unreadable object does not stop
// a later apply.
func TestExecute_LaterApplySkipsTheExistenceCheck(t *testing.T) {
	withReleasedCLIVersion(t)
	captureLog(t)
	cluster := newApplyCluster(cliOwnedInstance("demo", "default", "app"))
	cluster.dyn.PrependReactor("get", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "app", errors.New("no access"))
	})

	assert.NoError(t, Execute(context.Background(), cluster.request(Options{}, "app")))
}

// --create-namespace creates the namespace before the read-only checks, so a
// refusal after it can only promise that no rendered resource was applied:
// the message makes no wider claim, and no resource or record is written.
func TestExecute_RefusalAfterNamespaceCreateClaimsNoMore(t *testing.T) {
	withReleasedCLIVersion(t)
	captureLog(t)
	cluster := newApplyCluster()
	cluster.dyn.PrependReactor("get", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "app", errors.New("no access"))
	})

	err := Execute(context.Background(), cluster.request(Options{CreateNS: true}, "app"))

	requireExitCode(t, err, opmexit.ExitPermissionDenied)
	assert.Contains(t, err.Error(), "before any rendered resource was applied")
	assert.NotContains(t, err.Error(), "before any change")
	assert.Empty(t, cluster.writes(), "no rendered resource is applied and no record is written")

	var created []string
	for _, a := range cluster.client.Clientset.(*k8sfake.Clientset).Actions() {
		if a.GetVerb() == "create" && a.GetResource().Resource == "namespaces" {
			created = append(created, a.GetResource().Resource)
		}
	}
	assert.Equal(t, []string{"namespaces"}, created, "the namespace the flag asks for is created before the check, as before this change")
}
