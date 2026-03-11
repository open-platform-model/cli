package apply

import (
	"context"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/internal/runtime/modulerelease"
	workflowrender "github.com/opmodel/cli/internal/workflow/render"
	pkginventory "github.com/opmodel/cli/pkg/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCurrentInventoryEntries(t *testing.T) {
	resources := []*unstructured.Unstructured{{Object: map[string]any{"apiVersion": "v1", "kind": "ConfigMap", "metadata": map[string]any{"name": "demo", "namespace": "apps"}}}}
	entries := CurrentInventoryEntries(resources)
	require.Len(t, entries, 1)
	assert.Equal(t, "ConfigMap", entries[0].Kind)
	assert.Equal(t, "demo", entries[0].Name)
	assert.Equal(t, "apps", entries[0].Namespace)
}

func TestPreviousInventoryEntries(t *testing.T) {
	prevInventory := &pkginventory.InventorySecret{Index: []string{"change-1"}, Changes: map[string]*pkginventory.ChangeEntry{"change-1": {Inventory: pkginventory.InventoryList{Entries: []pkginventory.InventoryEntry{{Kind: "Service", Name: "web"}}}}}}
	entries := PreviousInventoryEntries(prevInventory)
	require.Len(t, entries, 1)
	assert.Equal(t, "Service", entries[0].Kind)
	assert.Equal(t, "web", entries[0].Name)
}

func TestGuardEmptyRender(t *testing.T) {
	releaseLog := output.ReleaseLogger("test")
	err := GuardEmptyRender(0, []pkginventory.InventoryEntry{{Kind: "ConfigMap"}}, false, releaseLog)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render produced 0 resources")
}

func TestFormatApplySummary(t *testing.T) {
	summary := FormatApplySummary(&kubernetes.ApplyResult{Applied: 5, Created: 2, Configured: 1, Unchanged: 2})
	assert.Equal(t, "applied 5 resources successfully (2 created, 1 configured, 2 unchanged)", summary)
}

func TestExecute_BlocksControllerManagedRelease(t *testing.T) {
	ctx := context.Background()
	inv := &inventory.InventorySecret{
		ReleaseMetadata: inventory.ReleaseMetadata{
			Kind:             "ModuleRelease",
			APIVersion:       "core.opmodel.dev/v1alpha1",
			ReleaseName:      "demo",
			ReleaseNamespace: "apps",
			ReleaseID:        "uuid-1",
			CreatedBy:        inventory.CreatedByController,
		},
		ModuleMetadata: inventory.ModuleMetadata{Kind: "Module", APIVersion: "core.opmodel.dev/v1alpha1", Name: "demo-module"},
		Index:          []string{},
		Changes:        map[string]*inventory.ChangeEntry{},
	}
	secret, err := inventory.MarshalToSecret(inv)
	require.NoError(t, err)
	secret.ResourceVersion = "1"

	client := &kubernetes.Client{Clientset: fake.NewClientset(secret)}
	req := Request{
		Result: &workflowrender.Result{
			Release: modulerelease.ReleaseMetadata{Name: "demo", Namespace: "apps", UUID: "uuid-1"},
		},
		K8sClient: client,
		Log:       log.New(nil),
		Options:   Options{},
	}

	err = Execute(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "controller-managed")
}
