//go:build integration

// Package kubernetes integration tests require envtest or a real cluster.
// Run with: go test -tags integration ./internal/kubernetes/ -v
package kubernetes

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/build"
)

// TestApplyDeleteRoundTrip tests the full apply-discover-delete cycle.
// Requires: KUBECONFIG set to a valid cluster (or envtest).
func TestApplyDeleteRoundTrip(t *testing.T) {
	ctx := context.Background()

	client, err := NewClient(ClientOptions{})
	require.NoError(t, err, "need a valid kubeconfig for integration tests")

	moduleName := "integration-test"
	namespace := "default"

	// Create a simple ConfigMap resource
	cm := &unstructured.Unstructured{}
	cm.SetAPIVersion("v1")
	cm.SetKind("ConfigMap")
	cm.SetName("opm-integration-test")
	cm.SetNamespace(namespace)
	cm.Object["data"] = map[string]interface{}{
		"key": "value",
	}

	resources := []*build.Resource{
		{
			Object:    cm,
			Component: "test-component",
		},
	}

	meta := build.ModuleReleaseMetadata{
		Name:      moduleName,
		Namespace: namespace,
		Version:   "0.1.0",
	}

	// Apply
	applyResult, err := Apply(ctx, client, resources, meta, ApplyOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, applyResult.Applied)
	assert.Empty(t, applyResult.Errors)

	// Discover
	discovered, err := DiscoverResources(ctx, client, moduleName, namespace)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(discovered), 1, "should discover at least the ConfigMap")

	// Delete
	deleteResult, err := Delete(ctx, client, DeleteOptions{
		ModuleName: moduleName,
		Namespace:  namespace,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, deleteResult.Deleted, 1)
	assert.Empty(t, deleteResult.Errors)
}

// TestApplyIdempotency tests that applying the same resources twice is safe.
func TestApplyIdempotency(t *testing.T) {
	ctx := context.Background()

	client, err := NewClient(ClientOptions{})
	require.NoError(t, err)

	moduleName := "idempotency-test"
	namespace := "default"

	cm := &unstructured.Unstructured{}
	cm.SetAPIVersion("v1")
	cm.SetKind("ConfigMap")
	cm.SetName("opm-idempotency-test")
	cm.SetNamespace(namespace)
	cm.Object["data"] = map[string]interface{}{
		"key": "value",
	}

	resources := []*build.Resource{
		{Object: cm, Component: "test"},
	}
	meta := build.ModuleReleaseMetadata{
		Name:      moduleName,
		Namespace: namespace,
		Version:   "0.1.0",
	}

	// Apply twice
	result1, err := Apply(ctx, client, resources, meta, ApplyOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, result1.Applied)

	result2, err := Apply(ctx, client, resources, meta, ApplyOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, result2.Applied)
	assert.Empty(t, result2.Errors, "second apply should have no errors")

	// Cleanup
	_, err = Delete(ctx, client, DeleteOptions{
		ModuleName: moduleName,
		Namespace:  namespace,
	})
	require.NoError(t, err)
}
