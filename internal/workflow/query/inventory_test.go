package query

import (
	"context"
	"errors"
	"testing"

	opmexit "github.com/opmodel/cli/internal/exit"

	"github.com/charmbracelet/log"
	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
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

func makeTestInventorySecret(t *testing.T, releaseName, namespace, releaseID string) *corev1.Secret {
	t.Helper()
	inv := &inventory.ReleaseInventoryRecord{ReleaseMetadata: inventory.ReleaseMetadata{Kind: "ModuleRelease", APIVersion: "core.opmodel.dev/v1alpha1", ReleaseName: releaseName, ReleaseNamespace: namespace, ReleaseID: releaseID, LastTransitionTime: "2026-01-01T00:00:00Z"}, ModuleMetadata: inventory.ModuleMetadata{Kind: "Module", APIVersion: "core.opmodel.dev/v1alpha1", Name: releaseName}, Inventory: inventory.Inventory{Entries: []inventory.InventoryEntry{}}}
	secret, err := inventory.MarshalToSecret(inv)
	require.NoError(t, err)
	secret.Namespace = namespace
	return secret
}

func silentLogger() *log.Logger { return log.New(nil) }

func TestResolveInventory_ByReleaseName_Success(t *testing.T) {
	secret := makeTestInventorySecret(t, "myapp", "default", "uuid-abc-123")
	client := makeTestK8sClient(secret)
	ctx := context.Background()
	rsf := &cmdutil.ReleaseSelectorFlags{ReleaseName: "myapp", Namespace: "default"}
	inv, live, missing, err := ResolveInventory(ctx, client, rsf, "default", silentLogger())
	require.NoError(t, err)
	require.NotNil(t, inv)
	assert.Equal(t, "myapp", inv.ReleaseMetadata.ReleaseName)
	assert.Equal(t, "uuid-abc-123", inv.ReleaseMetadata.ReleaseID)
	assert.Empty(t, live)
	assert.Empty(t, missing)
}

func TestResolveInventory_ByReleaseID_Success(t *testing.T) {
	secret := makeTestInventorySecret(t, "myapp", "production", "uuid-xyz-789")
	client := makeTestK8sClient(secret)
	ctx := context.Background()
	rsf := &cmdutil.ReleaseSelectorFlags{ReleaseName: "myapp", ReleaseID: "uuid-xyz-789", Namespace: "production"}
	inv, live, missing, err := ResolveInventory(ctx, client, rsf, "production", silentLogger())
	require.NoError(t, err)
	require.NotNil(t, inv)
	assert.Equal(t, "uuid-xyz-789", inv.ReleaseMetadata.ReleaseID)
	assert.Empty(t, live)
	assert.Empty(t, missing)
}

func TestResolveInventory_ByReleaseID_NoReleaseName(t *testing.T) {
	secret := makeTestInventorySecret(t, "myapp", "default", "uuid-nnn-000")
	client := makeTestK8sClient(secret)
	ctx := context.Background()
	rsf := &cmdutil.ReleaseSelectorFlags{ReleaseID: "uuid-nnn-000", Namespace: "default"}
	inv, _, _, err := ResolveInventory(ctx, client, rsf, "default", silentLogger())
	require.NoError(t, err)
	require.NotNil(t, inv)
	assert.Equal(t, "uuid-nnn-000", inv.ReleaseMetadata.ReleaseID)
}

func TestResolveInventory_NotFound(t *testing.T) {
	client := makeTestK8sClient()
	ctx := context.Background()
	rsf := &cmdutil.ReleaseSelectorFlags{ReleaseName: "nonexistent", Namespace: "default"}
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
	brokenSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: inventory.SecretName("brokenapp", "uuid-broken"), Namespace: "default", Labels: map[string]string{"opmodel.dev/component": "inventory", "module-release.opmodel.dev/name": "brokenapp"}}, Data: map[string][]byte{"inventory": []byte("not-valid-json")}}
	client := makeTestK8sClient(brokenSecret)
	ctx := context.Background()
	rsf := &cmdutil.ReleaseSelectorFlags{ReleaseName: "brokenapp", Namespace: "default"}
	inv, live, missing, err := ResolveInventory(ctx, client, rsf, "default", silentLogger())
	require.Error(t, err)
	assert.Nil(t, inv)
	assert.Nil(t, live)
	assert.Nil(t, missing)
	var exitErr *opmexit.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, opmexit.ExitGeneralError, exitErr.Code)
}
