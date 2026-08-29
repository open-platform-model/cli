package operator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/cli/internal/kubernetes"
)

func TestResolveManifest_EmptyVersionUsesEmbedded(t *testing.T) {
	objs, version, source, err := resolveManifest(context.Background(), "")
	require.NoError(t, err)

	assert.Equal(t, PinnedOperatorVersion, version)
	assert.Equal(t, "embedded", source)
	assert.Len(t, objs, 17)
}

func TestResolveManifest_VersionFetchesInstead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("apiVersion: v1\nkind: Namespace\nmetadata:\n  name: fetched-ns\n"))
	}))
	defer server.Close()

	objs, version, source, err := resolveManifestFrom(context.Background(), server.URL, "v1.0.0-alpha.3")
	require.NoError(t, err)

	assert.Equal(t, "v1.0.0-alpha.3", version)
	assert.Equal(t, "fetched", source)
	require.Len(t, objs, 1)
	assert.Equal(t, "fetched-ns", objs[0].GetName())
}

func TestResolveManifest_FetchErrorPropagates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	objs, version, source, err := resolveManifestFrom(context.Background(), server.URL, "v9.9.9")
	require.Error(t, err)
	assert.Empty(t, objs)
	assert.Empty(t, version)
	assert.Empty(t, source)
	assert.ErrorContains(t, err, "v9.9.9")
}

// terminatingFixture returns obj with a deletionTimestamp set, as the
// apiserver leaves a foreground-deleted object until its dependents are gone.
func terminatingFixture(obj *unstructured.Unstructured) *unstructured.Unstructured {
	obj = obj.DeepCopy()
	now := metav1.Now()
	obj.SetDeletionTimestamp(&now)
	obj.SetFinalizers([]string{"foregroundDeletion"})
	return obj
}

func fastPolling(t *testing.T) {
	t.Helper()
	prev := waitPollInterval
	waitPollInterval = 5 * time.Millisecond
	t.Cleanup(func() { waitPollInterval = prev })
}

// stubApply makes the fake dynamic client accept server-side-apply patches
// (which its tracker cannot do for unstructured objects) by storing the
// applied body, marked ready, and counting each apply.
func stubApply(t *testing.T, client *kubernetes.Client) *int {
	t.Helper()
	fake, ok := client.Dynamic.(*fakedynamic.FakeDynamicClient)
	require.True(t, ok)
	applied := 0
	fake.PrependReactor("patch", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		patch, ok := action.(k8stesting.PatchAction)
		require.True(t, ok)
		obj := &unstructured.Unstructured{}
		require.NoError(t, json.Unmarshal(patch.GetPatch(), obj))
		switch obj.GetKind() {
		case kindCustomResourceDefinition:
			_ = unstructured.SetNestedSlice(obj.Object, []any{
				map[string]any{"type": "Established", "status": "True"},
			}, "status", "conditions")
		case kindDeployment:
			_ = unstructured.SetNestedSlice(obj.Object, []any{
				map[string]any{"type": "Available", "status": "True"},
			}, "status", "conditions")
		}
		gvr := kubernetes.GVRFromUnstructured(obj)
		if err := fake.Tracker().Add(obj); err != nil {
			require.NoError(t, fake.Tracker().Update(gvr, obj, obj.GetNamespace()))
		}
		applied++
		return true, obj, nil
	})
	return &applied
}

func deleteLater(t *testing.T, client *kubernetes.Client, obj *unstructured.Unstructured, after time.Duration) {
	t.Helper()
	go func() {
		time.Sleep(after)
		err := client.ResourceClient(kubernetes.GVRFromUnstructured(obj), obj.GetNamespace()).Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
		assert.NoError(t, err)
	}()
}

func TestInstall_TerminatingObjectDelaysApplyUntilGone(t *testing.T) {
	fastPolling(t)
	doomed := terminatingFixture(deploymentFixture(false))
	client := fakeClientWith(doomed)
	applied := stubApply(t, client)
	deleteLater(t, client, doomed, 30*time.Millisecond)

	result, err := Install(context.Background(), client, InstallOptions{Timeout: 2 * time.Second})
	require.NoError(t, err)
	assert.Equal(t, 17, result.Applied)
	assert.Equal(t, 17, *applied)
}

func TestInstall_TerminatingObjectBeyondBudgetFailsWithNothingApplied(t *testing.T) {
	fastPolling(t)
	doomed := terminatingFixture(deploymentFixture(false))
	client := fakeClientWith(doomed)
	applied := stubApply(t, client)

	result, err := Install(context.Background(), client, InstallOptions{Timeout: 30 * time.Millisecond})
	require.Error(t, err)
	assert.ErrorContains(t, err, "timed out after")
	assert.ErrorContains(t, err, "Deployment/opm-operator-controller-manager in opm-operator-system to finish terminating")
	assert.Equal(t, 0, result.Applied)
	assert.Equal(t, 0, *applied, "nothing may be applied while a planned object is terminating")
}

func TestInstall_AbsentAndLiveObjectsDoNotDelayApply(t *testing.T) {
	fastPolling(t)
	// The Deployment exists and is live; everything else is absent.
	client := fakeClientWith(deploymentFixture(true))
	applied := stubApply(t, client)

	start := time.Now()
	result, err := Install(context.Background(), client, InstallOptions{Timeout: 2 * time.Second})
	require.NoError(t, err)
	assert.Equal(t, 17, result.Applied)
	assert.Equal(t, 17, *applied)
	assert.Less(t, time.Since(start), time.Second)
}

func TestInstall_CRDsOnlyPlanIsGuarded(t *testing.T) {
	fastPolling(t)
	doomed := terminatingFixture(crdFixture(false))
	client := fakeClientWith(doomed)
	applied := stubApply(t, client)

	result, err := Install(context.Background(), client, InstallOptions{CRDsOnly: true, Timeout: 30 * time.Millisecond})
	require.Error(t, err)
	assert.ErrorContains(t, err, "CustomResourceDefinition/moduleinstances.opmodel.dev to finish terminating")
	assert.Equal(t, 0, result.Applied)
	assert.Equal(t, 0, *applied)

	deleteLater(t, client, doomed, 30*time.Millisecond)
	result, err = Install(context.Background(), client, InstallOptions{CRDsOnly: true, Timeout: 2 * time.Second})
	require.NoError(t, err)
	assert.Equal(t, 3, result.Applied)
}

func TestInstall_RBACPlanIsGuarded(t *testing.T) {
	fastPolling(t)
	doomed := terminatingFixture(clusterRole())
	client := fakeClientWith(doomed)
	applied := stubApply(t, client)
	rbac := RBACOptions{Enabled: true, User: "alice"}

	result, err := Install(context.Background(), client, InstallOptions{CRDsOnly: true, RBAC: rbac, Timeout: 30 * time.Millisecond})
	require.Error(t, err)
	assert.ErrorContains(t, err, "ClusterRole/opm-cli-user to finish terminating")
	assert.Equal(t, 0, result.Applied)
	assert.Equal(t, 0, *applied)

	deleteLater(t, client, doomed, 30*time.Millisecond)
	result, err = Install(context.Background(), client, InstallOptions{CRDsOnly: true, RBAC: rbac, Timeout: 2 * time.Second})
	require.NoError(t, err)
	assert.Equal(t, 5, result.Applied, "3 CRDs + ClusterRole + ClusterRoleBinding")
}

func TestInstall_AppliedObjectDisappearingFailsFast(t *testing.T) {
	fastPolling(t)
	client := fakeClientWith()
	fake, ok := client.Dynamic.(*fakedynamic.FakeDynamicClient)
	require.True(t, ok)
	// Apply "succeeds" but stores nothing: every readiness read is NotFound.
	fake.PrependReactor("patch", "*", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, crdFixture(false), nil
	})

	start := time.Now()
	_, err := Install(context.Background(), client, InstallOptions{CRDsOnly: true, Timeout: 5 * time.Second})
	require.Error(t, err)
	assert.ErrorContains(t, err, "was applied and has since disappeared")
	assert.Less(t, time.Since(start), time.Second)
}
