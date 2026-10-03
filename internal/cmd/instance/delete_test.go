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
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
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

// --force skips the confirmation prompt; it must not reach the readiness
// guard, since forcing past that guard produces the wedge rather than avoiding
// it. deleteOperatorOwned takes no force parameter at all — this pins the flag
// to its stated meaning so a later change cannot quietly widen it.
func TestDeleteForceFlagIsConfirmationOnly(t *testing.T) {
	cmd := NewInstanceDeleteCmd(&config.GlobalConfig{})
	forceFlag := cmd.Flags().Lookup("force")
	require.NotNil(t, forceFlag)
	assert.Contains(t, forceFlag.Usage, "confirmation")
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
