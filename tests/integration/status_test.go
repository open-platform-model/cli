package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/kubernetes"
)

func TestStatus_EmptyModule(t *testing.T) {
	if testClient == nil {
		t.Skip("No test client available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get status of non-existent module
	status, err := testClient.GetModuleStatus(ctx, "non-existent", "default", "")
	require.NoError(t, err)
	require.NotNil(t, status)
	require.Empty(t, status.Resources)
	require.Equal(t, 0, status.Summary.Total)
}

func TestStatus_HealthEvaluation(t *testing.T) {
	// Test health evaluation logic with mock resources
	evaluator := kubernetes.NewHealthEvaluator()

	tests := []struct {
		name     string
		kind     string
		obj      *unstructured.Unstructured
		expected kubernetes.HealthStatus
	}{
		{
			name: "ConfigMap is always ready",
			kind: "ConfigMap",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]interface{}{
						"name":              "test",
						"creationTimestamp": "2024-01-01T00:00:00Z",
					},
				},
			},
			expected: kubernetes.HealthReady,
		},
		{
			name: "Secret is always ready",
			kind: "Secret",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Secret",
					"metadata": map[string]interface{}{
						"name":              "test",
						"creationTimestamp": "2024-01-01T00:00:00Z",
					},
				},
			},
			expected: kubernetes.HealthReady,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluator.EvaluateHealth(tt.obj)
			require.Equal(t, tt.expected, result.Health)
		})
	}
}
