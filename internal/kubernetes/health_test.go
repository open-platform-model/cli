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

// makeWorkload builds an apps/v1 workload with the given metadata.generation,
// spec and status maps. A nil spec or status is omitted.
func makeWorkload(kind string, generation int64, spec, status map[string]interface{}) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       kind,
			"metadata": map[string]interface{}{
				"name":       "test-workload",
				"namespace":  "default",
				"generation": generation,
			},
		},
	}
	if spec != nil {
		obj.Object["spec"] = spec
	}
	if status != nil {
		obj.Object["status"] = status
	}
	return obj
}

func TestEvaluateHealth_Deployment(t *testing.T) {
	progressDeadline := []interface{}{
		map[string]interface{}{"type": "Progressing", "status": "False", "reason": "ProgressDeadlineExceeded"},
		map[string]interface{}{"type": "Available", "status": "True"},
	}
	tests := []struct {
		name       string
		generation int64
		spec       map[string]interface{}
		status     map[string]interface{}
		expected   HealthStatus
	}{
		{
			name:       "fully rolled out",
			generation: 2,
			spec:       map[string]interface{}{"replicas": int64(3)},
			status: map[string]interface{}{
				"observedGeneration": int64(2), "replicas": int64(3),
				"updatedReplicas": int64(3), "availableReplicas": int64(3),
			},
			expected: HealthReady,
		},
		{
			name:       "stuck upgrade: old ReplicaSet serves, Available=True (issue 228)",
			generation: 2,
			spec:       map[string]interface{}{"replicas": int64(2)},
			status: map[string]interface{}{
				"observedGeneration": int64(2), "replicas": int64(2),
				"updatedReplicas": int64(1), "availableReplicas": int64(1), "readyReplicas": int64(1),
				"conditions": []interface{}{
					map[string]interface{}{"type": "Available", "status": "True", "reason": "MinimumReplicasAvailable"},
					map[string]interface{}{"type": "Progressing", "status": "True", "reason": "ReplicaSetUpdated"},
				},
			},
			expected: HealthNotReady,
		},
		{
			name:       "controller has not observed the latest generation",
			generation: 3,
			spec:       map[string]interface{}{"replicas": int64(1)},
			status: map[string]interface{}{
				"observedGeneration": int64(2), "replicas": int64(1),
				"updatedReplicas": int64(1), "availableReplicas": int64(1),
			},
			expected: HealthNotReady,
		},
		{
			name:       "all replicas updated but not all available",
			generation: 1,
			spec:       map[string]interface{}{"replicas": int64(2)},
			status: map[string]interface{}{
				"observedGeneration": int64(1), "replicas": int64(2),
				"updatedReplicas": int64(2), "availableReplicas": int64(1),
			},
			expected: HealthNotReady,
		},
		{
			name:       "old replicas still terminating",
			generation: 1,
			spec:       map[string]interface{}{"replicas": int64(1)},
			status: map[string]interface{}{
				"observedGeneration": int64(1), "replicas": int64(2),
				"updatedReplicas": int64(1), "availableReplicas": int64(1),
			},
			expected: HealthNotReady,
		},
		{
			name:       "ProgressDeadlineExceeded is never healthy",
			generation: 1,
			spec:       map[string]interface{}{"replicas": int64(1)},
			status: map[string]interface{}{
				"observedGeneration": int64(1), "replicas": int64(1),
				"updatedReplicas": int64(1), "availableReplicas": int64(1),
				"conditions": progressDeadline,
			},
			expected: HealthNotReady,
		},
		{
			name:       "spec.replicas omitted defaults to 1, available",
			generation: 1,
			spec:       map[string]interface{}{},
			status: map[string]interface{}{
				"observedGeneration": int64(1), "replicas": int64(1),
				"updatedReplicas": int64(1), "availableReplicas": int64(1),
			},
			expected: HealthReady,
		},
		{
			name:       "spec.replicas omitted defaults to 1, none available",
			generation: 1,
			spec:       map[string]interface{}{},
			status:     map[string]interface{}{"observedGeneration": int64(1)},
			expected:   HealthNotReady,
		},
		{
			name:       "scaled to zero",
			generation: 1,
			spec:       map[string]interface{}{"replicas": int64(0)},
			status:     map[string]interface{}{"observedGeneration": int64(1)},
			expected:   HealthReady,
		},
		{
			name:       "no status yet",
			generation: 1,
			spec:       map[string]interface{}{"replicas": int64(1)},
			status:     nil,
			expected:   HealthNotReady,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resource := makeWorkload("Deployment", tc.generation, tc.spec, tc.status)
			assert.Equal(t, tc.expected, EvaluateHealth(resource))
		})
	}
}

func TestEvaluateHealth_StatefulSet(t *testing.T) {
	// ssStatus builds a StatefulSet status; revisions default to a settled "rev-1".
	ssStatus := func(observed, ready, updated int64, current, update string) map[string]interface{} {
		return map[string]interface{}{
			"observedGeneration": observed, "readyReplicas": ready, "updatedReplicas": updated,
			"currentRevision": current, "updateRevision": update,
		}
	}
	tests := []struct {
		name       string
		generation int64
		spec       map[string]interface{}
		status     map[string]interface{}
		expected   HealthStatus
	}{
		{
			name:       "3/3 ready and settled",
			generation: 1,
			spec:       map[string]interface{}{"replicas": int64(3)},
			status:     ssStatus(1, 3, 3, "rev-1", "rev-1"),
			expected:   HealthReady,
		},
		{
			name:       "0/1 ready",
			generation: 1,
			spec:       map[string]interface{}{"replicas": int64(1)},
			status:     ssStatus(1, 0, 1, "rev-1", "rev-1"),
			expected:   HealthNotReady,
		},
		{
			name:       "1/3 ready",
			generation: 1,
			spec:       map[string]interface{}{"replicas": int64(3)},
			status:     ssStatus(1, 1, 3, "rev-1", "rev-1"),
			expected:   HealthNotReady,
		},
		{
			name:       "rolling update in progress: updateRevision differs from currentRevision",
			generation: 2,
			spec:       map[string]interface{}{"replicas": int64(3), "updateStrategy": map[string]interface{}{"type": "RollingUpdate"}},
			status:     ssStatus(2, 3, 3, "rev-1", "rev-2"),
			expected:   HealthNotReady,
		},
		{
			name:       "rolling update in progress: not all pods updated",
			generation: 2,
			spec:       map[string]interface{}{"replicas": int64(3)},
			status:     ssStatus(2, 3, 1, "rev-2", "rev-2"),
			expected:   HealthNotReady,
		},
		{
			name:       "controller has not observed the latest generation",
			generation: 3,
			spec:       map[string]interface{}{"replicas": int64(1)},
			status:     ssStatus(2, 1, 1, "rev-1", "rev-1"),
			expected:   HealthNotReady,
		},
		{
			name:       "partitioned rollout: revisions differ by design, updated >= replicas-partition",
			generation: 2,
			spec: map[string]interface{}{
				"replicas": int64(3),
				"updateStrategy": map[string]interface{}{
					"type":          "RollingUpdate",
					"rollingUpdate": map[string]interface{}{"partition": int64(2)},
				},
			},
			status:   ssStatus(2, 3, 1, "rev-1", "rev-2"),
			expected: HealthReady,
		},
		{
			name:       "partitioned rollout: partitioned pods not yet updated",
			generation: 2,
			spec: map[string]interface{}{
				"replicas": int64(3),
				"updateStrategy": map[string]interface{}{
					"type":          "RollingUpdate",
					"rollingUpdate": map[string]interface{}{"partition": int64(2)},
				},
			},
			status:   ssStatus(2, 3, 0, "rev-1", "rev-2"),
			expected: HealthNotReady,
		},
		{
			name:       "OnDelete: revisions differ until pods are deleted, still healthy",
			generation: 2,
			spec: map[string]interface{}{
				"replicas":       int64(2),
				"updateStrategy": map[string]interface{}{"type": "OnDelete"},
			},
			status:   ssStatus(2, 2, 0, "rev-1", "rev-2"),
			expected: HealthReady,
		},
		{
			name:       "spec.replicas omitted defaults to 1, pod ready",
			generation: 1,
			spec:       map[string]interface{}{},
			status:     ssStatus(1, 1, 1, "rev-1", "rev-1"),
			expected:   HealthReady,
		},
		{
			name:       "spec.replicas omitted defaults to 1, pod not ready",
			generation: 1,
			spec:       map[string]interface{}{},
			status:     ssStatus(1, 0, 1, "rev-1", "rev-1"),
			expected:   HealthNotReady,
		},
		{
			name:       "scaled to zero",
			generation: 1,
			spec:       map[string]interface{}{"replicas": int64(0)},
			status:     ssStatus(1, 0, 0, "rev-1", "rev-1"),
			expected:   HealthReady,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resource := makeWorkload("StatefulSet", tc.generation, tc.spec, tc.status)
			assert.Equal(t, tc.expected, EvaluateHealth(resource))
		})
	}
}

func TestEvaluateHealth_DaemonSet(t *testing.T) {
	dsStatus := func(observed, desired, updated, available int64) map[string]interface{} {
		return map[string]interface{}{
			"observedGeneration": observed, "desiredNumberScheduled": desired,
			"updatedNumberScheduled": updated, "numberAvailable": available,
		}
	}
	tests := []struct {
		name       string
		generation int64
		status     map[string]interface{}
		expected   HealthStatus
	}{
		{name: "all nodes updated and available", generation: 1, status: dsStatus(1, 3, 3, 3), expected: HealthReady},
		{name: "no node matches the selector", generation: 1, status: dsStatus(1, 0, 0, 0), expected: HealthReady},
		{name: "rollout in progress: not all nodes updated", generation: 2, status: dsStatus(2, 3, 1, 3), expected: HealthNotReady},
		{name: "updated pods not yet available", generation: 1, status: dsStatus(1, 3, 3, 2), expected: HealthNotReady},
		{name: "controller has not observed the latest generation", generation: 2, status: dsStatus(1, 3, 3, 3), expected: HealthNotReady},
		{name: "no status yet", generation: 1, status: nil, expected: HealthNotReady},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resource := makeWorkload("DaemonSet", tc.generation, nil, tc.status)
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
