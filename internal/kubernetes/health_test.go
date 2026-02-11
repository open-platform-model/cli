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
		expected   healthStatus
	}{
		{
			name: "Deployment with Available=True",
			kind: "Deployment",
			conditions: []map[string]interface{}{
				{"type": "Available", "status": "True"},
			},
			expected: healthReady,
		},
		{
			name: "Deployment with Available=False",
			kind: "Deployment",
			conditions: []map[string]interface{}{
				{"type": "Available", "status": "False"},
			},
			expected: healthNotReady,
		},
		{
			name:       "Deployment with no conditions",
			kind:       "Deployment",
			conditions: nil,
			expected:   healthNotReady,
		},
		{
			name: "StatefulSet with Ready=True",
			kind: "StatefulSet",
			conditions: []map[string]interface{}{
				{"type": "Ready", "status": "True"},
			},
			expected: healthReady,
		},
		{
			name: "DaemonSet with Available=False",
			kind: "DaemonSet",
			conditions: []map[string]interface{}{
				{"type": "Available", "status": "False"},
			},
			expected: healthNotReady,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resource := makeResource(tc.kind, tc.conditions)
			assert.Equal(t, tc.expected, evaluateHealth(resource))
		})
	}
}

func TestEvaluateHealth_Job(t *testing.T) {
	tests := []struct {
		name       string
		conditions []map[string]interface{}
		expected   healthStatus
	}{
		{
			name: "Job completed",
			conditions: []map[string]interface{}{
				{"type": "Complete", "status": "True"},
			},
			expected: healthComplete,
		},
		{
			name: "Job failed",
			conditions: []map[string]interface{}{
				{"type": "Failed", "status": "True"},
			},
			expected: healthNotReady,
		},
		{
			name:       "Job in progress",
			conditions: nil,
			expected:   healthNotReady,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resource := makeResource("Job", tc.conditions)
			assert.Equal(t, tc.expected, evaluateHealth(resource))
		})
	}
}

func TestEvaluateHealth_CronJob(t *testing.T) {
	resource := makeResource("CronJob", nil)
	assert.Equal(t, healthReady, evaluateHealth(resource))
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
			assert.Equal(t, healthReady, evaluateHealth(resource))
		})
	}
}

func TestEvaluateHealth_Custom(t *testing.T) {
	tests := []struct {
		name       string
		conditions []map[string]interface{}
		expected   healthStatus
	}{
		{
			name: "Custom with Ready=True",
			conditions: []map[string]interface{}{
				{"type": "Ready", "status": "True"},
			},
			expected: healthReady,
		},
		{
			name: "Custom with Ready=False",
			conditions: []map[string]interface{}{
				{"type": "Ready", "status": "False"},
			},
			expected: healthNotReady,
		},
		{
			name:       "Custom without Ready condition (passive fallback)",
			conditions: nil,
			expected:   healthReady,
		},
		{
			name: "Custom with other conditions but no Ready",
			conditions: []map[string]interface{}{
				{"type": "Synced", "status": "True"},
			},
			expected: healthReady,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resource := makeResource("MyCustomResource", tc.conditions)
			assert.Equal(t, tc.expected, evaluateHealth(resource))
		})
	}
}
