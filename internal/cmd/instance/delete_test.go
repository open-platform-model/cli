package instance

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/operator"
	"github.com/open-platform-model/cli/internal/output"
	pkgcore "github.com/open-platform-model/cli/pkg/core"
)

func emptyClusterClient() *kubernetes.Client {
	listKinds := map[schema.GroupVersionResource]string{
		inventory.ModuleInstanceGVR: "ModuleInstanceList",
	}
	return &kubernetes.Client{
		Dynamic: fakedynamic.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), listKinds),
	}
}

func operatorOwnedRecord() *inventory.Record {
	return &inventory.Record{
		Name:      "podinfo",
		Namespace: "demo",
		Owner:     inventory.OwnerOperator,
		Inventory: inventory.Inventory{Entries: []inventory.InventoryEntry{{Kind: "Deployment", Name: "podinfo", Namespace: "demo"}}},
	}
}

// Deleting a finalizer-armed ModuleInstance with no controller running does not
// delete anything — it wedges the CR in Terminating with its workloads
// orphaned. So the readiness guard refuses rather than proceeding, and it says
// why.
func TestDeleteOperatorOwned_RefusesWhenOperatorIsNotReady(t *testing.T) {
	err := deleteOperatorOwned(context.Background(), emptyClusterClient(), operatorOwnedRecord(),
		time.Second, false, output.InstanceLogger("test"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not ready")
	assert.Contains(t, err.Error(), inventory.CleanupFinalizer)
	assert.Contains(t, err.Error(), "Terminating")
	assert.Contains(t, err.Error(), "opm operator install")
}

// The guard runs before the dry-run short-circuit: a dry run that reported
// "would delete" against a down operator would be describing an outcome that
// cannot happen.
func TestDeleteOperatorOwned_DryRunStillRequiresAReadyOperator(t *testing.T) {
	err := deleteOperatorOwned(context.Background(), emptyClusterClient(), operatorOwnedRecord(),
		time.Second, true, output.InstanceLogger("test"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not ready")
}

// runningOperatorObjects is an operator applied with kubectl from a release
// manifest: its four CRDs Established and its controller Deployment rolled
// out, at the operator's fixed names, with no ModuleInstance recording it.
func runningOperatorObjects() []runtime.Object {
	var objs []runtime.Object
	for _, name := range operator.CRDNames() {
		objs = append(objs, &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "apiextensions.k8s.io/v1", "kind": "CustomResourceDefinition",
			"metadata": map[string]any{"name": name},
			"status":   map[string]any{"conditions": []any{map[string]any{"type": "Established", "status": "True"}}},
		}})
	}
	return append(objs, &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apps/v1", "kind": "Deployment",
		"metadata": map[string]any{"name": operator.ControllerDeploymentName, "namespace": operator.OperatorNamespace},
		"status": map[string]any{
			"observedGeneration": int64(0), "replicas": int64(1),
			"updatedReplicas": int64(1), "availableReplicas": int64(1),
		},
	}})
}

// fakeClusterClient is a fake cluster holding objs, with the ModuleInstance
// list kind registered so a cluster-wide list works on an empty fixture.
func fakeClusterClient(objs ...runtime.Object) (*kubernetes.Client, *fakedynamic.FakeDynamicClient) {
	fake := fakedynamic.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{inventory.ModuleInstanceGVR: "ModuleInstanceList"}, objs...)
	return &kubernetes.Client{Dynamic: fake}, fake
}

// An operator applied with kubectl has no instance record; the readiness check
// finds it by its fixed names, so an operator-owned delete goes ahead.
func TestDeleteOperatorOwned_KubectlInstalledOperatorIsFound(t *testing.T) {
	rec := operatorOwnedRecord()
	mi := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": inventory.APIVersionModuleInstance, "kind": inventory.KindModuleInstance,
		"metadata": map[string]any{"name": rec.Name, "namespace": rec.Namespace},
	}}
	client, fake := fakeClusterClient(append(runningOperatorObjects(), mi)...)

	var runErr error
	captureOutput(t, func() {
		runErr = deleteOperatorOwned(context.Background(), client, rec, 5*time.Second, false, output.InstanceLogger("test"))
	})
	require.NoError(t, runErr)

	_, err := fake.Tracker().Get(inventory.ModuleInstanceGVR, rec.Namespace, rec.Name)
	assert.True(t, apierrors.IsNotFound(err), "the ModuleInstance is deleted")
}

// --force skips the confirmation prompt; it must not reach the readiness
// guard, since forcing past that guard produces the wedge rather than avoiding
// it. deleteOperatorOwned takes no force parameter at all — this pins the flag
// to its stated meaning so a later change cannot quietly widen it.
func TestDeleteForceFlagIsConfirmationOnly(t *testing.T) {
	cmd := NewInstanceDeleteCmd(&config.GlobalConfig{})
	forceFlag := cmd.Flags().Lookup("force")
	require.NotNil(t, forceFlag)
	assert.Contains(t, forceFlag.Usage, "confirmation")

	// Orphaning armed instances is uninstall's explicit choice; the generic
	// delete offers no way to strip finalizers.
	assert.Nil(t, cmd.Flags().Lookup("remove-finalizers"))
}

// captureOutput runs fn with standard output, standard error and the logger
// redirected, and returns everything they wrote.
func captureOutput(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	origStdout, origStderr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = w, w
	output.SetLogWriter(w)

	done := make(chan []byte)
	go func() {
		b, _ := io.ReadAll(r)
		done <- b
	}()
	defer func() {
		os.Stdout, os.Stderr = origStdout, origStderr
		output.SetLogWriter(origStderr)
	}()

	fn()
	require.NoError(t, w.Close())
	return string(<-done)
}

// A CLI-owned instance whose inventory tracks a ConfigMap, its Namespace and a
// ConfigMap another instance has since claimed: delete removes the first
// ConfigMap and the ModuleInstance, keeps the Namespace and the foreign
// ConfigMap, lists both as left behind and exits 0. The foreign ConfigMap pins
// that the command passes the recorded instance UUID to Delete.
func TestExecuteInstanceDelete_LeavesNamespaceBehind(t *testing.T) {
	const uuid = "uuid-demo"
	labels := map[string]any{pkgcore.LabelManagedBy: pkgcore.LabelManagedByValue, pkgcore.LabelModuleInstanceUUID: uuid}
	cm := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]any{"name": "web", "namespace": "apps", "labels": labels},
	}}
	foreign := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]any{"name": "shared", "namespace": "apps", "labels": map[string]any{
			pkgcore.LabelManagedBy: pkgcore.LabelManagedByValue, pkgcore.LabelModuleInstanceUUID: "uuid-other",
		}},
	}}
	ns := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "Namespace",
		"metadata": map[string]any{"name": "apps", "labels": labels},
	}}
	mi := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": inventory.APIVersionModuleInstance, "kind": inventory.KindModuleInstance,
		"metadata": map[string]any{"name": "demo", "namespace": "apps"},
	}}
	inv := &inventory.Record{Name: "demo", Namespace: "apps", Owner: inventory.OwnerCLI, InstanceUUID: uuid}
	rsf := &cmdutil.InstanceSelectorFlags{InstanceName: "demo"}

	for _, dryRun := range []bool{false, true} {
		t.Run(fmt.Sprintf("dryRun=%v", dryRun), func(t *testing.T) {
			ctx := context.Background()
			fake := fakedynamic.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
				map[schema.GroupVersionResource]string{inventory.ModuleInstanceGVR: "ModuleInstanceList"},
				cm.DeepCopy(), foreign.DeepCopy(), ns.DeepCopy(), mi.DeepCopy())
			client := &kubernetes.Client{Dynamic: fake}

			var runErr error
			out := captureOutput(t, func() {
				runErr = executeInstanceDelete(ctx, client, rsf, "apps", inv,
					[]*unstructured.Unstructured{cm.DeepCopy(), foreign.DeepCopy(), ns.DeepCopy()}, dryRun, output.InstanceLogger("demo"))
			})
			require.NoError(t, runErr, out)

			assert.Contains(t, out, "Namespace/apps")
			assert.Contains(t, out, output.StatusLeftBehind)
			assert.Contains(t, out, kubernetes.ProtectedKindReason)
			assert.Contains(t, out, "ConfigMap/apps/shared")
			assert.Contains(t, out, "owned by another instance")
			assert.NotContains(t, out, "all resources have been deleted")

			_, nsErr := fake.Tracker().Get(schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}, "", "apps")
			assert.NoError(t, nsErr, "the Namespace stays")
			_, foreignErr := fake.Tracker().Get(schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}, "apps", "shared")
			assert.NoError(t, foreignErr, "the ConfigMap owned by another instance stays")
			_, cmErr := fake.Tracker().Get(schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}, "apps", "web")
			_, miErr := fake.Tracker().Get(inventory.ModuleInstanceGVR, "apps", "demo")

			if dryRun {
				assert.Contains(t, out, "dry run complete: 1 resources would be deleted, 2 left behind")
				assert.NoError(t, cmErr, "a dry run deletes nothing")
				assert.NoError(t, miErr, "a dry run keeps the ModuleInstance")
				return
			}
			assert.Contains(t, out, "Instance deleted — 2 resource(s) left behind")
			assert.Contains(t, out, "kubectl delete")
			assert.True(t, apierrors.IsNotFound(cmErr), "the ConfigMap is deleted")
			assert.True(t, apierrors.IsNotFound(miErr), "the ModuleInstance is deleted")
		})
	}
}

// A re-read that fails with anything but NotFound fails that resource: the
// ModuleInstance is kept for a re-run, the command exits non-zero and claims
// no completion. A dry run reports the failure as a check it could not make
// and prints no "dry run complete" line.
func TestExecuteInstanceDelete_ReadErrorKeepsModuleInstance(t *testing.T) {
	labels := map[string]any{pkgcore.LabelManagedBy: pkgcore.LabelManagedByValue}
	cm := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]any{"name": "web", "namespace": "apps", "labels": labels},
	}}
	mi := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": inventory.APIVersionModuleInstance, "kind": inventory.KindModuleInstance,
		"metadata": map[string]any{"name": "demo", "namespace": "apps"},
	}}
	inv := &inventory.Record{Name: "demo", Namespace: "apps", Owner: inventory.OwnerCLI}

	for _, dryRun := range []bool{false, true} {
		t.Run(fmt.Sprintf("dryRun=%v", dryRun), func(t *testing.T) {
			ctx := context.Background()
			dyn := fakedynamic.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
				map[schema.GroupVersionResource]string{inventory.ModuleInstanceGVR: "ModuleInstanceList"},
				cm.DeepCopy(), mi.DeepCopy())
			dyn.PrependReactor("get", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "web", errors.New("denied"))
			})
			client := &kubernetes.Client{Dynamic: dyn}

			var runErr error
			out := captureOutput(t, func() {
				runErr = executeInstanceDelete(ctx, client, &cmdutil.InstanceSelectorFlags{InstanceName: "demo"}, "apps", inv,
					[]*unstructured.Unstructured{cm.DeepCopy()}, dryRun, output.InstanceLogger("demo"))
			})
			require.Error(t, runErr)
			assert.NotContains(t, out, "Instance deleted")
			assert.NotContains(t, out, "all resources have been deleted")
			assert.NotContains(t, out, "dry run complete")
			if dryRun {
				assert.Contains(t, runErr.Error(), "1 resource(s) could not be checked")
			} else {
				assert.Contains(t, runErr.Error(), "1 resource(s) failed to delete")
			}

			_, miErr := dyn.Tracker().Get(inventory.ModuleInstanceGVR, "apps", "demo")
			assert.NoError(t, miErr, "the ModuleInstance is kept")
			_, cmErr := dyn.Tracker().Get(schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}, "apps", "web")
			assert.NoError(t, cmErr, "the ConfigMap is not deleted")
		})
	}
}

// guardScenario is a cluster for the operator-instance guard: the target's
// ModuleInstance and the one ConfigMap its inventory tracks, plus any other
// ModuleInstances.
type guardScenario struct {
	rec    *inventory.Record
	cm     *unstructured.Unstructured
	client *kubernetes.Client
	fake   *fakedynamic.FakeDynamicClient
}

func moduleInstanceObj(namespace, name string, finalizers ...string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": inventory.APIVersionModuleInstance, "kind": inventory.KindModuleInstance,
		"metadata": map[string]any{"name": name, "namespace": namespace},
	}}
	if len(finalizers) > 0 {
		obj.SetFinalizers(finalizers)
	}
	return obj
}

// newGuardScenario builds a CLI-owned record name/namespace with modulePath and
// extra inventory entries; targetArmed puts the cleanup finalizer on the
// target's own ModuleInstance; others are further objects in the cluster.
func newGuardScenario(namespace, name, modulePath string, targetArmed bool, extraEntries []inventory.InventoryEntry, others ...runtime.Object) *guardScenario {
	uuid := "uuid-" + name
	cm := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]any{"name": name + "-config", "namespace": namespace, "labels": map[string]any{
			pkgcore.LabelManagedBy: pkgcore.LabelManagedByValue, pkgcore.LabelModuleInstanceUUID: uuid,
		}},
	}}
	entries := append([]inventory.InventoryEntry{{Kind: "ConfigMap", Name: cm.GetName(), Namespace: namespace}}, extraEntries...)
	rec := &inventory.Record{
		Name: name, Namespace: namespace, Owner: inventory.OwnerCLI, ModulePath: modulePath, InstanceUUID: uuid,
		Inventory: inventory.Inventory{Entries: entries},
	}
	var finalizers []string
	if targetArmed {
		finalizers = []string{inventory.CleanupFinalizer}
	}
	objs := append([]runtime.Object{cm.DeepCopy(), moduleInstanceObj(namespace, name, finalizers...)}, others...)
	client, fake := fakeClusterClient(objs...)
	return &guardScenario{rec: rec, cm: cm, client: client, fake: fake}
}

func (g *guardScenario) run(t *testing.T, dryRun bool) (string, error) {
	t.Helper()
	var runErr error
	out := captureOutput(t, func() {
		runErr = deleteResolvedInstance(context.Background(), g.client, &cmdutil.InstanceSelectorFlags{InstanceName: g.rec.Name}, g.rec.Namespace,
			g.rec, []*unstructured.Unstructured{g.cm.DeepCopy()}, 5*time.Second, dryRun, output.InstanceLogger(g.rec.Name))
	})
	return out, runErr
}

func (g *guardScenario) assertUntouched(t *testing.T) {
	t.Helper()
	_, cmErr := g.fake.Tracker().Get(schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}, g.rec.Namespace, g.cm.GetName())
	assert.NoError(t, cmErr, "the tracked ConfigMap still exists")
	_, miErr := g.fake.Tracker().Get(inventory.ModuleInstanceGVR, g.rec.Namespace, g.rec.Name)
	assert.NoError(t, miErr, "the ModuleInstance still exists")
}

func (g *guardScenario) assertDeleted(t *testing.T) {
	t.Helper()
	_, cmErr := g.fake.Tracker().Get(schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}, g.rec.Namespace, g.cm.GetName())
	assert.True(t, apierrors.IsNotFound(cmErr), "the tracked ConfigMap is deleted")
	_, miErr := g.fake.Tracker().Get(inventory.ModuleInstanceGVR, g.rec.Namespace, g.rec.Name)
	assert.True(t, apierrors.IsNotFound(miErr), "the ModuleInstance is deleted")
}

func requireExitCode(t *testing.T, err error, code int) {
	t.Helper()
	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, code, exitErr.Code)
}

func armedHello() runtime.Object {
	return moduleInstanceObj("default", "hello", inventory.CleanupFinalizer)
}

func TestDeleteResolvedInstance_OperatorInstanceRefusedWhileArmed(t *testing.T) {
	g := newGuardScenario(operator.OperatorNamespace, operator.OperatorInstanceName, "", false, nil, armedHello())

	_, err := g.run(t, false)
	requireExitCode(t, err, opmexit.ExitValidationError)
	assert.Contains(t, err.Error(), "default/hello")
	assert.Contains(t, err.Error(), "opm operator uninstall --remove-finalizers")
	assert.Contains(t, err.Error(), "orphaning")
	g.assertUntouched(t)
}

// A dry run that passed where the real run refuses would mislead.
func TestDeleteResolvedInstance_OperatorInstanceDryRunRefusesToo(t *testing.T) {
	g := newGuardScenario(operator.OperatorNamespace, operator.OperatorInstanceName, "", false, nil, armedHello())

	_, err := g.run(t, true)
	requireExitCode(t, err, opmexit.ExitValidationError)
	assert.Contains(t, err.Error(), "default/hello")
	g.assertUntouched(t)
}

// The operator never reconciles or prunes the instance that deploys it, so the
// operator-owned branch would report a prune that does not happen. The record
// is refused before that branch, even with a ready operator.
func TestDeleteResolvedInstance_OperatorOwnedOperatorInstanceRefused(t *testing.T) {
	g := newGuardScenario(operator.OperatorNamespace, operator.OperatorInstanceName, "", false, nil, runningOperatorObjects()...)
	g.rec.Owner = inventory.OwnerOperator
	g.rec.Prune = true

	out, err := g.run(t, false)
	requireExitCode(t, err, opmexit.ExitValidationError)
	assert.Contains(t, err.Error(), "spec.owner")
	assert.Contains(t, err.Error(), "cli")
	assert.Contains(t, err.Error(), "matched by coordinates")
	assert.NotContains(t, out, "pruned")
	assert.NotContains(t, err.Error(), "pruned")
	g.assertUntouched(t)
}

func TestDeleteResolvedInstance_TargetItselfArmed(t *testing.T) {
	g := newGuardScenario(operator.OperatorNamespace, operator.OperatorInstanceName, "", true, nil)

	_, err := g.run(t, false)
	requireExitCode(t, err, opmexit.ExitValidationError)
	assert.Contains(t, err.Error(), "opm-operator-system/opm-operator (the instance being deleted)")
	g.assertUntouched(t)
}

func TestDeleteResolvedInstance_ModulePathSignal(t *testing.T) {
	g := newGuardScenario("platform", "ops", "opmodel.dev/modules/opm_operator@v0", false, nil, armedHello())

	_, err := g.run(t, false)
	requireExitCode(t, err, opmexit.ExitValidationError)
	assert.Contains(t, err.Error(), "default/hello")
	g.assertUntouched(t)
}

func TestDeleteResolvedInstance_InventoryCRDSignal(t *testing.T) {
	crd := inventory.InventoryEntry{Group: "apiextensions.k8s.io", Kind: "CustomResourceDefinition", Name: "moduleinstances.opmodel.dev"}
	g := newGuardScenario("platform", "crds", "example.com/modules/crds@v0", false, []inventory.InventoryEntry{crd}, armedHello())

	_, err := g.run(t, false)
	requireExitCode(t, err, opmexit.ExitValidationError)
	assert.Contains(t, err.Error(), "default/hello")
	g.assertUntouched(t)
}

func TestDeleteResolvedInstance_LookAlikeIsNotGuarded(t *testing.T) {
	crd := inventory.InventoryEntry{Group: "apiextensions.k8s.io", Kind: "CustomResourceDefinition", Name: "widgets.example.opmodel.dev.io"}
	g := newGuardScenario("default", "dash", "opmodel.dev/modules/opm_operator_dashboard@v0", false, []inventory.InventoryEntry{crd}, armedHello())

	out, err := g.run(t, false)
	require.NoError(t, err, out)
	g.assertDeleted(t)
}

func TestDeleteResolvedInstance_OperatorInstanceNoArmedProceeds(t *testing.T) {
	g := newGuardScenario(operator.OperatorNamespace, operator.OperatorInstanceName, "", false, nil)

	out, err := g.run(t, false)
	require.NoError(t, err, out)
	g.assertDeleted(t)
}

func TestDeleteResolvedInstance_ListFailureFailsClosed(t *testing.T) {
	g := newGuardScenario(operator.OperatorNamespace, operator.OperatorInstanceName, "", false, nil)
	g.fake.PrependReactor("list", "moduleinstances", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Group: "opmodel.dev", Resource: "moduleinstances"}, "", errors.New("denied"))
	})

	_, err := g.run(t, false)
	requireExitCode(t, err, opmexit.ExitPermissionDenied)
	g.assertUntouched(t)
}

func TestDeleteResolvedInstance_OtherInstancesAreNotGuarded(t *testing.T) {
	g := newGuardScenario("default", "hello", "example.com/modules/hello@v0", false, nil,
		moduleInstanceObj("demo", "x", inventory.CleanupFinalizer))

	out, err := g.run(t, false)
	require.NoError(t, err, out)
	g.assertDeleted(t)
}

// The operator-owned refusal comes before the cluster-wide list, so a user who
// may not list ModuleInstances still gets exit 2 and the spec.owner remedy, in
// a dry run too, and no list is attempted.
func TestDeleteResolvedInstance_OperatorOwnedRefusedBeforeList(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		t.Run(fmt.Sprintf("dryRun=%v", dryRun), func(t *testing.T) {
			g := newGuardScenario(operator.OperatorNamespace, operator.OperatorInstanceName, "", false, nil)
			g.rec.Owner = inventory.OwnerOperator
			g.fake.PrependReactor("list", "moduleinstances", func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, apierrors.NewForbidden(schema.GroupResource{Group: "opmodel.dev", Resource: "moduleinstances"}, "", errors.New("denied"))
			})

			_, err := g.run(t, dryRun)
			requireExitCode(t, err, opmexit.ExitValidationError)
			assert.Contains(t, err.Error(), "spec.owner")
			for _, a := range g.fake.Actions() {
				assert.False(t, a.GetVerb() == "list" && a.GetResource().Resource == "moduleinstances",
					"no ModuleInstance list before the operator-owned refusal")
			}
			g.assertUntouched(t)
		})
	}
}
