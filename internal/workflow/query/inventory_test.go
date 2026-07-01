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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func makeTestK8sClient(objects ...runtime.Object) *kubernetes.Client {
	return &kubernetes.Client{Clientset: fake.NewClientset(objects...)}
}

func makeTestInventorySecret(t *testing.T, instanceName, namespace, instanceID string) *corev1.Secret {
	t.Helper()
	inv := &inventory.InstanceInventoryRecord{InstanceMetadata: inventory.InstanceMetadata{Kind: "ModuleInstance", APIVersion: inventory.APIVersionV1Alpha1, InstanceName: instanceName, InstanceNamespace: namespace, InstanceID: instanceID, LastTransitionTime: "2026-01-01T00:00:00Z"}, ModuleMetadata: inventory.ModuleMetadata{Kind: "Module", APIVersion: inventory.APIVersionV1Alpha1, Name: instanceName}, Inventory: inventory.Inventory{Entries: []inventory.InventoryEntry{}}}
	secret, err := inventory.MarshalToSecret(inv)
	require.NoError(t, err)
	secret.Namespace = namespace
	return secret
}

func silentLogger() *log.Logger { return log.New(nil) }

func TestResolveInventory_ByInstanceName_Success(t *testing.T) {
	secret := makeTestInventorySecret(t, "myapp", "default", "uuid-abc-123")
	client := makeTestK8sClient(secret)
	ctx := context.Background()
	rsf := &cmdutil.InstanceSelectorFlags{InstanceName: "myapp", Namespace: "default"}
	inv, live, missing, err := ResolveInventory(ctx, client, rsf, "default", silentLogger())
	require.NoError(t, err)
	require.NotNil(t, inv)
	assert.Equal(t, "myapp", inv.InstanceMetadata.InstanceName)
	assert.Equal(t, "uuid-abc-123", inv.InstanceMetadata.InstanceID)
	assert.Empty(t, live)
	assert.Empty(t, missing)
}

func TestResolveInventory_ByInstanceID_Success(t *testing.T) {
	secret := makeTestInventorySecret(t, "myapp", "production", "uuid-xyz-789")
	client := makeTestK8sClient(secret)
	ctx := context.Background()
	rsf := &cmdutil.InstanceSelectorFlags{InstanceName: "myapp", InstanceID: "uuid-xyz-789", Namespace: "production"}
	inv, live, missing, err := ResolveInventory(ctx, client, rsf, "production", silentLogger())
	require.NoError(t, err)
	require.NotNil(t, inv)
	assert.Equal(t, "uuid-xyz-789", inv.InstanceMetadata.InstanceID)
	assert.Empty(t, live)
	assert.Empty(t, missing)
}

func TestResolveInventory_ByInstanceID_NoInstanceName(t *testing.T) {
	secret := makeTestInventorySecret(t, "myapp", "default", "uuid-nnn-000")
	client := makeTestK8sClient(secret)
	ctx := context.Background()
	rsf := &cmdutil.InstanceSelectorFlags{InstanceID: "uuid-nnn-000", Namespace: "default"}
	inv, _, _, err := ResolveInventory(ctx, client, rsf, "default", silentLogger())
	require.NoError(t, err)
	require.NotNil(t, inv)
	assert.Equal(t, "uuid-nnn-000", inv.InstanceMetadata.InstanceID)
}

func TestResolveInventory_NotFound(t *testing.T) {
	client := makeTestK8sClient()
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

func TestResolveInventory_K8sError_LookupFails(t *testing.T) {
	brokenSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: inventory.SecretName("brokenapp", "uuid-broken"), Namespace: "default", Labels: map[string]string{"opmodel.dev/component": "inventory", "module-instance.opmodel.dev/name": "brokenapp"}}, Data: map[string][]byte{"inventory": []byte("not-valid-json")}}
	client := makeTestK8sClient(brokenSecret)
	ctx := context.Background()
	rsf := &cmdutil.InstanceSelectorFlags{InstanceName: "brokenapp", Namespace: "default"}
	inv, live, missing, err := ResolveInventory(ctx, client, rsf, "default", silentLogger())
	require.Error(t, err)
	assert.Nil(t, inv)
	assert.Nil(t, live)
	assert.Nil(t, missing)
	var exitErr *opmexit.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, opmexit.ExitGeneralError, exitErr.Code)
}
