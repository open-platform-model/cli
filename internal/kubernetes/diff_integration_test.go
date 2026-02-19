//go:build integration

package kubernetes

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/build"
)

// --- 8.1 / 8.2: Integration test for diff showing modifications ---

func TestDiffIntegration_ShowsModifications(t *testing.T) {
	ctx := context.Background()

	client, err := NewClient(ClientOptions{})
	require.NoError(t, err, "need a valid kubeconfig for integration tests")

	releaseName := "diff-integration-test"
	namespace := "default"
	comparer := NewComparer()

	// Create and apply a ConfigMap
	cm := &unstructured.Unstructured{}
	cm.SetAPIVersion("v1")
	cm.SetKind("ConfigMap")
	cm.SetName("opm-diff-test")
	cm.SetNamespace(namespace)
	cm.Object["data"] = map[string]interface{}{
		"key": "original-value",
	}

	resources := []*build.Resource{
		{Object: cm, Component: "test-component"},
	}
	meta := build.ModuleReleaseMetadata{
		Name:      releaseName,
		Namespace: namespace,
		Version:   "0.1.0",
	}

	// Apply the original resource
	applyResult, err := Apply(ctx, client, resources, meta, ApplyOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, applyResult.Applied)

	// Modify locally
	modifiedCM := cm.DeepCopy()
	modifiedCM.Object["data"] = map[string]interface{}{
		"key": "modified-value",
	}
	modifiedResources := []*build.Resource{
		{Object: modifiedCM, Component: "test-component"},
	}

	// Diff should show modifications
	diffResult, err := Diff(ctx, client, modifiedResources, meta, comparer)
	require.NoError(t, err)
	assert.Equal(t, 1, diffResult.Modified, "should detect 1 modified resource")
	assert.Equal(t, 0, diffResult.Added)
	assert.Equal(t, 0, diffResult.Orphaned)

	// Cleanup
	_, err = Delete(ctx, client, DeleteOptions{
		ReleaseName: releaseName,
		Namespace:   namespace,
	})
	require.NoError(t, err)
}

// --- Integration test: apply then diff with no changes shows no differences ---

func TestDiffIntegration_ApplyThenDiffShowsNoDifferences(t *testing.T) {
	ctx := context.Background()

	client, err := NewClient(ClientOptions{})
	require.NoError(t, err, "need a valid kubeconfig for integration tests")

	releaseName := "diff-no-change-test"
	namespace := "default"
	comparer := NewComparer()

	// Create and apply a ConfigMap
	cm := &unstructured.Unstructured{}
	cm.SetAPIVersion("v1")
	cm.SetKind("ConfigMap")
	cm.SetName("opm-diff-no-change-test")
	cm.SetNamespace(namespace)
	cm.Object["data"] = map[string]interface{}{
		"key": "value",
	}

	resources := []*build.Resource{
		{Object: cm, Component: "test-component"},
	}
	meta := build.ModuleReleaseMetadata{
		Name:      releaseName,
		Namespace: namespace,
		Version:   "0.1.0",
	}

	// Apply
	applyResult, err := Apply(ctx, client, resources, meta, ApplyOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, applyResult.Applied)

	// Diff immediately — with field projection, should show "No differences found"
	diffResult, err := Diff(ctx, client, resources, meta, comparer)
	require.NoError(t, err)
	assert.Equal(t, 0, diffResult.Modified, "apply-then-diff with no changes should report 0 modified")
	assert.Equal(t, 0, diffResult.Added, "apply-then-diff with no changes should report 0 added")
	assert.True(t, diffResult.IsEmpty(), "diff result should be empty")
	assert.Equal(t, "No differences found", diffResult.SummaryLine())

	// Cleanup
	_, err = Delete(ctx, client, DeleteOptions{
		ReleaseName: releaseName,
		Namespace:   namespace,
	})
	require.NoError(t, err)
}

// --- 8.3: Integration test for status reporting health ---

func TestStatusIntegration_ReportsHealth(t *testing.T) {
	ctx := context.Background()

	client, err := NewClient(ClientOptions{})
	require.NoError(t, err, "need a valid kubeconfig for integration tests")

	releaseName := "status-integration-test"
	namespace := "default"

	// Create and apply a ConfigMap (passive resource = always healthy)
	cm := &unstructured.Unstructured{}
	cm.SetAPIVersion("v1")
	cm.SetKind("ConfigMap")
	cm.SetName("opm-status-test")
	cm.SetNamespace(namespace)
	cm.Object["data"] = map[string]interface{}{
		"key": "value",
	}

	resources := []*build.Resource{
		{Object: cm, Component: "test-component"},
	}
	meta := build.ModuleReleaseMetadata{
		Name:      releaseName,
		Namespace: namespace,
		Version:   "0.1.0",
	}

	// Apply
	applyResult, err := Apply(ctx, client, resources, meta, ApplyOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, applyResult.Applied)

	// Check status
	statusResult, err := GetReleaseStatus(ctx, client, StatusOptions{
		ReleaseName: releaseName,
		Namespace:   namespace,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(statusResult.Resources), 1)
	assert.Equal(t, healthReady, statusResult.AggregateStatus)

	// Cleanup
	_, err = Delete(ctx, client, DeleteOptions{
		ReleaseName: releaseName,
		Namespace:   namespace,
	})
	require.NoError(t, err)
}

// --- 8.4: Integration test for diff with no prior deployment (all additions) ---

func TestDiffIntegration_AllAdditions(t *testing.T) {
	ctx := context.Background()

	client, err := NewClient(ClientOptions{})
	require.NoError(t, err, "need a valid kubeconfig for integration tests")

	releaseName := "diff-additions-test"
	namespace := "default"
	comparer := NewComparer()

	// Create a resource that doesn't exist on cluster
	cm := &unstructured.Unstructured{}
	cm.SetAPIVersion("v1")
	cm.SetKind("ConfigMap")
	cm.SetName("opm-diff-additions-test")
	cm.SetNamespace(namespace)
	cm.Object["data"] = map[string]interface{}{
		"key": "value",
	}

	resources := []*build.Resource{
		{Object: cm, Component: "test-component"},
	}
	meta := build.ModuleReleaseMetadata{
		Name:      releaseName,
		Namespace: namespace,
		Version:   "0.1.0",
	}

	// Diff without prior deployment — all should be additions
	diffResult, err := Diff(ctx, client, resources, meta, comparer)
	require.NoError(t, err)
	assert.Equal(t, 0, diffResult.Modified)
	assert.Equal(t, 1, diffResult.Added, "resource should show as added")
	assert.Equal(t, 0, diffResult.Orphaned)
}

// --- 8.5: Integration test for status with no matching resources ---

func TestStatusIntegration_NoResources(t *testing.T) {
	ctx := context.Background()

	client, err := NewClient(ClientOptions{})
	require.NoError(t, err, "need a valid kubeconfig for integration tests")

	// Query for a release that doesn't exist
	_, err = GetReleaseStatus(ctx, client, StatusOptions{
		ReleaseName: "nonexistent-module",
		Namespace:   "default",
	})
	// After YAGNI removal, GetReleaseStatus returns noResourcesFoundError
	// when no resources match the selector.
	require.Error(t, err)
	assert.True(t, IsNoResourcesFound(err))
}
