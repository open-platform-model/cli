package apply

import (
	"context"
	"sync"
	"testing"
	"time"

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

	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	workflowrender "github.com/open-platform-model/cli/internal/workflow/render"
)

func TestFormatDryRunSummary(t *testing.T) {
	tests := []struct {
		name   string
		result kubernetes.ApplyResult
		want   string
	}{
		{"nothing skipped", kubernetes.ApplyResult{Applied: 3}, "dry run complete: 3 resources would be applied"},
		{"skipped custom resources", kubernetes.ApplyResult{Applied: 2, Skipped: 1}, "dry run complete: 2 resources would be applied, 1 skipped (CustomResourceDefinition created by this apply)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FormatDryRunSummary(&tt.result))
		})
	}
}

// fooCRD is a CustomResourceDefinition a module ships; it carries no status,
// so it never reports Established.
func fooCRD() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apiextensions.k8s.io/v1",
		"kind":       "CustomResourceDefinition",
		"metadata":   map[string]any{"name": "foos.example.com"},
		"spec": map[string]any{
			"group": "example.com",
			"names": map[string]any{"kind": "Foo", "plural": "foos"},
			"scope": "Cluster",
		},
	}}
}

// A CustomResourceDefinition that never becomes established within --timeout
// fails the command before the second stage, and the apply neither prunes the
// stale entry of the recorded inventory nor writes the ModuleInstance.
func TestExecute_CRDNotEstablishedSkipsPruneAndInventoryWrite(t *testing.T) {
	ctx := context.Background()
	withReleasedCLIVersion(t)
	prev := kubernetes.WaitPollInterval
	kubernetes.WaitPollInterval = 5 * time.Millisecond
	t.Cleanup(func() { kubernetes.WaitPollInterval = prev })

	fake := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			inventory.ModuleInstanceGVR: "ModuleInstanceList",
			inventory.PlatformGVR:       "PlatformList",
			crdGVR:                      "CustomResourceDefinitionList",
			configMapGVR:                "ConfigMapList",
		},
		makeModuleInstanceCRD(true, true),
		cliOwnedInstance("demo", "default", "fresh", "stale"))

	var mu sync.Mutex
	var mutations, patched []string
	fake.PrependReactor("patch", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		mu.Lock()
		defer mu.Unlock()
		res := action.GetResource().Resource
		switch res {
		case "customresourcedefinitions":
			patched = append(patched, res)
			crd := fooCRD()
			if err := fake.Tracker().Add(crd); err != nil {
				return true, nil, err
			}
			return true, crd, nil
		case "configmaps":
			patched = append(patched, res)
			return true, renderedConfigMap("fresh"), nil
		}
		mutations = append(mutations, "patch "+res)
		return false, nil, nil
	})
	fake.PrependReactor("*", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		switch action.GetVerb() {
		case "delete", "create", "update", "delete-collection":
			mu.Lock()
			mutations = append(mutations, action.GetVerb()+" "+action.GetResource().Resource)
			mu.Unlock()
		}
		return false, nil, nil
	})

	cs := k8sfake.NewClientset()
	cs.PrependReactor("create", "selfsubjectaccessreviews", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, &authorizationv1.SelfSubjectAccessReview{
			Status: authorizationv1.SubjectAccessReviewStatus{Allowed: true},
		}, nil
	})

	req := Request{
		Result: &workflowrender.Result{
			Resources: []*unstructured.Unstructured{renderedConfigMap("fresh"), fooCRD()},
			Instance:  module.InstanceMetadata{Name: "demo", Namespace: "default", UUID: "uuid-1"},
		},
		K8sClient: &kubernetes.Client{Dynamic: fake, Clientset: cs},
		Log:       output.InstanceLogger("demo"),
		Options:   Options{Timeout: 100 * time.Millisecond, SuccessAppliedMessage: "applied", SuccessUpToDateMessage: "up to date"},
	}

	err := Execute(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CustomResourceDefinition/foos.example.com")
	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, opmexit.ExitGeneralError, exitErr.Code)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"customresourcedefinitions"}, patched, "the second stage (the ConfigMap) is never applied")
	assert.Empty(t, mutations, "no prune delete and no ModuleInstance write")
}
