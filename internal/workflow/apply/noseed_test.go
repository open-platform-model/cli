package apply

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"

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
	"github.com/open-platform-model/cli/internal/platform"
	workflowrender "github.com/open-platform-model/cli/internal/workflow/render"
)

// TestExecute_NeverSeedsAPlatform covers "Apply never seeds a Platform": an
// instance apply whose render fell back from a cluster without a Platform to
// the instance's own deps applies its resources and then touches the
// platforms resource with nothing but the ceiling gate's read, and says
// nothing about seeding. This is the path the removed write-if-absent seed
// ran on: a successful, non-dry-run apply after a cluster fallback.
func TestExecute_NeverSeedsAPlatform(t *testing.T) {
	ctx := context.Background()
	withReleasedCLIVersion(t)

	var logBuf bytes.Buffer
	output.SetLogWriter(&logBuf)
	t.Cleanup(func() { output.SetLogWriter(os.Stderr) })

	fake := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			inventory.ModuleInstanceGVR: "ModuleInstanceList",
			inventory.PlatformGVR:       "PlatformList",
			crdGVR:                      "CustomResourceDefinitionList",
			configMapGVR:                "ConfigMapList",
		},
		makeModuleInstanceCRD(true, true)) // no Platform in the cluster

	// The plain tracker cannot serve server-side apply: answer the resource
	// patch with the object itself so the apply succeeds.
	fake.PrependReactor("patch", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, configMap(), nil
	})
	var platformActions []string
	fake.PrependReactor("*", "platforms", func(action k8stesting.Action) (bool, runtime.Object, error) {
		platformActions = append(platformActions, action.GetVerb())
		return false, nil, nil // passthrough to the tracker
	})

	client := &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: fake}

	req := Request{
		Result: &workflowrender.Result{
			Resources: []*unstructured.Unstructured{configMap()},
			Instance:  module.InstanceMetadata{Name: "demo", Namespace: "default"},
			Platform: platform.Resolution{
				Source:   platform.SourceModuleDeps,
				DepsKind: platform.DepsInstance,
				Warning:  "cluster Platform not used (no Platform CR in the cluster)",
			},
		},
		K8sClient: client,
		Log:       output.InstanceLogger("demo"),
		Options:   Options{SuccessAppliedMessage: "applied", SuccessUpToDateMessage: "up to date"},
	}
	require.NoError(t, Execute(ctx, req))

	assert.Contains(t, logBuf.String(), "applied 1 resources successfully", "the apply ran to completion")
	require.Contains(t, platformActions, "get", "the recorder sees the ceiling gate's Platform read")
	for _, verb := range platformActions {
		assert.Equal(t, "get", verb, "the only platforms action is the ceiling gate's read")
	}
	for _, verb := range []string{"create", "update", "patch"} {
		assert.NotContains(t, platformActions, verb)
	}
	assert.False(t, strings.Contains(strings.ToLower(logBuf.String()), "seed"), "log: %s", logBuf.String())
}

// configMapGVR is the resource the apply test renders.
var configMapGVR = schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}

// configMap is the one rendered object the apply test applies.
func configMap() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]any{"name": "demo", "namespace": "default"},
		"data":       map[string]any{"k": "v"},
	}}
}
