package apply

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/library/opm/module"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	workflowrender "github.com/open-platform-model/cli/internal/workflow/render"
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
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": inventory.APIVersionModuleInstance,
		"kind":       inventory.KindModuleInstance,
		"metadata":   map[string]any{"name": name, "namespace": namespace},
		"spec":       map[string]any{"owner": inventory.OwnerCLI},
		"status":     map[string]any{"inventory": map[string]any{"revision": int64(1), "entries": entries}},
	}}
}

func renderedConfigMap(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]any{"name": name, "namespace": "default", "labels": map[string]any{"component.opmodel.dev/name": "app"}},
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
		cliOwnedInstance("demo", "default", "keep", "stale"))
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
			K8sClient: &kubernetes.Client{Dynamic: fake},
			Log:       output.InstanceLogger("demo"),
			Options:   Options{DryRun: true, NoPrune: noPrune},
		}
	}

	require.NoError(t, Execute(ctx, newReq(false)))
	assert.Contains(t, logBuf.String(), "would prune 1 stale resource(s)")
	assert.Contains(t, logBuf.String(), "ConfigMap/default/stale")
	for _, line := range strings.Split(logBuf.String(), "\n") {
		if strings.Contains(line, "ConfigMap/default/keep") {
			assert.NotContains(t, line, "would prune", "a still-rendered resource is not listed as pruned")
		}
	}
	assert.Empty(t, mutations, "a dry-run deletes and writes nothing besides the dry-run resource patch")

	logBuf.Reset()
	require.NoError(t, Execute(ctx, newReq(true)))
	assert.NotContains(t, logBuf.String(), "would prune", "--no-prune reports nothing to prune")
}

func TestPreviewPrune_SkipsNamespaces(t *testing.T) {
	var logBuf bytes.Buffer
	output.SetLogWriter(&logBuf)
	t.Cleanup(func() { output.SetLogWriter(os.Stderr) })

	previewPrune([]inventory.InventoryEntry{
		{Kind: "Namespace", Name: "ns"},
		{Kind: "Service", Namespace: "default", Name: "svc"},
	}, output.InstanceLogger("demo"))
	assert.Contains(t, logBuf.String(), "would prune 1 stale resource(s)")
	assert.Contains(t, logBuf.String(), "Service/default/svc")
	assert.NotContains(t, logBuf.String(), "Namespace/ns")

	logBuf.Reset()
	previewPrune([]inventory.InventoryEntry{{Kind: "Namespace", Name: "ns"}}, output.InstanceLogger("demo"))
	assert.Empty(t, logBuf.String(), "nothing prunable, nothing reported")
}
