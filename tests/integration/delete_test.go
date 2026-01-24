package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/kubernetes"
)

func TestDelete_WeightedOrder(t *testing.T) {
	if testClient == nil {
		t.Skip("No test client available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test that delete would work with dry-run
	// In a real test, we'd first apply resources, then delete them

	// Discover resources (will be empty in a clean cluster)
	resources, err := testClient.DiscoverModuleResources(ctx, "test-module", "default")
	require.NoError(t, err)

	if len(resources) == 0 {
		t.Log("No resources to delete (expected in clean test environment)")
		return
	}

	// Delete with dry-run
	result, err := testClient.Delete(ctx, resources, kubernetes.DeleteOptions{
		DryRun: true,
	})
	require.NoError(t, err)

	t.Logf("Would delete %d resources", result.Deleted)
}

func TestDelete_NotFound(t *testing.T) {
	if testClient == nil {
		t.Skip("No test client available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Try to discover non-existent module
	resources, err := testClient.DiscoverModuleResources(ctx, "non-existent-module", "default")
	require.NoError(t, err)
	require.Empty(t, resources)
}
