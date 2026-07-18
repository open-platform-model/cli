package query

import (
	"context"
	"errors"
	"testing"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/charmbracelet/log"
	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func makeCRClient(objs ...*unstructured.Unstructured) *kubernetes.Client {
	scheme := runtime.NewScheme()
	runtimeObjs := make([]runtime.Object, len(objs))
	for i, o := range objs {
		runtimeObjs[i] = o
	}
	listKinds := map[schema.GroupVersionResource]string{
		inventory.ModuleInstanceGVR: "ModuleInstanceList",
	}
	return &kubernetes.Client{
		Dynamic: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds, runtimeObjs...),
	}
}

// makeModuleInstanceCR builds a ModuleInstance CR with the CLI's spec/status
// subset for tests.
func makeModuleInstanceCR(name, namespace, instanceID string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": inventory.APIVersionModuleInstance,
		"kind":       inventory.KindModuleInstance,
		"metadata": map[string]any{
			"name":      name,
			"namespace": namespace,
		},
		"spec": map[string]any{
			"owner": inventory.OwnerCLI,
			"module": map[string]any{
				"path":    "opmodel.dev/modules/myapp@v0",
				"version": "0.1.0",
			},
		},
		"status": map[string]any{
			"instanceUUID": instanceID,
			"inventory": map[string]any{
				"revision": int64(1),
				"count":    int64(0),
				"entries":  []any{},
			},
		},
	}}
}

func silentLogger() *log.Logger { return log.New(nil) }

func TestResolveInventory_ByInstanceName_Success(t *testing.T) {
	cr := makeModuleInstanceCR("myapp", "default", "uuid-abc-123")
	client := makeCRClient(cr)
	ctx := context.Background()
	rsf := &cmdutil.InstanceSelectorFlags{InstanceName: "myapp", Namespace: "default"}
	inv, live, missing, err := ResolveInventory(ctx, client, rsf, "default", silentLogger())
	require.NoError(t, err)
	require.NotNil(t, inv)
	assert.Equal(t, "myapp", inv.Name)
	assert.Equal(t, "uuid-abc-123", inv.InstanceUUID)
	assert.Empty(t, live)
	assert.Empty(t, missing)
}

func TestResolveInventory_ByInstanceID_Success(t *testing.T) {
	cr := makeModuleInstanceCR("myapp", "production", "uuid-xyz-789")
	client := makeCRClient(cr)
	ctx := context.Background()
	rsf := &cmdutil.InstanceSelectorFlags{InstanceName: "myapp", InstanceID: "uuid-xyz-789", Namespace: "production"}
	inv, live, missing, err := ResolveInventory(ctx, client, rsf, "production", silentLogger())
	require.NoError(t, err)
	require.NotNil(t, inv)
	assert.Equal(t, "uuid-xyz-789", inv.InstanceUUID)
	assert.Empty(t, live)
	assert.Empty(t, missing)
}

func TestResolveInventory_ByInstanceID_NoInstanceName(t *testing.T) {
	cr := makeModuleInstanceCR("myapp", "default", "uuid-nnn-000")
	client := makeCRClient(cr)
	ctx := context.Background()
	rsf := &cmdutil.InstanceSelectorFlags{InstanceID: "uuid-nnn-000", Namespace: "default"}
	inv, _, _, err := ResolveInventory(ctx, client, rsf, "default", silentLogger())
	require.NoError(t, err)
	require.NotNil(t, inv)
	assert.Equal(t, "uuid-nnn-000", inv.InstanceUUID)
}

func TestResolveInventory_NotFound(t *testing.T) {
	client := makeCRClient()
	ctx := context.Background()
	rsf := &cmdutil.InstanceSelectorFlags{InstanceName: "nonexistent", Namespace: "default"}
	inv, live, missing, err := ResolveInventory(ctx, client, rsf, "default", silentLogger())
	require.Error(t, err)
	assert.Nil(t, inv)
	assert.Nil(t, live)
	assert.Nil(t, missing)
	var exitErr *opmexit.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, opmexit.ExitNotFound, exitErr.Code)
}
