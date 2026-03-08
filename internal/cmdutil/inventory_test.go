package cmdutil_test

import (
	"context"
	"errors"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/kubernetes"
	oerrors "github.com/opmodel/cli/pkg/errors"
)

// makeTestK8sClient creates a fake kubernetes.Client for unit tests.
func makeTestK8sClient(objects ...runtime.Object) *kubernetes.Client {
	return &kubernetes.Client{
		Clientset: fake.NewClientset(objects...),
	}
}

// makeTestInventorySecret builds a minimal InventorySecret and serializes it
// to a Kubernetes Secret for pre-seeding the fake client.
func makeTestInventorySecret(t *testing.T, releaseName, namespace, releaseID string) *corev1.Secret {
	t.Helper()
	inv := &inventory.InventorySecret{
		ReleaseMetadata: inventory.ReleaseMetadata{
			Kind:               "ModuleRelease",
			APIVersion:         "core.opmodel.dev/v1alpha1",
			ReleaseName:        releaseName,
			ReleaseNamespace:   namespace,
			ReleaseID:          releaseID,
			LastTransitionTime: "2026-01-01T00:00:00Z",
		},
		ModuleMetadata: inventory.ModuleMetadata{
			Kind:       "Module",
			APIVersion: "core.opmodel.dev/v1alpha1",
			Name:       releaseName,
		},
		Index:   []string{},
		Changes: map[string]*inventory.ChangeEntry{},
	}
	secret, err := inventory.MarshalToSecret(inv)
	require.NoError(t, err)
	secret.Namespace = namespace
	return secret
}

// silentLogger returns a logger that discards all output (used to suppress log noise in tests).
func silentLogger() *log.Logger {
	return log.New(nil)
}

// TestResolveInventory_ByReleaseName_Success verifies that a release is found
// when ReleaseName is set and the inventory Secret exists.
func TestResolveInventory_ByReleaseName_Success(t *testing.T) {
	secret := makeTestInventorySecret(t, "myapp", "default", "uuid-abc-123")
	client := makeTestK8sClient(secret)
	ctx := context.Background()

	rsf := &cmdutil.ReleaseSelectorFlags{ReleaseName: "myapp", Namespace: "default"}
	inv, live, missing, err := cmdutil.ResolveInventory(ctx, client, rsf, "default", silentLogger())

	require.NoError(t, err)
	require.NotNil(t, inv)
	assert.Equal(t, "myapp", inv.ReleaseMetadata.ReleaseName)
	assert.Equal(t, "uuid-abc-123", inv.ReleaseMetadata.ReleaseID)
	assert.Empty(t, live)    // no resources tracked in inventory
	assert.Empty(t, missing) // no missing entries either
}

// TestResolveInventory_ByReleaseID_Success verifies that a release is found
// when ReleaseID is set and the inventory Secret exists.
func TestResolveInventory_ByReleaseID_Success(t *testing.T) {
	secret := makeTestInventorySecret(t, "myapp", "production", "uuid-xyz-789")
	client := makeTestK8sClient(secret)
	ctx := context.Background()

	rsf := &cmdutil.ReleaseSelectorFlags{ReleaseName: "myapp", ReleaseID: "uuid-xyz-789", Namespace: "production"}
	inv, live, missing, err := cmdutil.ResolveInventory(ctx, client, rsf, "production", silentLogger())

	require.NoError(t, err)
	require.NotNil(t, inv)
	assert.Equal(t, "uuid-xyz-789", inv.ReleaseMetadata.ReleaseID)
	assert.Empty(t, live)
	assert.Empty(t, missing)
}

// TestResolveInventory_ByReleaseID_NoReleaseName verifies that when only ReleaseID is set
// (no ReleaseName), the release ID is used as the display name internally.
func TestResolveInventory_ByReleaseID_NoReleaseName(t *testing.T) {
	secret := makeTestInventorySecret(t, "myapp", "default", "uuid-nnn-000")
	client := makeTestK8sClient(secret)
	ctx := context.Background()

	// Only ReleaseID set — no ReleaseName
	rsf := &cmdutil.ReleaseSelectorFlags{ReleaseID: "uuid-nnn-000", Namespace: "default"}
	inv, _, _, err := cmdutil.ResolveInventory(ctx, client, rsf, "default", silentLogger())

	require.NoError(t, err)
	require.NotNil(t, inv)
	assert.Equal(t, "uuid-nnn-000", inv.ReleaseMetadata.ReleaseID)
}

// TestResolveInventory_NotFound verifies that when the inventory
// Secret does not exist, an ExitNotFound error is returned.
func TestResolveInventory_NotFound(t *testing.T) {
	client := makeTestK8sClient() // empty — no secrets
	ctx := context.Background()

	rsf := &cmdutil.ReleaseSelectorFlags{ReleaseName: "nonexistent", Namespace: "default"}
	inv, live, missing, err := cmdutil.ResolveInventory(ctx, client, rsf, "default", silentLogger())

	require.Error(t, err)
	assert.Nil(t, inv)
	assert.Nil(t, live)
	assert.Nil(t, missing)

	var exitErr *oerrors.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, oerrors.ExitNotFound, exitErr.Code)
}

// TestResolveInventory_K8sError_LookupFails verifies that a Kubernetes API error
// during inventory lookup returns an ExitGeneralError.
func TestResolveInventory_K8sError_LookupFails(t *testing.T) {
	// Inject a Secret with malformed JSON data to cause an unmarshal error
	// (simulates a Kubernetes error path after a successful GET).
	brokenSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      inventory.SecretName("brokenapp", "uuid-broken"),
			Namespace: "default",
			Labels: map[string]string{
				"opmodel.dev/component":           "inventory",
				"module-release.opmodel.dev/name": "brokenapp",
			},
		},
		Data: map[string][]byte{
			"inventory": []byte("not-valid-json"),
		},
	}
	client := makeTestK8sClient(brokenSecret)
	ctx := context.Background()

	rsf := &cmdutil.ReleaseSelectorFlags{ReleaseName: "brokenapp", Namespace: "default"}
	inv, live, missing, err := cmdutil.ResolveInventory(ctx, client, rsf, "default", silentLogger())

	require.Error(t, err)
	assert.Nil(t, inv)
	assert.Nil(t, live)
	assert.Nil(t, missing)

	var exitErr *oerrors.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, oerrors.ExitGeneralError, exitErr.Code)
}
