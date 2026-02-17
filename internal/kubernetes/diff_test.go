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

// --- Tests for projection + comparer integration ---

func TestCompareResource_ServerManagedFieldsNoLongerProduceDiffs(t *testing.T) {
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
			},
		},
	}

	// Live object has server-managed fields that would previously produce diffs
	live := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":              "test-cm",
				"namespace":         "default",
				"managedFields":     []interface{}{map[string]interface{}{"manager": "kubectl"}},
				"uid":               "abc-123-def",
				"resourceVersion":   "99999",
				"creationTimestamp": "2024-01-01T00:00:00Z",
				"generation":        int64(3),
			},
			"data": map[string]interface{}{
				"key1": "value1",
			},
			"status": map[string]interface{}{
				"phase": "Active",
			},
		},
	}

	// Apply the same filtering the Diff() function applies
	stripServerManagedFields(live.Object)
	live.Object = projectLiveToRendered(rendered.Object, live.Object)

	diff, err := comparer.Compare(rendered, live)
	require.NoError(t, err)
	assert.Empty(t, diff, "server-managed fields should not produce diffs after projection")
}

func TestCompareResource_ServerDefaultsNoLongerProduceDiffs(t *testing.T) {
	comparer := NewComparer()

	rendered := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "StatefulSet",
			"metadata": map[string]interface{}{
				"name":      "test-ss",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": int64(1),
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{"app": "test"},
				},
			},
		},
	}

	live := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "StatefulSet",
			"metadata": map[string]interface{}{
				"name":              "test-ss",
				"namespace":         "default",
				"uid":               "xyz-789",
				"resourceVersion":   "1234",
				"creationTimestamp": "2024-06-01T00:00:00Z",
				"generation":        int64(1),
			},
			"spec": map[string]interface{}{
				"replicas":             int64(1),
				"podManagementPolicy":  "OrderedReady",
				"revisionHistoryLimit": int64(10),
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{"app": "test"},
				},
			},
			"status": map[string]interface{}{
				"replicas": int64(1),
			},
		},
	}

	stripServerManagedFields(live.Object)
	live.Object = projectLiveToRendered(rendered.Object, live.Object)

	diff, err := comparer.Compare(rendered, live)
	require.NoError(t, err)
	assert.Empty(t, diff, "server defaults should not produce diffs after projection")
}

func TestCompareResource_ActualChangesStillAppearAfterProjection(t *testing.T) {
	comparer := NewComparer()

	rendered := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "test-deploy",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": int64(3),
			},
		},
	}

	live := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":              "test-deploy",
				"namespace":         "default",
				"uid":               "aaa-bbb",
				"resourceVersion":   "5555",
				"creationTimestamp": "2024-01-01T00:00:00Z",
			},
			"spec": map[string]interface{}{
				"replicas":             int64(1),
				"revisionHistoryLimit": int64(10),
			},
			"status": map[string]interface{}{
				"availableReplicas": int64(1),
			},
		},
	}

	stripServerManagedFields(live.Object)
	live.Object = projectLiveToRendered(rendered.Object, live.Object)

	diff, err := comparer.Compare(rendered, live)
	require.NoError(t, err)
	assert.NotEmpty(t, diff, "actual value changes should still produce diffs")
	assert.Contains(t, diff, "replicas", "diff should mention the changed field")
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

// --- Tests for stripServerManagedFields ---

func TestStripServerManagedFields_RemovesAllServerFields(t *testing.T) {
	obj := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":              "test",
			"namespace":         "default",
			"managedFields":     []interface{}{"some-data"},
			"uid":               "abc-123",
			"resourceVersion":   "12345",
			"creationTimestamp": "2024-01-01T00:00:00Z",
			"generation":        int64(1),
			"labels":            map[string]interface{}{"app": "test"},
			"annotations":       map[string]interface{}{"note": "keep-me"},
		},
		"status": map[string]interface{}{
			"phase": "Active",
		},
		"data": map[string]interface{}{
			"key": "value",
		},
	}

	stripServerManagedFields(obj)

	// Server fields removed
	meta := obj["metadata"].(map[string]interface{})
	assert.NotContains(t, meta, "managedFields")
	assert.NotContains(t, meta, "uid")
	assert.NotContains(t, meta, "resourceVersion")
	assert.NotContains(t, meta, "creationTimestamp")
	assert.NotContains(t, meta, "generation")
	assert.NotContains(t, obj, "status")

	// Non-server fields preserved
	assert.Equal(t, "test", meta["name"])
	assert.Equal(t, "default", meta["namespace"])
	assert.Equal(t, map[string]interface{}{"app": "test"}, meta["labels"])
	assert.Equal(t, map[string]interface{}{"note": "keep-me"}, meta["annotations"])
	assert.Equal(t, map[string]interface{}{"key": "value"}, obj["data"])
	assert.Equal(t, "v1", obj["apiVersion"])
	assert.Equal(t, "ConfigMap", obj["kind"])
}

func TestStripServerManagedFields_SafeOnMissingFields(t *testing.T) {
	// Object with no metadata at all
	obj := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
	}
	// Should not panic
	stripServerManagedFields(obj)
	assert.Equal(t, "v1", obj["apiVersion"])

	// Object with empty metadata
	obj2 := map[string]interface{}{
		"metadata": map[string]interface{}{},
	}
	stripServerManagedFields(obj2)
	assert.Empty(t, obj2["metadata"])
}

func TestStripServerManagedFields_NonMapMetadata(t *testing.T) {
	// Edge case: metadata is not a map (shouldn't happen in practice)
	obj := map[string]interface{}{
		"metadata": "not-a-map",
		"status":   "removed",
	}
	stripServerManagedFields(obj)
	assert.NotContains(t, obj, "status")
	// metadata left as-is since it's not a map
	assert.Equal(t, "not-a-map", obj["metadata"])
}

// --- Tests for projectLiveToRendered ---

func TestProjectLiveToRendered(t *testing.T) {
	tests := []struct {
		name     string
		rendered map[string]interface{}
		live     map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "identical objects",
			rendered: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"data":       map[string]interface{}{"key": "value"},
			},
			live: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"data":       map[string]interface{}{"key": "value"},
			},
			expected: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"data":       map[string]interface{}{"key": "value"},
			},
		},
		{
			name: "server defaults stripped from live",
			rendered: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Service",
				"spec": map[string]interface{}{
					"ports": []interface{}{
						map[string]interface{}{"port": int64(80)},
					},
				},
			},
			live: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Service",
				"spec": map[string]interface{}{
					"ports": []interface{}{
						map[string]interface{}{"port": int64(80), "protocol": "TCP", "targetPort": int64(80)},
					},
					"clusterIP":       "10.0.0.1",
					"sessionAffinity": "None",
				},
			},
			expected: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Service",
				"spec": map[string]interface{}{
					"ports": []interface{}{
						map[string]interface{}{"port": int64(80)},
					},
				},
			},
		},
		{
			name: "nested map projection",
			rendered: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "test",
					"labels": map[string]interface{}{
						"app": "myapp",
					},
				},
			},
			live: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "test",
					"labels": map[string]interface{}{
						"app":                          "myapp",
						"app.kubernetes.io/managed-by": "Helm",
					},
					"annotations": map[string]interface{}{
						"kubectl.kubernetes.io/last-applied": "...",
					},
				},
			},
			expected: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "test",
					"labels": map[string]interface{}{
						"app": "myapp",
					},
				},
			},
		},
		{
			name: "list matching by name",
			rendered: map[string]interface{}{
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"name":  "app",
							"image": "myapp:v2",
						},
					},
				},
			},
			live: map[string]interface{}{
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"name":                     "app",
							"image":                    "myapp:v1",
							"terminationMessagePath":   "/dev/termination-log",
							"terminationMessagePolicy": "File",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"name":  "app",
							"image": "myapp:v1",
						},
					},
				},
			},
		},
		{
			name: "list fallback to index matching",
			rendered: map[string]interface{}{
				"spec": map[string]interface{}{
					"ports": []interface{}{
						map[string]interface{}{
							"port":       int64(80),
							"targetPort": int64(8080),
						},
					},
				},
			},
			live: map[string]interface{}{
				"spec": map[string]interface{}{
					"ports": []interface{}{
						map[string]interface{}{
							"port":       int64(80),
							"targetPort": int64(8080),
							"protocol":   "TCP",
							"nodePort":   int64(30080),
						},
					},
				},
			},
			expected: map[string]interface{}{
				"spec": map[string]interface{}{
					"ports": []interface{}{
						map[string]interface{}{
							"port":       int64(80),
							"targetPort": int64(8080),
						},
					},
				},
			},
		},
		{
			name: "empty rendered map preserved",
			rendered: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":        "test",
					"annotations": map[string]interface{}{},
				},
			},
			live: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "test",
					"annotations": map[string]interface{}{
						"server-only": "value",
					},
				},
			},
			expected: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "test",
					// Empty rendered map is preserved (not stripped) so both
					// sides match and dyff sees no spurious diff.
					"annotations": map[string]interface{}{},
				},
			},
		},
		{
			name: "non-empty map with no overlapping keys is stripped",
			rendered: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "test",
					"annotations": map[string]interface{}{
						"user-defined": "keep",
					},
				},
			},
			live: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "test",
					"annotations": map[string]interface{}{
						"server-only": "value",
					},
				},
			},
			expected: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "test",
					// rendered has "user-defined" but live doesn't — rendered
					// value kept. "server-only" stripped. Non-empty rendered map
					// with no overlapping live keys still produces content.
					"annotations": map[string]interface{}{
						"user-defined": "keep",
					},
				},
			},
		},
		{
			name: "scalar list preservation",
			rendered: map[string]interface{}{
				"spec": map[string]interface{}{
					"finalizers": []interface{}{"kubernetes.io/pvc-protection"},
				},
			},
			live: map[string]interface{}{
				"spec": map[string]interface{}{
					"finalizers": []interface{}{"kubernetes.io/pvc-protection", "extra-finalizer"},
				},
			},
			expected: map[string]interface{}{
				"spec": map[string]interface{}{
					"finalizers": []interface{}{"kubernetes.io/pvc-protection", "extra-finalizer"},
				},
			},
		},
		{
			name: "missing keys in live",
			rendered: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"data": map[string]interface{}{
					"new-key": "new-value",
				},
			},
			live: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"data": map[string]interface{}{
					"old-key": "old-value",
				},
			},
			expected: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"data": map[string]interface{}{
					// new-key not in live, so rendered value is used (shows as addition)
					"new-key": "new-value",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := projectLiveToRendered(tc.rendered, tc.live)
			assert.Equal(t, tc.expected, result)
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
