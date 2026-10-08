package operator

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/version"
)

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
	prev := kubernetes.WaitPollInterval
	kubernetes.WaitPollInterval = 5 * time.Millisecond
	t.Cleanup(func() { kubernetes.WaitPollInterval = prev })
}

func deleteLater(t *testing.T, client *kubernetes.Client, obj *unstructured.Unstructured, after time.Duration) {
	t.Helper()
	go func() {
		time.Sleep(after)
		err := client.ResourceClient(kubernetes.GVRFromUnstructured(obj), obj.GetNamespace()).Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
		assert.NoError(t, err)
	}()
}

// releasedCLI makes the ldflags-set CLI version a released semver for the
// test, so the instance apply's gates behave as in a release.
func releasedCLI(t *testing.T) {
	t.Helper()
	orig := version.Version
	version.Version = "v1.0.0-beta.9"
	t.Cleanup(func() { version.Version = orig })
}

// install plans and installs a render on the fake cluster.
func install(t *testing.T, fc *fakeCluster, r *fakeRender, opts PlanOptions) (*InstallResult, error) {
	t.Helper()
	if opts.Timeout == 0 {
		opts.Timeout = 2 * time.Second
	}
	env := newEnv(fc, r)
	plan, err := PlanInstall(context.Background(), env, testResolution("v0.1.0"), defaultTarget, opts)
	require.NoError(t, err)
	return Install(context.Background(), env, plan)
}

var (
	deploymentGVR  = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	clusterRoleGVR = schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles"}
	namespaceGVR   = schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}
)

func (fc *fakeCluster) mustGet(gvr schema.GroupVersionResource, ns, name string) *unstructured.Unstructured {
	fc.t.Helper()
	obj, err := fc.client.ResourceClient(gvr, ns).Get(context.Background(), name, metav1.GetOptions{})
	require.NoError(fc.t, err, "%s %s/%s", gvr.Resource, ns, name)
	return obj
}

func (fc *fakeCluster) exists(gvr schema.GroupVersionResource, ns, name string) bool {
	_, err := fc.client.ResourceClient(gvr, ns).Get(context.Background(), name, metav1.GetOptions{})
	return err == nil
}

func (fc *fakeCluster) record() *inventory.Record {
	fc.t.Helper()
	rec, err := inventory.GetRecord(context.Background(), fc.client, OperatorInstanceName, OperatorNamespace)
	require.NoError(fc.t, err)
	return rec
}

func entryNames(rec *inventory.Record) []string {
	names := make([]string, 0, len(rec.Inventory.Entries))
	for _, e := range rec.Inventory.Entries {
		names = append(names, e.Kind+"/"+e.Name)
	}
	return names
}

// "Fresh cluster" and "The module's Namespace is not refused".
func TestInstall_FreshCluster(t *testing.T) {
	releasedCLI(t)
	fastPolling(t)
	fc := newFakeCluster(t)

	var logBuf bytes.Buffer
	output.SetLogWriter(&logBuf)
	t.Cleanup(func() { output.SetLogWriter(os.Stderr) })

	result, err := install(t, fc, &fakeRender{objs: moduleObjects(renderOpts{})}, PlanOptions{})
	require.NoError(t, err)
	assert.Equal(t, 4, result.CRDs)
	assert.True(t, result.Recorded)
	assert.NotContains(t, logBuf.String(), "already exist and are managed by OPM",
		"install applies the CRDs itself first: a fresh install is not warned about them")

	writes := fc.Writes()
	require.GreaterOrEqual(t, len(writes), 4)
	assert.Equal(t, []string{
		"patch customresourcedefinitions", "patch customresourcedefinitions",
		"patch customresourcedefinitions", "patch customresourcedefinitions",
	}, writes[:4], "the CRDs are applied before any other object")
	assert.NotContains(t, writes, "create namespaces", "install never creates the Namespace outside the render")

	rec := fc.record()
	require.NotNil(t, rec)
	assert.Equal(t, inventory.OwnerCLI, rec.Owner)
	names := entryNames(rec)
	assert.Len(t, names, 8)
	for _, crd := range CRDNames() {
		assert.Contains(t, names, "CustomResourceDefinition/"+crd)
	}
	assert.Contains(t, names, "Namespace/"+OperatorNamespace)
	assert.Contains(t, names, "Deployment/"+ControllerDeploymentName)
	assert.True(t, fc.exists(deploymentGVR, OperatorNamespace, ControllerDeploymentName))
}

// "Newer running operator does not block repair": the instance apply skips
// the running-operator ceiling, so a Platform reporting an operator above
// the CLI does not refuse install.
func TestInstall_NewerRunningOperatorDoesNotBlockRepair(t *testing.T) {
	releasedCLI(t)
	fastPolling(t)
	platform := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": inventory.APIVersionModuleInstance,
		"kind":       inventory.KindPlatform,
		"metadata":   map[string]any{"name": inventory.PlatformSingletonName},
		"spec":       map[string]any{"type": "kubernetes"},
		"status":     map[string]any{"operatorVersion": "1.1.0"},
	}}
	fc := newFakeCluster(t, platform)

	result, err := install(t, fc, &fakeRender{objs: moduleObjects(renderOpts{})}, PlanOptions{})
	require.NoError(t, err)
	assert.True(t, result.Recorded)
	assert.NotNil(t, fc.record())
}

// "Unchanged reinstall": every object keeps its uid and resourceVersion.
func TestInstall_UnchangedReinstall(t *testing.T) {
	releasedCLI(t)
	fastPolling(t)
	fc := newFakeCluster(t)
	r := &fakeRender{objs: moduleObjects(renderOpts{})}
	_, err := install(t, fc, r, PlanOptions{})
	require.NoError(t, err)

	type ident struct{ uid, rv string }
	before := map[string]ident{}
	for _, obj := range r.objs {
		live := fc.mustGet(kubernetes.GVRFromUnstructured(obj), obj.GetNamespace(), obj.GetName())
		before[obj.GetKind()+"/"+obj.GetName()] = ident{string(live.GetUID()), live.GetResourceVersion()}
	}

	_, err = install(t, fc, r, PlanOptions{})
	require.NoError(t, err)
	for _, obj := range r.objs {
		live := fc.mustGet(kubernetes.GVRFromUnstructured(obj), obj.GetNamespace(), obj.GetName())
		assert.Equal(t, before[obj.GetKind()+"/"+obj.GetName()], ident{string(live.GetUID()), live.GetResourceVersion()},
			"%s/%s changed on an unchanged reinstall", obj.GetKind(), obj.GetName())
	}
}

// "Upgrade drops an object the new version no longer renders".
func TestInstall_UpgradePrunesWhatTheNewVersionDropped(t *testing.T) {
	releasedCLI(t)
	fastPolling(t)
	fc := newFakeCluster(t)
	_, err := install(t, fc, &fakeRender{objs: moduleObjects(renderOpts{extraRole: "opm-operator-retired-role"})}, PlanOptions{})
	require.NoError(t, err)
	require.True(t, fc.exists(clusterRoleGVR, "", "opm-operator-retired-role"))

	_, err = install(t, fc, &fakeRender{objs: moduleObjects(renderOpts{moduleVer: "0.2.0"})}, PlanOptions{})
	require.NoError(t, err)

	assert.False(t, fc.exists(clusterRoleGVR, "", "opm-operator-retired-role"), "the dropped ClusterRole is pruned")
	assert.True(t, fc.exists(namespaceGVR, "", OperatorNamespace))
	for _, crd := range CRDNames() {
		assert.True(t, fc.exists(crdGVR, "", crd))
	}
	assert.NotContains(t, entryNames(fc.record()), "ClusterRole/opm-operator-retired-role")
}

// "Platform and role are not recorded".
func TestInstall_UserRoleIsNotRecorded(t *testing.T) {
	releasedCLI(t)
	fastPolling(t)
	fc := newFakeCluster(t)
	rbac := RBACOptions{Enabled: true, User: "alice"}

	result, err := install(t, fc, &fakeRender{objs: moduleObjects(renderOpts{})}, PlanOptions{Extra: rbac.Objects()})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Extra)
	assert.True(t, fc.exists(clusterRoleGVR, "", "opm-cli-user"))
	for _, name := range entryNames(fc.record()) {
		assert.NotContains(t, name, "opm-cli-user")
		assert.NotContains(t, name, "Platform/")
	}
}

// "Rollout does not complete": nothing is rolled back.
func TestInstall_RolloutTimeoutKeepsEverything(t *testing.T) {
	releasedCLI(t)
	fastPolling(t)
	fc := newFakeCluster(t)
	fc.notReady = true

	_, err := install(t, fc, &fakeRender{objs: moduleObjects(renderOpts{})}, PlanOptions{Timeout: 300 * time.Millisecond})
	require.Error(t, err)
	var rolloutErr *RolloutError
	require.ErrorAs(t, err, &rolloutErr)
	assert.Contains(t, err.Error(), ControllerDeploymentName)
	assert.Contains(t, err.Error(), "timed out after")
	assert.Contains(t, err.Error(), "re-running 'opm operator install' completes it")
	assert.True(t, fc.exists(deploymentGVR, OperatorNamespace, ControllerDeploymentName))
	assert.NotNil(t, fc.record(), "the record remains")
}

// "Solo-cluster CRD install" and "Full install after CRDs-only".
func TestInstall_CRDsOnlyThenFullInstall(t *testing.T) {
	releasedCLI(t)
	fastPolling(t)
	fc := newFakeCluster(t)
	r := &fakeRender{objs: moduleObjects(renderOpts{})}

	result, err := install(t, fc, r, PlanOptions{CRDsOnly: true})
	require.NoError(t, err)
	assert.Equal(t, 4, result.CRDs)
	assert.False(t, result.Recorded)
	uids := map[string]string{}
	for _, crd := range CRDNames() {
		live := fc.mustGet(crdGVR, "", crd)
		uids[crd] = string(live.GetUID())
	}
	assert.Nil(t, fc.record(), "no ModuleInstance")
	assert.False(t, fc.exists(deploymentGVR, OperatorNamespace, ControllerDeploymentName), "no Deployment")
	assert.False(t, fc.exists(namespaceGVR, "", OperatorNamespace), "no Namespace")

	_, err = install(t, fc, r, PlanOptions{})
	require.NoError(t, err)
	for _, crd := range CRDNames() {
		assert.Equal(t, uids[crd], string(fc.mustGet(crdGVR, "", crd).GetUID()), "%s kept its uid", crd)
	}
	names := entryNames(fc.record())
	for _, crd := range CRDNames() {
		assert.Contains(t, names, "CustomResourceDefinition/"+crd)
	}
}

// --rbac applies on the CRDs-only path as on the full one.
func TestInstall_CRDsOnlyWithRBAC(t *testing.T) {
	fastPolling(t)
	fc := newFakeCluster(t)
	rbac := RBACOptions{Enabled: true, User: "alice"}

	result, err := install(t, fc, &fakeRender{objs: moduleObjects(renderOpts{})}, PlanOptions{CRDsOnly: true, Extra: rbac.Objects()})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Extra)
	assert.True(t, fc.exists(clusterRoleGVR, "", "opm-cli-user"))
}
