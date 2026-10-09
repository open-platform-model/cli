package controller

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/open-platform-model/cli/internal/controller/crds"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/version"
	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"
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
		resource, err := client.ResourceClientFor(context.Background(), obj.GroupVersionKind(), obj.GetNamespace())
		if assert.NoError(t, err) {
			assert.NoError(t, resource.Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{}))
		}
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
	rec, err := inventory.GetRecord(context.Background(), fc.client, ControllerInstanceName, ControllerNamespace)
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
	assert.Contains(t, names, "Namespace/"+ControllerNamespace)
	assert.Contains(t, names, "Deployment/"+ControllerDeploymentName)
	assert.True(t, fc.exists(deploymentGVR, ControllerNamespace, ControllerDeploymentName))
}

// "Newer running controller does not block repair": the instance apply skips
// the running-controller ceiling, so a Platform reporting a controller above
// the CLI does not refuse install.
func TestInstall_NewerRunningControllerDoesNotBlockRepair(t *testing.T) {
	releasedCLI(t)
	fastPolling(t)
	platform := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": inventory.APIVersionModuleInstance,
		"kind":       inventory.KindPlatform,
		"metadata":   map[string]any{"name": inventory.PlatformSingletonName},
		"spec":       map[string]any{"type": "kubernetes"},
		"status":     map[string]any{"controllerVersion": "1.1.0"},
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
		live := fc.mustGet(kubetest.GVR(obj), obj.GetNamespace(), obj.GetName())
		before[obj.GetKind()+"/"+obj.GetName()] = ident{string(live.GetUID()), live.GetResourceVersion()}
	}

	_, err = install(t, fc, r, PlanOptions{})
	require.NoError(t, err)
	for _, obj := range r.objs {
		live := fc.mustGet(kubetest.GVR(obj), obj.GetNamespace(), obj.GetName())
		assert.Equal(t, before[obj.GetKind()+"/"+obj.GetName()], ident{string(live.GetUID()), live.GetResourceVersion()},
			"%s/%s changed on an unchanged reinstall", obj.GetKind(), obj.GetName())
	}
}

// "Upgrade drops an object the new version no longer renders".
func TestInstall_UpgradePrunesWhatTheNewVersionDropped(t *testing.T) {
	releasedCLI(t)
	fastPolling(t)
	fc := newFakeCluster(t)
	_, err := install(t, fc, &fakeRender{objs: moduleObjects(renderOpts{extraRole: "opm-controller-retired-role"})}, PlanOptions{})
	require.NoError(t, err)
	require.True(t, fc.exists(clusterRoleGVR, "", "opm-controller-retired-role"))

	_, err = install(t, fc, &fakeRender{objs: moduleObjects(renderOpts{moduleVer: "0.2.0"})}, PlanOptions{})
	require.NoError(t, err)

	assert.False(t, fc.exists(clusterRoleGVR, "", "opm-controller-retired-role"), "the dropped ClusterRole is pruned")
	assert.True(t, fc.exists(namespaceGVR, "", ControllerNamespace))
	for _, crd := range CRDNames() {
		assert.True(t, fc.exists(crdGVR, "", crd))
	}
	assert.NotContains(t, entryNames(fc.record()), "ClusterRole/opm-controller-retired-role")
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
	assert.Contains(t, err.Error(), "re-running 'opm controller install' completes it")
	assert.True(t, fc.exists(deploymentGVR, ControllerNamespace, ControllerDeploymentName))
	assert.NotNil(t, fc.record(), "the record remains")
}

// embeddedForTestInstance makes the embedded CRDs carry the instance UUID the
// fixture render stamps, as they do in a released cli: the pin renders them
// for the same instance a full install renders, so both carry one UUID, and
// the apply guard judges the objects of either as the instance's own.
func embeddedForTestInstance(t *testing.T) {
	t.Helper()
	prev := embeddedCRDs
	embeddedCRDs = func() ([]*unstructured.Unstructured, error) {
		objs, err := prev()
		if err != nil {
			return nil, err
		}
		for _, obj := range objs {
			labels := obj.GetLabels()
			labels[opmlabels.ModuleInstanceUUID] = testInstanceUUID
			obj.SetLabels(labels)
		}
		return objs, nil
	}
	t.Cleanup(func() { embeddedCRDs = prev })
}

// installCRDsOnly plans and installs the embedded CRDs on the fake cluster.
func installCRDsOnly(t *testing.T, fc *fakeCluster, r *fakeRender, opts PlanOptions) (*InstallResult, error) {
	t.Helper()
	if opts.Timeout == 0 {
		opts.Timeout = 2 * time.Second
	}
	env := newEnv(fc, r)
	plan, err := PlanCRDsOnly(context.Background(), env, opts)
	if err != nil {
		return nil, err
	}
	return Install(context.Background(), env, plan)
}

// "Solo-cluster CRD install with no registry" and "The controller's instance
// record is not read": the embedded CRDs are applied as opm-cli and served,
// nothing renders, and no ModuleInstance is read or written, even over a
// record a full install would refuse.
func TestInstall_CRDsOnlyAppliesTheEmbeddedCRDsOffline(t *testing.T) {
	fastPolling(t)
	fc := newFakeCluster(t, controllerRecord("controller", nil))
	r := &fakeRender{err: errors.New("the CRDs-only install must not render")}

	result, err := installCRDsOnly(t, fc, r, PlanOptions{})
	require.NoError(t, err)
	assert.Equal(t, 4, result.CRDs)
	assert.False(t, result.Recorded)
	assert.Equal(t, PinnedTarget(), result.Target)
	assert.Zero(t, r.calls, "no render")

	embedded, err := crds.Objects()
	require.NoError(t, err)
	for _, want := range embedded {
		live := fc.mustGet(crdGVR, "", want.GetName())
		assert.Equal(t, want.GetLabels(), live.GetLabels(), "%s carries the pinned render's labels", want.GetName())
		assert.Equal(t, want.Object["spec"], live.Object["spec"], "%s is the embedded CRD", want.GetName())
		assert.Equal(t, "opm-cli", fc.Manager("customresourcedefinitions", want.GetName()))
	}
	for _, read := range fc.Reads() {
		assert.NotContains(t, read, "moduleinstances", "no ModuleInstance is read")
	}
	for _, w := range fc.Writes() {
		assert.Equal(t, "patch customresourcedefinitions", w)
	}
	assert.Equal(t, "controller", fc.record().Owner, "the record is untouched")
}

// "CRDs OPM does not manage are refused, not adopted": a CRD without OPM's
// managed-by label is an exit-2 guard error with the shared guard's message,
// which names the adopt annotation and the UUID of the controller's instance;
// nothing is applied.
func TestPlanCRDsOnly_RefusesCRDsOPMDoesNotManage(t *testing.T) {
	foreign := renderedCRDObject(inventory.CRDNameModuleInstances, true)
	fc := newFakeCluster(t, foreign)
	embedded, err := crds.Objects()
	require.NoError(t, err)

	_, err = installCRDsOnly(t, fc, &fakeRender{}, PlanOptions{})
	var guardErr *GuardError
	require.ErrorAs(t, err, &guardErr)
	assert.Contains(t, err.Error(), "refusing to install: 1 object(s) cannot be applied by this instance:")
	assert.Contains(t, err.Error(), "CustomResourceDefinition/"+inventory.CRDNameModuleInstances+" exists and is not managed by OPM")
	assert.Contains(t, err.Error(), opmlabels.AnnotationAdopt+"="+embeddedInstanceUUID(embedded))
	assert.NotContains(t, err.Error(), "start fresh")
	assert.NotContains(t, err.Error(), "remove or rename")
	assert.Empty(t, fc.Writes())
}

// "CRDs a newer module version installed are not replaced": module version
// 0.2.0 installed the CRDs (a full install, or --crds-only from a CLI that
// pins it), so --crds-only at the 0.1.0 pin refuses with exit 2, names both
// remedies and writes nothing.
func TestPlanCRDsOnly_RefusesCRDsOfANewerModuleVersion(t *testing.T) {
	newer := labeled(renderedCRDObject(inventory.CRDNameModuleInstances, true), "crds", "0.2.0")
	fc := newFakeCluster(t, newer)

	_, err := installCRDsOnly(t, fc, &fakeRender{}, PlanOptions{})
	var targetErr *TargetError
	require.ErrorAs(t, err, &targetErr)
	assert.Contains(t, err.Error(), inventory.CRDNameModuleInstances)
	assert.Contains(t, err.Error(), "module version 0.2.0")
	assert.Contains(t, err.Error(), "run --crds-only from a CLI that pins module version 0.2.0 or newer")
	assert.Contains(t, err.Error(), "opm controller install --version 0.2.0")
	assert.Empty(t, fc.Writes())
}

// A newer version of another module is not compared with the pin: the CRD
// goes on to the apply guard, which accepts a CRD that carries OPM's
// managed-by label and the instance's UUID, and is applied over.
func TestInstall_CRDsOnlyIgnoresAnotherModulesVersion(t *testing.T) {
	fastPolling(t)
	embeddedForTestInstance(t)
	other := labeled(renderedCRDObject(inventory.CRDNameModuleInstances, true), "crds", "9.0.0")
	labels := other.GetLabels()
	labels[moduleNameLabel] = "another_module"
	other.SetLabels(labels)
	fc := newFakeCluster(t, other)

	result, err := installCRDsOnly(t, fc, &fakeRender{}, PlanOptions{})
	require.NoError(t, err)
	assert.Equal(t, 4, result.CRDs)
}

// CRDs an older or the same module version installed are applied over.
func TestInstall_CRDsOnlyOverCRDsOfAnOlderModuleVersion(t *testing.T) {
	fastPolling(t)
	embeddedForTestInstance(t)
	older := labeled(renderedCRDObject(inventory.CRDNameModuleInstances, true), "crds", "0.0.9")
	fc := newFakeCluster(t, older)

	result, err := installCRDsOnly(t, fc, &fakeRender{}, PlanOptions{})
	require.NoError(t, err)
	assert.Equal(t, 4, result.CRDs)
	assert.Equal(t, PinnedModuleVersion,
		fc.mustGet(crdGVR, "", inventory.CRDNameModuleInstances).GetLabels()[moduleVersionLabel])
}

// "Full install after CRDs-only".
func TestInstall_CRDsOnlyThenFullInstall(t *testing.T) {
	releasedCLI(t)
	fastPolling(t)
	embeddedForTestInstance(t)
	fc := newFakeCluster(t)
	r := &fakeRender{objs: moduleObjects(renderOpts{})}

	result, err := installCRDsOnly(t, fc, r, PlanOptions{})
	require.NoError(t, err)
	assert.Equal(t, 4, result.CRDs)
	assert.False(t, result.Recorded)
	uids := map[string]string{}
	for _, crd := range CRDNames() {
		live := fc.mustGet(crdGVR, "", crd)
		uids[crd] = string(live.GetUID())
	}
	assert.Nil(t, fc.record(), "no ModuleInstance")
	assert.False(t, fc.exists(deploymentGVR, ControllerNamespace, ControllerDeploymentName), "no Deployment")
	assert.False(t, fc.exists(namespaceGVR, "", ControllerNamespace), "no Namespace")

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

	result, err := installCRDsOnly(t, fc, &fakeRender{}, PlanOptions{Extra: rbac.Objects()})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Extra)
	assert.True(t, fc.exists(clusterRoleGVR, "", "opm-cli-user"))
}
