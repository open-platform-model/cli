package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestSortByWeightDescending(t *testing.T) {
	// Create resources of different kinds
	resources := []*unstructured.Unstructured{
		makeUnstructured("apps/v1", "Deployment", "my-deploy", "default"),
		makeUnstructured("v1", "Namespace", "my-ns", ""),
		makeUnstructured("admissionregistration.k8s.io/v1", "ValidatingWebhookConfiguration", "my-webhook", ""),
		makeUnstructured("v1", "ConfigMap", "my-cm", "default"),
		makeUnstructured("v1", "Service", "my-svc", "default"),
	}

	sortByWeightDescending(resources)

	// Expected order: Webhook(500) > Deployment(100) > Service(50) > ConfigMap(15) > Namespace(0)
	assert.Equal(t, "ValidatingWebhookConfiguration", resources[0].GetKind())
	assert.Equal(t, "Deployment", resources[1].GetKind())
	assert.Equal(t, "Service", resources[2].GetKind())
	assert.Equal(t, "ConfigMap", resources[3].GetKind())
	assert.Equal(t, "Namespace", resources[4].GetKind())
}

func makeUnstructured(apiVersion, kind, name, namespace string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(apiVersion)
	obj.SetKind(kind)
	obj.SetName(name)
	if namespace != "" {
		obj.SetNamespace(namespace)
	}
	return obj
}
