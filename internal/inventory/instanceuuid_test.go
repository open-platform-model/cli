package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func labeledResource(labels map[string]string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]any{"name": "x"},
	}}
	if labels != nil {
		obj.SetLabels(labels)
	}
	return obj
}

func TestExtractInstanceUUID_FirstNonEmpty(t *testing.T) {
	resources := []*unstructured.Unstructured{
		labeledResource(nil),
		labeledResource(map[string]string{LabelInstanceUUID: "7c9e6679-7425-40de-944b-e07fc1f90ae7"}),
		labeledResource(map[string]string{LabelInstanceUUID: "other"}),
	}
	assert.Equal(t, "7c9e6679-7425-40de-944b-e07fc1f90ae7", ExtractInstanceUUID(resources))
}

func TestExtractInstanceUUID_AbsentReturnsEmpty(t *testing.T) {
	resources := []*unstructured.Unstructured{labeledResource(nil), labeledResource(map[string]string{"other": "x"})}
	assert.Equal(t, "", ExtractInstanceUUID(resources))
}

func TestExtractInstanceUUID_NilSafe(t *testing.T) {
	assert.Equal(t, "", ExtractInstanceUUID(nil))
	assert.Equal(t, "", ExtractInstanceUUID([]*unstructured.Unstructured{nil}))
}
