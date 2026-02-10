package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// --- 7.1: Tests for CompareResource (dyff comparer) ---

func TestCompareResource_Identical(t *testing.T) {
	comparer := NewComparer()

	rendered := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "test-cm",
				"namespace": "default",
			},
			"data": map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}

	live := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "test-cm",
				"namespace": "default",
			},
			"data": map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}

	diff, err := comparer.Compare(rendered, live)
	require.NoError(t, err)
	assert.Empty(t, diff, "identical resources should produce no diff")
}

func TestCompareResource_Modified(t *testing.T) {
	comparer := NewComparer()

	rendered := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "test-cm",
				"namespace": "default",
			},
			"data": map[string]interface{}{
				"key1": "new-value",
			},
		},
	}

	live := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "test-cm",
				"namespace": "default",
			},
			"data": map[string]interface{}{
				"key1": "old-value",
			},
		},
	}

	diff, err := comparer.Compare(rendered, live)
	require.NoError(t, err)
	assert.NotEmpty(t, diff, "modified resources should produce a diff")
}

func TestCompareResource_FieldReordering(t *testing.T) {
	comparer := NewComparer()

	// Same data, different field order
	rendered := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "test-cm",
				"namespace": "default",
			},
			"data": map[string]interface{}{
				"alpha": "a",
				"beta":  "b",
				"gamma": "c",
			},
		},
	}

	live := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"namespace": "default",
				"name":      "test-cm",
			},
			"data": map[string]interface{}{
				"gamma": "c",
				"alpha": "a",
				"beta":  "b",
			},
		},
	}

	diff, err := comparer.Compare(rendered, live)
	require.NoError(t, err)
	assert.Empty(t, diff, "field reordering should not produce a diff")
}

// --- 7.2: Tests for resource categorization ---

func TestResourceKey(t *testing.T) {
	tests := []struct {
		name      string
		gvk       schema.GroupVersionKind
		namespace string
		resName   string
		expected  string
	}{
		{
			name:      "core resource",
			gvk:       schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"},
			namespace: "default",
			resName:   "my-cm",
			expected:  "/v1/ConfigMap/default/my-cm",
		},
		{
			name:      "apps resource",
			gvk:       schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
			namespace: "production",
			resName:   "my-deploy",
			expected:  "apps/v1/Deployment/production/my-deploy",
		},
		{
			name:      "cluster-scoped",
			gvk:       schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"},
			namespace: "",
			resName:   "my-role",
			expected:  "rbac.authorization.k8s.io/v1/ClusterRole//my-role",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := resourceKey(tc.gvk, tc.namespace, tc.resName)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// --- 7.4: Tests for summary line formatting ---

func TestDiffResult_SummaryLine(t *testing.T) {
	tests := []struct {
		name     string
		result   DiffResult
		expected string
	}{
		{
			name:     "no differences",
			result:   DiffResult{},
			expected: "No differences found",
		},
		{
			name:     "only modified",
			result:   DiffResult{Modified: 3},
			expected: "3 modified",
		},
		{
			name:     "only added",
			result:   DiffResult{Added: 2},
			expected: "2 added",
		},
		{
			name:     "only orphaned",
			result:   DiffResult{Orphaned: 1},
			expected: "1 orphaned",
		},
		{
			name:     "mixed changes",
			result:   DiffResult{Modified: 2, Added: 1, Orphaned: 1},
			expected: "2 modified, 1 added, 1 orphaned",
		},
		{
			name:     "modified and added",
			result:   DiffResult{Modified: 5, Added: 3},
			expected: "5 modified, 3 added",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.result.SummaryLine())
		})
	}
}

func TestDiffResult_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		result   DiffResult
		expected bool
	}{
		{
			name:     "empty result",
			result:   DiffResult{},
			expected: true,
		},
		{
			name:     "only unchanged",
			result:   DiffResult{Unchanged: 5},
			expected: true,
		},
		{
			name:     "has modified",
			result:   DiffResult{Modified: 1},
			expected: false,
		},
		{
			name:     "has added",
			result:   DiffResult{Added: 1},
			expected: false,
		},
		{
			name:     "has orphaned",
			result:   DiffResult{Orphaned: 1},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.result.IsEmpty())
		})
	}
}
