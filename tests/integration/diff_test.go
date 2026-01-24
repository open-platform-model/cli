package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/cue"
	"github.com/opmodel/cli/internal/kubernetes"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestDiff_NoChanges(t *testing.T) {
	if testClient == nil {
		t.Skip("No test client available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a ConfigMap
	cm := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "diff-test-no-changes",
				"namespace": "default",
			},
			"data": map[string]interface{}{
				"key": "value",
			},
		},
	}

	// Apply it first
	labels := kubernetes.ModuleLabels("diff-test", "default", "v1.0.0", "")
	_, err := testClient.Apply(ctx, []*unstructured.Unstructured{cm}, kubernetes.ApplyOptions{
		Namespace: "default",
		Labels:    labels,
	})
	require.NoError(t, err)

	// Diff with same object
	result, err := testClient.Diff(ctx, []*unstructured.Unstructured{cm}, kubernetes.DiffOptions{
		Namespace:       "default",
		ModuleName:      "diff-test",
		ModuleNamespace: "default",
	})
	require.NoError(t, err)

	// No changes expected
	assert.False(t, result.HasChanges)
	assert.Empty(t, result.Added)
	assert.Empty(t, result.Removed)
	assert.Empty(t, result.Modified)

	// Cleanup
	_, _ = testClient.Delete(ctx, []*unstructured.Unstructured{cm}, kubernetes.DeleteOptions{})
}

func TestDiff_DetectsModification(t *testing.T) {
	if testClient == nil {
		t.Skip("No test client available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a ConfigMap
	cmLive := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "diff-test-modify",
				"namespace": "default",
			},
			"data": map[string]interface{}{
				"key": "old-value",
			},
		},
	}

	// Apply it first
	labels := kubernetes.ModuleLabels("diff-test-mod", "default", "v1.0.0", "")
	_, err := testClient.Apply(ctx, []*unstructured.Unstructured{cmLive}, kubernetes.ApplyOptions{
		Namespace: "default",
		Labels:    labels,
	})
	require.NoError(t, err)

	// Create desired state with different value
	cmDesired := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "diff-test-modify",
				"namespace": "default",
			},
			"data": map[string]interface{}{
				"key": "new-value",
			},
		},
	}

	// Diff should detect the change
	result, err := testClient.Diff(ctx, []*unstructured.Unstructured{cmDesired}, kubernetes.DiffOptions{
		Namespace:       "default",
		UseColor:        false,
		ModuleName:      "diff-test-mod",
		ModuleNamespace: "default",
	})
	require.NoError(t, err)

	assert.True(t, result.HasChanges)
	assert.Empty(t, result.Added)
	assert.Empty(t, result.Removed)
	assert.Len(t, result.Modified, 1)
	assert.Contains(t, result.Modified[0].Name, "ConfigMap")
	assert.Contains(t, result.Modified[0].Diff, "key")

	// Cleanup
	_, _ = testClient.Delete(ctx, []*unstructured.Unstructured{cmLive}, kubernetes.DeleteOptions{})
}

func TestDiff_DetectsNewResource(t *testing.T) {
	if testClient == nil {
		t.Skip("No test client available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a ConfigMap that doesn't exist in cluster
	cm := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "diff-test-new-resource",
				"namespace": "default",
			},
			"data": map[string]interface{}{
				"key": "value",
			},
		},
	}

	// Diff should show it as added
	result, err := testClient.Diff(ctx, []*unstructured.Unstructured{cm}, kubernetes.DiffOptions{
		Namespace: "default",
	})
	require.NoError(t, err)

	assert.True(t, result.HasChanges)
	assert.Len(t, result.Added, 1)
	assert.Contains(t, result.Added[0], "ConfigMap")
	assert.Contains(t, result.Added[0], "diff-test-new-resource")
	assert.Empty(t, result.Removed)
	assert.Empty(t, result.Modified)
}

func TestDiff_ModuleWorkflow(t *testing.T) {
	if testClient == nil {
		t.Skip("No test client available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Load test module
	loader := cue.NewLoader()
	module, err := loader.LoadModule(ctx, "../fixtures/hello-world", nil)
	require.NoError(t, err)

	// Render manifests
	renderer := cue.NewRenderer()
	manifestSet, err := renderer.RenderModule(ctx, module)
	require.NoError(t, err)
	require.Greater(t, manifestSet.Len(), 0)

	// Before applying - all resources should be "added"
	objects := manifestSet.Objects()
	result, err := testClient.Diff(ctx, objects, kubernetes.DiffOptions{
		Namespace:       "default",
		ModuleName:      module.Metadata.Name,
		ModuleNamespace: "default",
	})
	require.NoError(t, err)

	// All should be added since nothing is deployed yet
	assert.True(t, result.HasChanges)
	assert.Equal(t, len(objects), len(result.Added), "All resources should be marked as added")
	assert.Empty(t, result.Modified)
	// Note: Removed may be empty or may have resources from previous test runs
}

func TestDiff_DryRunConsistency(t *testing.T) {
	if testClient == nil {
		t.Skip("No test client available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a ConfigMap
	cm := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "diff-test-dry-run",
				"namespace": "default",
			},
			"data": map[string]interface{}{
				"key": "value",
			},
		},
	}

	// Apply with dry-run first
	labels := kubernetes.ModuleLabels("diff-dry-run", "default", "v1.0.0", "")
	_, err := testClient.Apply(ctx, []*unstructured.Unstructured{cm}, kubernetes.ApplyOptions{
		Namespace: "default",
		Labels:    labels,
		DryRun:    true,
	})
	require.NoError(t, err)

	// Diff should still show as added (dry-run doesn't create)
	result, err := testClient.Diff(ctx, []*unstructured.Unstructured{cm}, kubernetes.DiffOptions{
		Namespace: "default",
	})
	require.NoError(t, err)

	assert.True(t, result.HasChanges)
	assert.Len(t, result.Added, 1)
}
