package operator

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stesting "k8s.io/client-go/testing"
)

// managerRole is a ClusterRole of the fixture render.
const managerRole = "opm-operator-manager-role"

// denyRoleReads makes every read of managerRole from the nth on answer
// Forbidden. PlanInstall reads each object it applies three times: in the
// terminating wait, in the migration proof and in the apply guard. n picks
// the check that meets the denial.
func denyRoleReads(fc *fakeCluster, n int32) {
	var reads atomic.Int32
	fc.fake.PrependReactor("get", "clusterroles", func(action k8stesting.Action) (bool, runtime.Object, error) {
		if action.(k8stesting.GetAction).GetName() != managerRole || reads.Add(1) < n {
			return false, nil, nil
		}
		return true, nil, apierrors.NewForbidden(
			schema.GroupResource{Group: "rbac.authorization.k8s.io", Resource: "clusterroles"}, managerRole, errors.New("no get"))
	})
}

// `opm operator install` refuses an object it cannot read, and writes
// nothing. The check that meets the failed read decides the error type, and
// the command maps the type to the exit code (TestInstallErrorMapping in
// internal/cmd/operator holds one row per shape below). A read the apply
// guard fails is a *GuardError and exits 2, like the guard's other refusals.
// An object that is unreadable from the start never reaches the guard: the
// terminating wait reads it first, and its error exits by the API error (4
// for Forbidden), as does the migration proof's.
func TestPlanInstall_UnreadableObjectRefuses(t *testing.T) {
	cases := []struct {
		name      string
		firstDeny int32
		want      string
		check     func(*testing.T, error)
	}{
		{
			name: "every read is denied: the terminating wait refuses", firstDeny: 1,
			want: "checking ClusterRole/" + managerRole + " before apply",
			check: func(t *testing.T, err error) {
				var ge *GuardError
				var me *MigrationReadError
				assert.False(t, errors.As(err, &ge) || errors.As(err, &me), "a plain wrapped API error")
			},
		},
		{
			name: "denied from the second read: the migration proof refuses", firstDeny: 2,
			want: "operator migration refused: cannot read ClusterRole/" + managerRole,
			check: func(t *testing.T, err error) {
				var me *MigrationReadError
				require.ErrorAs(t, err, &me)
			},
		},
		{
			name: "denied at the third read: the apply guard refuses", firstDeny: 3,
			want: "cannot check whether ClusterRole/" + managerRole,
			check: func(t *testing.T, err error) {
				var ge *GuardError
				require.ErrorAs(t, err, &ge)
				assert.Contains(t, err.Error(), "apply stopped before any change")
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fc := newFakeCluster(t)
			denyRoleReads(fc, c.firstDeny)
			r := &fakeRender{objs: moduleObjects(renderOpts{})}

			_, err := PlanInstall(context.Background(), newEnv(fc, r), testResolution("v0.1.0"), defaultTarget, PlanOptions{Timeout: time.Second})

			require.Error(t, err)
			assert.Contains(t, err.Error(), c.want)
			assert.True(t, apierrors.IsForbidden(err), "the read error stays in the chain")
			c.check(t, err)
			assert.Empty(t, fc.Writes(), "a refused install writes nothing")
		})
	}
}
