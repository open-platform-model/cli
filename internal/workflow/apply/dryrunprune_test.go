package apply

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"

	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/library/opm/module"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	workflowrender "github.com/open-platform-model/cli/internal/workflow/render"
	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"
)

// cliOwnedInstance is a CLI-owned ModuleInstance CR whose recorded inventory
// holds the given ConfigMap names.
func cliOwnedInstance(name, namespace string, configMaps ...string) *unstructured.Unstructured {
	entries := make([]any, 0, len(configMaps))
	for _, cm := range configMaps {
		entries = append(entries, map[string]any{
			"group": "", "kind": "ConfigMap", "namespace": namespace, "name": cm, "v": "v1", "component": "app",
		})
	}
	return cliOwnedInstanceWith(name, namespace, entries...)
}

// protectedEntries are inventory entries for a core Namespace and a CRD, the
// two kinds prune leaves behind.
func protectedEntries() []any {
	return []any{
		map[string]any{"group": "", "kind": "Namespace", "namespace": "", "name": "apps", "v": "v1", "component": "app"},
		map[string]any{"group": "apiextensions.k8s.io", "kind": "CustomResourceDefinition", "namespace": "", "name": "widgets.example.io", "v": "v1", "component": "app"},
	}
}

// cliOwnedInstanceWith is a CLI-owned ModuleInstance CR whose recorded
// inventory holds the given raw entries.
func cliOwnedInstanceWith(name, namespace string, entries ...any) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": inventory.APIVersionModuleInstance,
		"kind":       inventory.KindModuleInstance,
		"metadata":   map[string]any{"name": name, "namespace": namespace},
		"spec":       map[string]any{"owner": inventory.OwnerCLI},
		"status":     map[string]any{"inventory": map[string]any{"revision": int64(1), "entries": entries}},
	}}
}

// renderedConfigMap is a ConfigMap as the instance renders and applies it:
// with the OPM managed-by label, so a prune of it passes the delete verdict.
func renderedConfigMap(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]any{"name": name, "namespace": "default", "labels": map[string]any{"component.opmodel.dev/name": "app", opmlabels.ManagedBy: opmlabels.ManagedByCLI}},
		"data":       map[string]any{"k": "v"},
	}}
}

// A dry-run reads the recorded inventory and reports the resources a real
// apply would prune, and issues no delete and no write to the ModuleInstance.
func TestExecute_DryRunReportsWouldPrune(t *testing.T) {
	ctx := context.Background()
	withReleasedCLIVersion(t)

	var logBuf bytes.Buffer
	output.SetLogWriter(&logBuf)
	t.Cleanup(func() { output.SetLogWriter(os.Stderr) })

	fake := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			inventory.ModuleInstanceGVR: "ModuleInstanceList",
			configMapGVR:                "ConfigMapList",
		},
		cliOwnedInstanceWith("demo", "default", append([]any{
			map[string]any{"group": "", "kind": "ConfigMap", "namespace": "default", "name": "keep", "v": "v1", "component": "app"},
			map[string]any{"group": "", "kind": "ConfigMap", "namespace": "default", "name": "stale", "v": "v1", "component": "app"},
		}, protectedEntries()...)...))
	fake.PrependReactor("patch", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, renderedConfigMap("keep"), nil
	})
	var mutations []string
	fake.PrependReactor("*", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		switch action.GetVerb() {
		case "delete", "create", "update", "delete-collection":
			mutations = append(mutations, action.GetVerb()+" "+action.GetResource().Resource)
		case "patch":
			if action.GetResource().Resource != "configmaps" {
				mutations = append(mutations, "patch "+action.GetResource().Resource)
			}
		}
		return false, nil, nil
	})

	newReq := func(noPrune bool) Request {
		return Request{
			Result: &workflowrender.Result{
				Resources: []*unstructured.Unstructured{renderedConfigMap("keep")},
				Instance:  module.InstanceMetadata{Name: "demo", Namespace: "default", UUID: "uuid-1"},
			},
			K8sClient: &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: fake},
			Log:       output.InstanceLogger("demo"),
			Options:   Options{DryRun: true, NoPrune: noPrune},
		}
	}

	require.NoError(t, Execute(ctx, newReq(false)))
	assert.Contains(t, logBuf.String(), "would prune 1 stale resource(s)")
	assert.Contains(t, logBuf.String(), "ConfigMap/default/stale")
	assert.Contains(t, logBuf.String(), "would leave 2 resource(s) behind")
	assertLeftBehind(t, logBuf.String(), "Namespace/apps", "CustomResourceDefinition/widgets.example.io")
	for _, line := range strings.Split(logBuf.String(), "\n") {
		if strings.Contains(line, "ConfigMap/default/keep") {
			assert.NotContains(t, line, "would prune", "a still-rendered resource is not listed as pruned")
		}
	}
	assert.Empty(t, mutations, "a dry-run deletes and writes nothing besides the dry-run resource patch")

	logBuf.Reset()
	require.NoError(t, Execute(ctx, newReq(true)))
	assert.NotContains(t, logBuf.String(), "would prune", "--no-prune reports nothing to prune")
	assert.NotContains(t, logBuf.String(), "left behind", "--no-prune lists nothing left behind")
}

// assertLeftBehind checks that each resource path appears on a line with the
// left-behind status, and never on a would-prune line.
func assertLeftBehind(t *testing.T, log string, paths ...string) {
	t.Helper()
	for _, path := range paths {
		found := false
		for _, line := range strings.Split(log, "\n") {
			if !strings.Contains(line, path) {
				continue
			}
			assert.NotContains(t, line, "would prune", "%s is never listed as pruned", path)
			if strings.Contains(line, output.StatusLeftBehind) {
				found = true
			}
		}
		assert.True(t, found, "%s is listed as %s in:\n%s", path, output.StatusLeftBehind, log)
	}
}

// A real apply prunes the stale ConfigMap, never deletes the Namespace or the
// CRD the render dropped, and lists both as left behind.
func TestExecute_PruneLeavesProtectedKindsBehind(t *testing.T) {
	ctx := context.Background()
	withReleasedCLIVersion(t)

	var logBuf bytes.Buffer
	output.SetLogWriter(&logBuf)
	t.Cleanup(func() { output.SetLogWriter(os.Stderr) })

	staleCM := renderedConfigMap("stale")
	ns := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "Namespace", "metadata": map[string]any{"name": "apps"},
	}}
	crd := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apiextensions.k8s.io/v1", "kind": "CustomResourceDefinition", "metadata": map[string]any{"name": "widgets.example.io"},
	}}
	fake := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			inventory.ModuleInstanceGVR: "ModuleInstanceList",
			inventory.PlatformGVR:       "PlatformList",
			crdGVR:                      "CustomResourceDefinitionList",
			configMapGVR:                "ConfigMapList",
		},
		makeModuleInstanceCRD(true, true), staleCM, ns, crd,
		cliOwnedInstanceWith("demo", "default", append([]any{
			map[string]any{"group": "", "kind": "ConfigMap", "namespace": "default", "name": "keep", "v": "v1", "component": "app"},
			map[string]any{"group": "", "kind": "ConfigMap", "namespace": "default", "name": "stale", "v": "v1", "component": "app"},
		}, protectedEntries()...)...))
	fake.PrependReactor("patch", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, renderedConfigMap("keep"), nil
	})
	fake.PrependReactor("patch", "moduleinstances", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, cliOwnedInstance("demo", "default", "keep"), nil
	})
	var deletes []string
	fake.PrependReactor("delete", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		deletes = append(deletes, action.GetResource().Resource+"/"+action.(k8stesting.DeleteAction).GetName())
		return false, nil, nil
	})

	clientset := k8sfake.NewClientset()
	clientset.PrependReactor("create", "selfsubjectaccessreviews", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, &authorizationv1.SelfSubjectAccessReview{Status: authorizationv1.SubjectAccessReviewStatus{Allowed: true}}, nil
	})

	req := Request{
		Result: &workflowrender.Result{
			Resources: []*unstructured.Unstructured{renderedConfigMap("keep")},
			Instance:  module.InstanceMetadata{Name: "demo", Namespace: "default", UUID: "uuid-1"},
		},
		K8sClient: &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: fake, Clientset: clientset},
		Log:       output.InstanceLogger("demo"),
		Options:   Options{SuccessAppliedMessage: "applied", SuccessUpToDateMessage: "up to date"},
	}
	require.NoError(t, Execute(ctx, req))

	assert.Equal(t, []string{"configmaps/stale"}, deletes, "only the stale ConfigMap is deleted")
	assert.Contains(t, logBuf.String(), "leaving 2 resource(s) behind")
	assertLeftBehind(t, logBuf.String(), "Namespace/apps", "CustomResourceDefinition/widgets.example.io")
}

func TestPreviewPrune_ListsLeftBehind(t *testing.T) {
	var logBuf bytes.Buffer
	output.SetLogWriter(&logBuf)
	t.Cleanup(func() { output.SetLogWriter(os.Stderr) })

	prunable, protected := inventory.SplitProtected([]k8sinventory.Entry{
		{Kind: "Namespace", Name: "ns"},
		{Kind: "Service", Namespace: "default", Name: "svc"},
	})
	previewPrune(prunable, protected, output.InstanceLogger("demo"))
	assert.Contains(t, logBuf.String(), "would prune 1 stale resource(s)")
	assert.Contains(t, logBuf.String(), "Service/default/svc")
	assert.Contains(t, logBuf.String(), "would leave 1 resource(s) behind")
	assertLeftBehind(t, logBuf.String(), "Namespace/ns")

	logBuf.Reset()
	prunable, protected = inventory.SplitProtected([]k8sinventory.Entry{{Kind: "Namespace", Name: "ns"}})
	previewPrune(prunable, protected, output.InstanceLogger("demo"))
	assert.NotContains(t, logBuf.String(), "would prune", "nothing prunable, no would-prune block")
	assertLeftBehind(t, logBuf.String(), "Namespace/ns")

	logBuf.Reset()
	previewPrune(nil, nil, output.InstanceLogger("demo"))
	assert.Empty(t, logBuf.String(), "an empty stale set reports nothing")
}
