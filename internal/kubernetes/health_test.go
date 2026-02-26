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

// makeStatefulSet builds a StatefulSet resource with the given spec and ready replica counts.
func makeStatefulSet(specReplicas *int64, readyReplicas int64) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "StatefulSet",
			"metadata":   map[string]interface{}{"name": "test-ss", "namespace": "default"},
			"status":     map[string]interface{}{"readyReplicas": readyReplicas},
		},
	}
	if specReplicas != nil {
		obj.Object["spec"] = map[string]interface{}{"replicas": *specReplicas}
	}
	return obj
}

func ptr64(v int64) *int64 { return &v }

func TestEvaluateHealth_StatefulSet(t *testing.T) {
	tests := []struct {
		name          string
		specReplicas  *int64
		readyReplicas int64
		expected      HealthStatus
	}{
		{
			name:          "1/1 ready",
			specReplicas:  ptr64(1),
			readyReplicas: 1,
			expected:      HealthReady,
		},
		{
			name:          "3/3 ready",
			specReplicas:  ptr64(3),
			readyReplicas: 3,
			expected:      HealthReady,
		},
		{
			name:          "0/1 ready",
			specReplicas:  ptr64(1),
			readyReplicas: 0,
			expected:      HealthNotReady,
		},
		{
			name:          "1/3 ready",
			specReplicas:  ptr64(3),
			readyReplicas: 1,
			expected:      HealthNotReady,
		},
		{
			name:          "spec.replicas omitted defaults to 1, pod ready",
			specReplicas:  nil,
			readyReplicas: 1,
			expected:      HealthReady,
		},
		{
			name:          "spec.replicas omitted defaults to 1, pod not ready",
			specReplicas:  nil,
			readyReplicas: 0,
			expected:      HealthNotReady,
		},
		{
			name:          "scaled to zero is always ready",
			specReplicas:  ptr64(0),
			readyReplicas: 0,
			expected:      HealthReady,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resource := makeStatefulSet(tc.specReplicas, tc.readyReplicas)
			assert.Equal(t, tc.expected, EvaluateHealth(resource))
		})
	}
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
	// PersistentVolumeClaim is intentionally excluded — it has its own evaluatePVCHealth branch.
	passiveResources := []string{
		"ConfigMap", "Secret", "Service",
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

func TestEvaluateHealth_PVC(t *testing.T) {
	tests := []struct {
		name     string
		phase    string
		expected HealthStatus
	}{
		{name: "Bound", phase: "Bound", expected: HealthBound},
		{name: "Pending", phase: "Pending", expected: HealthStatus("Pending")},
		{name: "Lost", phase: "Lost", expected: HealthStatus("Lost")},
		{name: "no phase (fallback)", phase: "", expected: HealthReady},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pvc := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "PersistentVolumeClaim",
					"metadata":   map[string]interface{}{"name": "data", "namespace": "ns"},
				},
			}
			if tc.phase != "" {
				_ = unstructured.SetNestedField(pvc.Object, tc.phase, "status", "phase")
			}
			assert.Equal(t, tc.expected, EvaluateHealth(pvc))
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
