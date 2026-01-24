package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestResourceKey(t *testing.T) {
	tests := []struct {
		name      string
		kind      string
		namespace string
		objName   string
		want      string
	}{
		{
			name:      "namespaced resource",
			kind:      "Deployment",
			namespace: "default",
			objName:   "my-app",
			want:      "Deployment/default/my-app",
		},
		{
			name:      "cluster-scoped resource",
			kind:      "Namespace",
			namespace: "",
			objName:   "my-ns",
			want:      "Namespace//my-ns",
		},
		{
			name:      "resource in custom namespace",
			kind:      "Service",
			namespace: "production",
			objName:   "api-gateway",
			want:      "Service/production/api-gateway",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &unstructured.Unstructured{}
			obj.SetKind(tt.kind)
			obj.SetNamespace(tt.namespace)
			obj.SetName(tt.objName)

			got := ResourceKey(obj)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStripManagedFields(t *testing.T) {
	t.Run("removes server-managed metadata fields", func(t *testing.T) {
		obj := map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":              "test",
				"namespace":         "default",
				"resourceVersion":   "12345",
				"uid":               "abc-123-def-456",
				"creationTimestamp": "2024-01-01T00:00:00Z",
				"generation":        int64(5),
				"managedFields":     []interface{}{},
				"selfLink":          "/apis/apps/v1/namespaces/default/deployments/test",
				"labels": map[string]interface{}{
					"app": "test",
				},
			},
			"spec": map[string]interface{}{
				"replicas": int64(3),
			},
			"status": map[string]interface{}{
				"availableReplicas": int64(3),
				"readyReplicas":     int64(3),
			},
		}

		stripManagedFields(obj)

		metadata := obj["metadata"].(map[string]interface{})

		// These should be preserved
		assert.Equal(t, "test", metadata["name"])
		assert.Equal(t, "default", metadata["namespace"])
		assert.Equal(t, map[string]interface{}{"app": "test"}, metadata["labels"])

		// These should be removed
		assert.NotContains(t, metadata, "resourceVersion")
		assert.NotContains(t, metadata, "uid")
		assert.NotContains(t, metadata, "creationTimestamp")
		assert.NotContains(t, metadata, "generation")
		assert.NotContains(t, metadata, "managedFields")
		assert.NotContains(t, metadata, "selfLink")

		// Status should be removed entirely
		assert.NotContains(t, obj, "status")

		// Spec should be preserved
		assert.Contains(t, obj, "spec")
	})

	t.Run("removes kubectl last-applied-configuration annotation", func(t *testing.T) {
		obj := map[string]interface{}{
			"metadata": map[string]interface{}{
				"name": "test",
				"annotations": map[string]interface{}{
					"kubectl.kubernetes.io/last-applied-configuration": `{"some":"json"}`,
					"custom-annotation": "keep-me",
				},
			},
		}

		stripManagedFields(obj)

		metadata := obj["metadata"].(map[string]interface{})
		annotations := metadata["annotations"].(map[string]interface{})

		assert.NotContains(t, annotations, "kubectl.kubernetes.io/last-applied-configuration")
		assert.Equal(t, "keep-me", annotations["custom-annotation"])
	})

	t.Run("removes empty annotations map", func(t *testing.T) {
		obj := map[string]interface{}{
			"metadata": map[string]interface{}{
				"name": "test",
				"annotations": map[string]interface{}{
					"kubectl.kubernetes.io/last-applied-configuration": `{"some":"json"}`,
				},
			},
		}

		stripManagedFields(obj)

		metadata := obj["metadata"].(map[string]interface{})
		assert.NotContains(t, metadata, "annotations")
	})

	t.Run("handles missing metadata", func(t *testing.T) {
		obj := map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
		}

		// Should not panic
		stripManagedFields(obj)

		assert.NotContains(t, obj, "metadata")
	})
}

func TestSerializeForDiff(t *testing.T) {
	t.Run("produces valid YAML without managed fields", func(t *testing.T) {
		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]interface{}{
					"name":            "test",
					"namespace":       "default",
					"resourceVersion": "12345",
				},
				"spec": map[string]interface{}{
					"replicas": int64(3),
				},
			},
		}

		data, err := serializeForDiff(obj)
		require.NoError(t, err)

		yamlStr := string(data)
		assert.Contains(t, yamlStr, "apiVersion: apps/v1")
		assert.Contains(t, yamlStr, "kind: Deployment")
		assert.Contains(t, yamlStr, "name: test")
		assert.Contains(t, yamlStr, "replicas: 3")
		assert.NotContains(t, yamlStr, "resourceVersion")
	})

	t.Run("does not modify original object", func(t *testing.T) {
		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name":            "test",
					"resourceVersion": "12345",
				},
			},
		}

		_, err := serializeForDiff(obj)
		require.NoError(t, err)

		// Original should still have resourceVersion
		metadata := obj.Object["metadata"].(map[string]interface{})
		assert.Equal(t, "12345", metadata["resourceVersion"])
	})
}

func TestDiffYAMLWithColor(t *testing.T) {
	t.Run("returns empty for identical documents", func(t *testing.T) {
		yaml := []byte("foo: bar\nbaz: qux\n")

		diff, err := diffYAMLWithColor(yaml, yaml, false)
		require.NoError(t, err)
		assert.Empty(t, diff)
	})

	t.Run("detects value changes", func(t *testing.T) {
		live := []byte("replicas: 3\n")
		desired := []byte("replicas: 5\n")

		diff, err := diffYAMLWithColor(live, desired, false)
		require.NoError(t, err)
		assert.NotEmpty(t, diff)
		assert.Contains(t, diff, "replicas")
	})

	t.Run("detects added fields", func(t *testing.T) {
		live := []byte("foo: bar\n")
		desired := []byte("foo: bar\nbaz: qux\n")

		diff, err := diffYAMLWithColor(live, desired, false)
		require.NoError(t, err)
		assert.NotEmpty(t, diff)
		assert.Contains(t, diff, "baz")
	})

	t.Run("detects removed fields", func(t *testing.T) {
		live := []byte("foo: bar\nbaz: qux\n")
		desired := []byte("foo: bar\n")

		diff, err := diffYAMLWithColor(live, desired, false)
		require.NoError(t, err)
		assert.NotEmpty(t, diff)
		assert.Contains(t, diff, "baz")
	})

	t.Run("returns error for empty vs non-empty", func(t *testing.T) {
		// dyff doesn't support comparing documents with different counts
		// This is expected - in practice we handle empty cases before calling dyff
		live := []byte("")
		desired := []byte("foo: bar\n")

		_, err := diffYAMLWithColor(live, desired, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "different number of documents")
	})

	t.Run("returns error for non-empty vs empty", func(t *testing.T) {
		live := []byte("foo: bar\n")
		desired := []byte("")

		_, err := diffYAMLWithColor(live, desired, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "different number of documents")
	})

	t.Run("handles both empty", func(t *testing.T) {
		diff, err := diffYAMLWithColor([]byte(""), []byte(""), false)
		require.NoError(t, err)
		assert.Empty(t, diff)
	})

	t.Run("returns error for invalid YAML", func(t *testing.T) {
		live := []byte("foo: [invalid")
		desired := []byte("foo: bar\n")

		_, err := diffYAMLWithColor(live, desired, false)
		require.Error(t, err)
	})
}

func TestDiffResult(t *testing.T) {
	t.Run("new result is empty", func(t *testing.T) {
		result := NewDiffResult()

		assert.True(t, result.IsEmpty())
		assert.False(t, result.HasChanges)
		assert.Empty(t, result.Added)
		assert.Empty(t, result.Removed)
		assert.Empty(t, result.Modified)
	})

	t.Run("AddAdded marks HasChanges", func(t *testing.T) {
		result := NewDiffResult()
		result.AddAdded("Deployment/default/test")

		assert.False(t, result.IsEmpty())
		assert.True(t, result.HasChanges)
		assert.Equal(t, []string{"Deployment/default/test"}, result.Added)
	})

	t.Run("AddRemoved marks HasChanges", func(t *testing.T) {
		result := NewDiffResult()
		result.AddRemoved("Service/default/old-svc")

		assert.False(t, result.IsEmpty())
		assert.True(t, result.HasChanges)
		assert.Equal(t, []string{"Service/default/old-svc"}, result.Removed)
	})

	t.Run("AddModified marks HasChanges", func(t *testing.T) {
		result := NewDiffResult()
		result.AddModified("ConfigMap/default/config", "some diff")

		assert.False(t, result.IsEmpty())
		assert.True(t, result.HasChanges)
		assert.Len(t, result.Modified, 1)
		assert.Equal(t, "ConfigMap/default/config", result.Modified[0].Name)
		assert.Equal(t, "some diff", result.Modified[0].Diff)
	})

	t.Run("Summary formats correctly", func(t *testing.T) {
		result := NewDiffResult()
		assert.Equal(t, "No changes", result.Summary())

		result.AddAdded("a")
		assert.Equal(t, "1 added", result.Summary())

		result.AddRemoved("b")
		assert.Equal(t, "1 added, 1 removed", result.Summary())

		result.AddModified("c", "diff")
		assert.Equal(t, "1 added, 1 removed, 1 modified", result.Summary())
	})
}
