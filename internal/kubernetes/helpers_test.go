package kubernetes

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// makeResource builds a v1 object of the given kind with no status.
func makeResource(kind string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       kind,
			"metadata": map[string]interface{}{
				"name":      "test-resource",
				"namespace": "default",
			},
		},
	}
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
