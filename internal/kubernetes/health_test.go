package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// --- 7.3: Tests for EvaluateHealth ---

func makeResource(kind string, conditions []map[string]interface{}) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       kind,
			"metadata": map[string]interface{}{
				"name":      "test-resource",
				"namespace": "default",
			},
		},
	}

	if conditions != nil {
		rawConditions := make([]interface{}, len(conditions))
		for i, c := range conditions {
			rawConditions[i] = c
		}
		obj.Object["status"] = map[string]interface{}{
			"conditions": rawConditions,
		}
	}

	return obj
}

func TestEvaluateHealth_Workloads(t *testing.T) {
	tests := []struct {
		name       string
		kind       string
		conditions []map[string]interface{}
		expected   HealthStatus
	}{
		{
			name: "Deployment with Available=True",
			kind: "Deployment",
			conditions: []map[string]interface{}{
				{"type": "Available", "status": "True"},
			},
			expected: HealthReady,
		},
		{
			name: "Deployment with Available=False",
			kind: "Deployment",
			conditions: []map[string]interface{}{
				{"type": "Available", "status": "False"},
			},
			expected: HealthNotReady,
		},
		{
			name:       "Deployment with no conditions",
			kind:       "Deployment",
			conditions: nil,
			expected:   HealthNotReady,
		},
		{
			name: "StatefulSet with Ready=True",
			kind: "StatefulSet",
			conditions: []map[string]interface{}{
				{"type": "Ready", "status": "True"},
			},
			expected: HealthReady,
		},
		{
			name: "DaemonSet with Available=False",
			kind: "DaemonSet",
			conditions: []map[string]interface{}{
				{"type": "Available", "status": "False"},
			},
			expected: HealthNotReady,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resource := makeResource(tc.kind, tc.conditions)
			assert.Equal(t, tc.expected, EvaluateHealth(resource))
		})
	}
}

func TestEvaluateHealth_Job(t *testing.T) {
	tests := []struct {
		name       string
		conditions []map[string]interface{}
		expected   HealthStatus
	}{
		{
			name: "Job completed",
			conditions: []map[string]interface{}{
				{"type": "Complete", "status": "True"},
			},
			expected: HealthComplete,
		},
		{
			name: "Job failed",
			conditions: []map[string]interface{}{
				{"type": "Failed", "status": "True"},
			},
			expected: HealthNotReady,
		},
		{
			name:       "Job in progress",
			conditions: nil,
			expected:   HealthNotReady,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resource := makeResource("Job", tc.conditions)
			assert.Equal(t, tc.expected, EvaluateHealth(resource))
		})
	}
}

func TestEvaluateHealth_CronJob(t *testing.T) {
	resource := makeResource("CronJob", nil)
	assert.Equal(t, HealthReady, EvaluateHealth(resource))
}

func TestEvaluateHealth_Passive(t *testing.T) {
	passiveResources := []string{
		"ConfigMap", "Secret", "Service", "PersistentVolumeClaim",
		"ServiceAccount", "Namespace", "ClusterRole", "ClusterRoleBinding",
		"Role", "RoleBinding",
	}

	for _, kind := range passiveResources {
		t.Run(kind, func(t *testing.T) {
			resource := makeResource(kind, nil)
			assert.Equal(t, HealthReady, EvaluateHealth(resource))
		})
	}
}

func TestEvaluateHealth_Custom(t *testing.T) {
	tests := []struct {
		name       string
		conditions []map[string]interface{}
		expected   HealthStatus
	}{
		{
			name: "Custom with Ready=True",
			conditions: []map[string]interface{}{
				{"type": "Ready", "status": "True"},
			},
			expected: HealthReady,
		},
		{
			name: "Custom with Ready=False",
			conditions: []map[string]interface{}{
				{"type": "Ready", "status": "False"},
			},
			expected: HealthNotReady,
		},
		{
			name:       "Custom without Ready condition (passive fallback)",
			conditions: nil,
			expected:   HealthReady,
		},
		{
			name: "Custom with other conditions but no Ready",
			conditions: []map[string]interface{}{
				{"type": "Synced", "status": "True"},
			},
			expected: HealthReady,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resource := makeResource("MyCustomResource", tc.conditions)
			assert.Equal(t, tc.expected, EvaluateHealth(resource))
		})
	}
}
