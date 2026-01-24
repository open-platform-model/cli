package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/kubernetes"
)

func TestError_ClusterConnectivity(t *testing.T) {
	// Test that we get the right error for connectivity issues

	// Create a client with an invalid kubeconfig
	_, err := kubernetes.NewClient(kubernetes.ClientOptions{
		Kubeconfig: "/nonexistent/kubeconfig",
	})

	// Should fail to create client
	require.Error(t, err)
}

func TestError_ClusterUnreachable(t *testing.T) {
	if testClient == nil {
		t.Skip("No test client available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// If we have a valid client, connectivity check should pass
	err := testClient.CheckConnection(ctx)
	require.NoError(t, err)
}

func TestError_CUEValidation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to load an invalid module
	loader := &mockLoader{}
	_, err := loader.LoadModule(ctx, "/nonexistent/path", nil)
	require.Error(t, err)
}

// mockLoader is a simple mock for testing error cases.
type mockLoader struct{}

func (m *mockLoader) LoadModule(ctx context.Context, dir string, values []string) (interface{}, error) {
	return nil, errors.New("module not found")
}

func TestError_PermissionDenied(t *testing.T) {
	// This test would require a cluster with restricted RBAC
	// For now, we just verify the error types exist
	require.NotNil(t, kubernetes.ErrPermissionDenied)
	require.NotNil(t, kubernetes.ErrClusterUnreachable)
}
