package apply

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	opmexit "github.com/open-platform-model/cli/internal/exit"
)

// clusterRole is a rendered cluster-scoped object: the kind of object the
// existence check can find on a cluster whose instance namespace is missing.
func clusterRole(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "rbac.authorization.k8s.io/v1",
		"kind":       "ClusterRole",
		"metadata":   map[string]any{"name": name},
	}}
}

// namespaceCreates counts the Namespace creates the typed clientset received.
// The fake clientset of newApplyCluster holds no namespace, so the instance
// namespace "default" is missing unless a test adds it.
func namespaceCreates(c *applyCluster) int {
	n := 0
	for _, a := range c.client.Clientset.(*k8sfake.Clientset).Actions() {
		if a.GetVerb() == "create" && a.GetResource().Resource == "namespaces" {
			n++
		}
	}
	return n
}

// With --create-namespace and a missing namespace, every check that can
// refuse the apply runs before the namespace is created: a refused apply has
// made no create call and no other write.
func TestExecute_RefusalCreatesNoNamespace(t *testing.T) {
	forbidden := func(resource, name string) error {
		return apierrors.NewForbidden(schema.GroupResource{Resource: resource}, name, errors.New("no access"))
	}
	terminating := clusterRole("demo-role")
	terminating.SetDeletionTimestamp(&metav1.Time{Time: metav1.Now().Time})

	cases := []struct {
		name    string
		cluster []*unstructured.Unstructured
		setup   func(*applyCluster)
		code    int
		want    string
	}{
		{
			name: "cluster gate: no ModuleInstance CRD",
			setup: func(c *applyCluster) {
				c.dyn.PrependReactor("get", "customresourcedefinitions", func(a k8stesting.Action) (bool, runtime.Object, error) {
					return true, nil, apierrors.NewNotFound(schema.GroupResource{Resource: "customresourcedefinitions"}, a.(k8stesting.GetAction).GetName())
				})
			},
			code: opmexit.ExitValidationError,
			want: "ModuleInstance CRD not found",
		},
		{
			name: "status permission denied",
			setup: func(c *applyCluster) {
				c.client.Clientset.(*k8sfake.Clientset).PrependReactor("create", "selfsubjectaccessreviews", func(k8stesting.Action) (bool, runtime.Object, error) {
					return true, &authorizationv1.SelfSubjectAccessReview{Status: authorizationv1.SubjectAccessReviewStatus{Allowed: false}}, nil
				})
			},
			code: opmexit.ExitPermissionDenied,
			want: "moduleinstances/status",
		},
		{
			name: "existence check: unreadable object",
			setup: func(c *applyCluster) {
				c.dyn.PrependReactor("get", "clusterroles", func(k8stesting.Action) (bool, runtime.Object, error) {
					return true, nil, forbidden("clusterroles", "demo-role")
				})
			},
			code: opmexit.ExitPermissionDenied,
			want: "cannot check whether ClusterRole/demo-role",
		},
		{
			name:    "existence check: untracked cluster-scoped object",
			cluster: []*unstructured.Unstructured{clusterRole("demo-role")},
			code:    opmexit.ExitGeneralError,
			want:    "ClusterRole/demo-role",
		},
		{
			name:    "existence check: terminating cluster-scoped object",
			cluster: []*unstructured.Unstructured{terminating},
			code:    opmexit.ExitGeneralError,
			want:    "is terminating",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			withReleasedCLIVersion(t)
			logBuf := captureLog(t)
			cluster := newApplyCluster(c.cluster...)
			if c.setup != nil {
				c.setup(cluster)
			}
			req := cluster.request(Options{CreateNS: true}, "app")
			req.Result.Resources = append(req.Result.Resources, clusterRole("demo-role"))

			err := Execute(context.Background(), req)

			requireExitCode(t, err, c.code)
			assert.Contains(t, err.Error(), c.want)
			if strings.HasPrefix(c.name, "existence check") {
				assert.Contains(t, err.Error(), "apply stopped before any change")
			}
			assert.NotContains(t, err.Error(), "rendered resource was applied", "the text carries no namespace caveat")
			assert.Zero(t, namespaceCreates(cluster), "a refused apply creates no namespace")
			assert.Empty(t, cluster.writes(), "nothing is applied and no record is written")
			assert.NotContains(t, logBuf.String(), `namespace "default" created`)
		})
	}
}

// When no check refuses, the namespace is created after the checks and before
// the first rendered resource is applied, and the apply ends as a first
// install: the record is written at revision 1.
func TestExecute_CreateNamespaceComesBeforeTheFirstApply(t *testing.T) {
	withReleasedCLIVersion(t)
	logBuf := captureLog(t)
	cluster := newApplyCluster()

	var mu sync.Mutex
	var calls []string
	note := func(call string) {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, call)
	}
	cluster.dyn.PrependReactor("*", "*", func(a k8stesting.Action) (bool, runtime.Object, error) {
		res := a.GetResource().Resource
		if a.GetSubresource() != "" {
			res += "/" + a.GetSubresource()
		}
		note(a.GetVerb() + " " + res)
		return false, nil, nil
	})
	cluster.client.Clientset.(*k8sfake.Clientset).PrependReactor("*", "*", func(a k8stesting.Action) (bool, runtime.Object, error) {
		note(a.GetVerb() + " " + a.GetResource().Resource)
		return false, nil, nil
	})

	require.NoError(t, Execute(context.Background(), cluster.request(Options{CreateNS: true}, "app")))

	mu.Lock()
	defer mu.Unlock()
	create := indexOf(calls, "create namespaces")
	require.GreaterOrEqual(t, create, 0, "the namespace is created: %v", calls)
	assert.Equal(t, "get namespaces", calls[0], "whether the namespace exists is read first")
	for _, check := range []string{"get customresourcedefinitions", "create selfsubjectaccessreviews"} {
		at := indexOf(calls, check)
		require.GreaterOrEqual(t, at, 0, "%s ran: %v", check, calls)
		assert.Less(t, at, create, "%s runs before the namespace is created", check)
	}
	for _, read := range []string{"get moduleinstances", "get configmaps"} {
		assert.NotContains(t, calls[:create], read, "no read is sent into the namespace while it is missing")
	}
	for i, call := range calls {
		if call == "patch configmaps" || call == "patch moduleinstances" || call == "patch moduleinstances/status" {
			assert.Greater(t, i, create, "%s comes after the namespace create", call)
		}
	}
	assert.Equal(t, 1, namespaceCreates(cluster))
	assert.Equal(t, 1, cluster.writtenRevision(t))
	assert.Contains(t, logBuf.String(), `namespace "default" created`)
}

// A namespace that exists is not created again.
func TestExecute_CreateNamespaceLeavesAnExistingNamespaceAlone(t *testing.T) {
	withReleasedCLIVersion(t)
	logBuf := captureLog(t)
	cluster := newApplyCluster()
	_, err := cluster.client.Clientset.CoreV1().Namespaces().Create(context.Background(),
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}}, metav1.CreateOptions{})
	require.NoError(t, err)

	require.NoError(t, Execute(context.Background(), cluster.request(Options{CreateNS: true}, "app")))

	assert.Equal(t, 1, namespaceCreates(cluster), "only the seeding create of this test")
	assert.NotContains(t, logBuf.String(), `namespace "default" created`)
	assert.Equal(t, 1, cluster.writtenRevision(t))
}

// An apply with nothing to apply and no record still creates the namespace
// the flag asks for, as it did when the create was the first step.
func TestExecute_CreateNamespaceWithNothingToApply(t *testing.T) {
	withReleasedCLIVersion(t)
	logBuf := captureLog(t)
	cluster := newApplyCluster()

	require.NoError(t, Execute(context.Background(), cluster.request(Options{CreateNS: true})))

	assert.Equal(t, 1, namespaceCreates(cluster))
	assert.Contains(t, logBuf.String(), "no resources to apply")
	assert.Empty(t, cluster.writes())
}

// A caller whose rights in the instance namespace come from a RoleBinding
// gets Forbidden, not NotFound, for a read in a namespace that does not exist
// yet. The apply sends no such read: a missing namespace holds no record and
// no resource, so the apply goes on as a first install.
func TestExecute_MissingNamespaceIsNotRead(t *testing.T) {
	withReleasedCLIVersion(t)
	captureLog(t)
	cluster := newApplyCluster()
	created := false
	cluster.client.Clientset.(*k8sfake.Clientset).PrependReactor("create", "namespaces", func(k8stesting.Action) (bool, runtime.Object, error) {
		created = true
		return false, nil, nil
	})
	denyUntilCreated := func(a k8stesting.Action) (bool, runtime.Object, error) {
		if created {
			return false, nil, nil
		}
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: a.GetResource().Resource}, "", errors.New("no role binding yet"))
	}
	cluster.dyn.PrependReactor("get", "moduleinstances", denyUntilCreated)
	cluster.dyn.PrependReactor("get", "configmaps", denyUntilCreated)

	require.NoError(t, Execute(context.Background(), cluster.request(Options{CreateNS: true}, "app")))

	assert.Equal(t, 1, namespaceCreates(cluster))
	assert.Equal(t, 1, cluster.writtenRevision(t), "a first install")
}

// A failed read of the namespace stops the apply first, and a failed create
// stops it before any resource is applied; both exit by the cause.
func TestExecute_NamespaceReadOrCreateFails(t *testing.T) {
	for _, verb := range []string{"get", "create"} {
		t.Run(verb, func(t *testing.T) {
			withReleasedCLIVersion(t)
			captureLog(t)
			cluster := newApplyCluster()
			cluster.client.Clientset.(*k8sfake.Clientset).PrependReactor(verb, "namespaces", func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "namespaces"}, "default", errors.New("no access"))
			})

			err := Execute(context.Background(), cluster.request(Options{CreateNS: true}, "app"))

			requireExitCode(t, err, opmexit.ExitPermissionDenied)
			assert.Contains(t, err.Error(), `namespace "default"`)
			assert.Empty(t, cluster.writes(), "nothing is applied and no record is written")
			if verb == "get" {
				assert.Empty(t, cluster.dyn.Actions(), "no other step ran")
			}
		})
	}
}

// A namespace that somebody else created between the read and the create
// stops the apply: the checks looked at nothing inside it.
func TestExecute_NamespaceAppearedDuringTheChecks(t *testing.T) {
	withReleasedCLIVersion(t)
	captureLog(t)
	cluster := newApplyCluster()
	cluster.client.Clientset.(*k8sfake.Clientset).PrependReactor("create", "namespaces", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewAlreadyExists(schema.GroupResource{Resource: "namespaces"}, "default")
	})

	err := Execute(context.Background(), cluster.request(Options{CreateNS: true}, "app"))

	requireExitCode(t, err, opmexit.ExitGeneralError)
	assert.Contains(t, err.Error(), `namespace "default" was created by someone else`)
	assert.Contains(t, err.Error(), "Run the command again")
	assert.Empty(t, cluster.writes())
}

// A caller that wrote to the cluster before the apply (opm operator install)
// gets the refusals without the claim that nothing was changed.
func TestExecute_RefusalAfterCallerWritesClaimsNothing(t *testing.T) {
	reads := map[string]string{"moduleinstances": "cannot read the ModuleInstance record", "configmaps": "cannot check whether ConfigMap/app"}
	for resource, want := range reads {
		t.Run(resource, func(t *testing.T) {
			withReleasedCLIVersion(t)
			captureLog(t)
			cluster := newApplyCluster()
			cluster.dyn.PrependReactor("get", resource, func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: resource}, "x", errors.New("no access"))
			})

			err := Execute(context.Background(), cluster.request(Options{AfterCallerWrites: true}, "app"))

			requireExitCode(t, err, opmexit.ExitPermissionDenied)
			assert.Contains(t, err.Error(), want)
			assert.NotContains(t, err.Error(), "before any change")
		})
	}
}

func indexOf(calls []string, call string) int {
	for i, c := range calls {
		if c == call {
			return i
		}
	}
	return -1
}

// A module that renders its own instance Namespace, applied with
// --create-namespace into a cluster without it: the existence check runs
// while the namespace is missing, so it finds no untracked Namespace; the
// namespace is then created and the rendered Namespace is applied over it.
func TestExecute_CreateNamespaceWithARenderedInstanceNamespace(t *testing.T) {
	withReleasedCLIVersion(t)
	captureLog(t)
	cluster := newApplyCluster()
	rendered := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "Namespace", "metadata": map[string]any{"name": "default"},
	}}
	cluster.dyn.PrependReactor("patch", "namespaces", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, rendered, nil
	})
	req := cluster.request(Options{CreateNS: true}, "app")
	req.Result.Resources = append(req.Result.Resources, rendered)

	require.NoError(t, Execute(context.Background(), req))

	assert.Equal(t, 1, namespaceCreates(cluster))
	assert.Contains(t, cluster.writes(), "patch namespaces default")
}
