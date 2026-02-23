package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// makeWideTestResource creates an unstructured resource for wide-info tests.
func makeWideTestResource(kind string, obj map[string]interface{}) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetKind(kind)
	u.Object = obj
	u.Object["kind"] = kind
	return u
}

func TestExtractWideInfo_Deployment(t *testing.T) {
	tests := []struct {
		name             string
		obj              map[string]interface{}
		expectedReplicas string
		expectedImage    string
	}{
		{
			name: "ready replicas",
			obj: map[string]interface{}{
				"spec":   map[string]interface{}{"replicas": int64(3), "template": map[string]interface{}{"spec": map[string]interface{}{"containers": []interface{}{map[string]interface{}{"image": "nginx:1.25"}}}}},
				"status": map[string]interface{}{"readyReplicas": int64(3)},
			},
			expectedReplicas: "3/3",
			expectedImage:    "nginx:1.25",
		},
		{
			name: "zero replicas",
			obj: map[string]interface{}{
				"spec":   map[string]interface{}{"replicas": int64(0), "template": map[string]interface{}{"spec": map[string]interface{}{"containers": []interface{}{}}}},
				"status": map[string]interface{}{"readyReplicas": int64(0)},
			},
			expectedReplicas: "",
			expectedImage:    "",
		},
		{
			name: "missing status",
			obj: map[string]interface{}{
				"spec": map[string]interface{}{"replicas": int64(2)},
			},
			expectedReplicas: "0/2",
			expectedImage:    "",
		},
		{
			name: "empty containers list",
			obj: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": int64(1),
					"template": map[string]interface{}{"spec": map[string]interface{}{"containers": []interface{}{}}},
				},
				"status": map[string]interface{}{"readyReplicas": int64(1)},
			},
			expectedReplicas: "1/1",
			expectedImage:    "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := makeWideTestResource("Deployment", tc.obj)
			wi := extractWideInfo(u)
			require.NotNil(t, wi)
			assert.Equal(t, tc.expectedReplicas, wi.Replicas)
			assert.Equal(t, tc.expectedImage, wi.Image)
		})
	}
}

func TestExtractWideInfo_StatefulSet(t *testing.T) {
	obj := map[string]interface{}{
		"spec":   map[string]interface{}{"replicas": int64(2), "template": map[string]interface{}{"spec": map[string]interface{}{"containers": []interface{}{map[string]interface{}{"image": "redis:7"}}}}},
		"status": map[string]interface{}{"readyReplicas": int64(1)},
	}
	u := makeWideTestResource("StatefulSet", obj)
	wi := extractWideInfo(u)
	require.NotNil(t, wi)
	assert.Equal(t, "1/2", wi.Replicas)
	assert.Equal(t, "redis:7", wi.Image)
}

func TestExtractWideInfo_DaemonSet(t *testing.T) {
	obj := map[string]interface{}{
		"spec": map[string]interface{}{"template": map[string]interface{}{"spec": map[string]interface{}{"containers": []interface{}{map[string]interface{}{"image": "fluentd:v1"}}}}},
		"status": map[string]interface{}{
			"numberReady":            int64(3),
			"desiredNumberScheduled": int64(5),
		},
	}
	u := makeWideTestResource("DaemonSet", obj)
	wi := extractWideInfo(u)
	require.NotNil(t, wi)
	assert.Equal(t, "3/5", wi.Replicas)
	assert.Equal(t, "fluentd:v1", wi.Image)
}

func TestExtractWideInfo_PVC(t *testing.T) {
	tests := []struct {
		name     string
		obj      map[string]interface{}
		expected string
	}{
		{
			name:     "with storage and phase",
			obj:      map[string]interface{}{"status": map[string]interface{}{"capacity": map[string]interface{}{"storage": "10Gi"}, "phase": "Bound"}},
			expected: "10Gi (Bound)",
		},
		{
			name:     "pending no capacity",
			obj:      map[string]interface{}{"status": map[string]interface{}{"phase": "Pending"}},
			expected: "Pending",
		},
		{
			name:     "missing fields",
			obj:      map[string]interface{}{"status": map[string]interface{}{}},
			expected: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := makeWideTestResource("PersistentVolumeClaim", tc.obj)
			wi := extractWideInfo(u)
			require.NotNil(t, wi)
			assert.Equal(t, tc.expected, wi.Replicas)
		})
	}
}

func TestExtractWideInfo_Ingress(t *testing.T) {
	tests := []struct {
		name         string
		obj          map[string]interface{}
		expectedHost string
	}{
		{
			name:         "with host",
			obj:          map[string]interface{}{"spec": map[string]interface{}{"rules": []interface{}{map[string]interface{}{"host": "app.example.com"}}}},
			expectedHost: "app.example.com",
		},
		{
			name:         "no rules",
			obj:          map[string]interface{}{"spec": map[string]interface{}{"rules": []interface{}{}}},
			expectedHost: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := makeWideTestResource("Ingress", tc.obj)
			wi := extractWideInfo(u)
			require.NotNil(t, wi)
			assert.Equal(t, tc.expectedHost, wi.Image)
		})
	}
}

func TestExtractWideInfo_Other(t *testing.T) {
	u := makeWideTestResource("ConfigMap", map[string]interface{}{})
	wi := extractWideInfo(u)
	assert.Nil(t, wi)
}

func TestExtractWideInfo_Nil(t *testing.T) {
	wi := extractWideInfo(nil)
	assert.Nil(t, wi)
}
