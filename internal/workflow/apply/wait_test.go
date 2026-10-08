package apply

import (
	"context"
	"testing"
	"time"

	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	workflowrender "github.com/open-platform-model/cli/internal/workflow/render"
	"github.com/open-platform-model/library/opm/module"
)

func waitDeployment(name string, replicas, updated, available int64) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata":   map[string]any{"name": name, "namespace": "demo", "generation": int64(2)},
		"spec":       map[string]any{"replicas": replicas},
		"status": map[string]any{
			"observedGeneration": int64(2),
			"replicas":           replicas,
			"updatedReplicas":    updated,
			"availableReplicas":  available,
		},
	}}
}

func waitClusterRole() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "rbac.authorization.k8s.io/v1",
		"kind":       "ClusterRole",
		"metadata":   map[string]any{"name": "reader"},
	}}
}

// waitRequest builds the apply Request waitForHealthy reads: the rendered
// resources are what get polled, the cluster holds the live copies.
func waitRequest(t *testing.T, rendered, live []*unstructured.Unstructured, timeout time.Duration) Request {
	t.Helper()
	client, _ := recordingDynamicClient(live...)
	return Request{
		Result: &workflowrender.Result{
			Resources: rendered,
			Instance:  module.InstanceMetadata{Name: "podinfo", Namespace: "demo"},
		},
		K8sClient: client,
		Log:       output.InstanceLogger("test"),
		Options:   Options{Wait: true, Timeout: timeout},
	}
}

func TestWaitForHealthy(t *testing.T) {
	tests := []struct {
		name     string
		rendered []*unstructured.Unstructured
		live     []*unstructured.Unstructured
		wantErr  []string // substrings of the error; nil means success
	}{
		{
			name:     "healthy rollout and an applied-only kind",
			rendered: []*unstructured.Unstructured{waitDeployment("web", 2, 2, 2), waitClusterRole()},
			live:     []*unstructured.Unstructured{waitDeployment("web", 2, 2, 2), waitClusterRole()},
		},
		{
			name:     "no resources",
			rendered: nil,
		},
		{
			name:     "stuck upgrade times out and names the Deployment (issue 228)",
			rendered: []*unstructured.Unstructured{waitDeployment("web", 2, 2, 2), waitDeployment("api", 2, 1, 1), waitClusterRole()},
			live:     []*unstructured.Unstructured{waitDeployment("web", 2, 2, 2), waitDeployment("api", 2, 1, 1), waitClusterRole()},
			wantErr:  []string{"not healthy", "timed out", "Deployment/api in demo", "opm instance status podinfo -n demo"},
		},
		{
			name:     "a resource that disappeared fails the wait",
			rendered: []*unstructured.Unstructured{waitDeployment("web", 1, 1, 1)},
			live:     nil,
			wantErr:  []string{"not healthy", "Deployment/web in demo", "disappeared"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := waitRequest(t, tc.rendered, tc.live, 100*time.Millisecond)
			err := waitForHealthy(context.Background(), req, req.Options.Timeout, req.Log)
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			for _, s := range tc.wantErr {
				assert.Contains(t, err.Error(), s)
			}
			var exitErr *opmexit.ExitError
			require.ErrorAs(t, err, &exitErr)
			assert.Equal(t, opmexit.ExitGeneralError, exitErr.Code)
		})
	}
}

// waitExecuteRequest builds an Execute request that applies one ConfigMap
// through a fake cluster. The patch lands the ConfigMap in the tracker only
// when persist is true, so a poll after an apply that left nothing behind
// reads NotFound.
func waitExecuteRequest(persist bool, opts Options) Request {
	listKinds := map[schema.GroupVersionResource]string{
		inventory.ModuleInstanceGVR: "ModuleInstanceList",
		inventory.PlatformGVR:       "PlatformList",
		crdGVR:                      "CustomResourceDefinitionList",
		configMapGVR:                "ConfigMapList",
	}
	fake := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), listKinds, makeModuleInstanceCRD(true, true))
	fake.PrependReactor("patch", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
		cm := configMap()
		if persist {
			if err := fake.Tracker().Add(cm); err != nil {
				return true, nil, err
			}
		}
		return true, cm, nil
	})

	opts.SuccessAppliedMessage, opts.SuccessUpToDateMessage = "applied", "up to date"
	return Request{
		Result: &workflowrender.Result{
			Resources: []*unstructured.Unstructured{configMap()},
			Instance:  module.InstanceMetadata{Name: "demo", Namespace: "default"},
		},
		K8sClient: &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: fake},
		Log:       output.InstanceLogger("demo"),
		Options:   opts,
	}
}

func TestExecute_Wait(t *testing.T) {
	ctx := context.Background()
	withReleasedCLIVersion(t)

	t.Run("waits for the applied resources after a CLI-owned apply", func(t *testing.T) {
		req := waitExecuteRequest(true, Options{Wait: true, Timeout: time.Second})
		require.NoError(t, Execute(ctx, req))
	})

	t.Run("a resource that is gone after apply fails the command", func(t *testing.T) {
		req := waitExecuteRequest(false, Options{Wait: true, Timeout: time.Second})
		err := Execute(ctx, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ConfigMap/demo in default")
	})

	t.Run("without --wait nothing is polled", func(t *testing.T) {
		req := waitExecuteRequest(false, Options{})
		require.NoError(t, Execute(ctx, req))
	})

	t.Run("--dry-run skips the wait", func(t *testing.T) {
		req := waitExecuteRequest(false, Options{Wait: true, DryRun: true, Timeout: time.Second})
		require.NoError(t, Execute(ctx, req))
	})
}
