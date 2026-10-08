package apply

import (
	"context"
	"errors"
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
		},
		{
			name: "unreadable record",
			setup: func(c *applyCluster) {
				c.dyn.PrependReactor("get", "moduleinstances", func(k8stesting.Action) (bool, runtime.Object, error) {
					return true, nil, forbidden("moduleinstances", "demo")
				})
			},
			code: opmexit.ExitPermissionDenied,
			want: "apply stopped before any change",
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
				c.dyn.PrependReactor("get", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
					return true, nil, forbidden("configmaps", "app")
				})
			},
			code: opmexit.ExitPermissionDenied,
			want: "apply stopped before any change",
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
	for _, check := range []string{"get customresourcedefinitions", "get moduleinstances", "create selfsubjectaccessreviews", "get configmaps"} {
		at := indexOf(calls, check)
		require.GreaterOrEqual(t, at, 0, "%s ran: %v", check, calls)
		assert.Less(t, at, create, "%s runs before the namespace is created", check)
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

func indexOf(calls []string, call string) int {
	for i, c := range calls {
		if c == call {
			return i
		}
	}
	return -1
}
