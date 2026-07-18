package kubernetes

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
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

func TestDelete_DeletesOnlyTrackedInventoryResources(t *testing.T) {
	ctx := context.Background()
	namespace := "default"

	tracked := makeUnstructured("v1", "ConfigMap", "tracked", namespace)
	untracked := makeUnstructured("v1", "ConfigMap", "untracked", namespace)

	scheme := runtime.NewScheme()
	client := &Client{
		Clientset: fake.NewClientset(tracked.DeepCopy(), untracked.DeepCopy()),
		Dynamic:   dynamicfake.NewSimpleDynamicClient(scheme, tracked.DeepCopy(), untracked.DeepCopy()),
	}

	// The ModuleInstance CR is deleted last by the caller, not by Delete; here
	// we only assert Delete removes the tracked workload and leaves untracked
	// resources alone.
	result, err := Delete(ctx, client, DeleteOptions{
		InstanceName:          "demo",
		Namespace:             namespace,
		InventoryLive:         []*unstructured.Unstructured{tracked.DeepCopy()},
		InventoryRecordExists: true,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.Deleted)

	_, err = client.ResourceClient(GVRFromUnstructured(tracked), namespace).Get(ctx, tracked.GetName(), metav1.GetOptions{})
	assert.Error(t, err)

	remaining, err := client.ResourceClient(GVRFromUnstructured(untracked), namespace).Get(ctx, untracked.GetName(), metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, untracked.GetName(), remaining.GetName())
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
