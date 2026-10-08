package operator

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"
)

// moduleInstanceCRD is the ModuleInstance CRD with one version whose spec has
// the given fields.
func moduleInstanceCRD(served bool, specFields ...string) *unstructured.Unstructured {
	props := map[string]any{}
	for _, f := range specFields {
		props[f] = map[string]any{"type": "string"}
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apiextensions.k8s.io/v1", "kind": "CustomResourceDefinition",
		"metadata": map[string]any{"name": inventory.CRDNameModuleInstances},
		"spec": map[string]any{"versions": []any{map[string]any{
			"name": "v1alpha1", "served": served,
			"schema": map[string]any{"openAPIV3Schema": map[string]any{
				"properties": map[string]any{"spec": map[string]any{"properties": props}},
			}},
		}}},
	}}
}

// The CRD's schema says whether the operator's API has a field, with no
// operator version to compare. A CRD that cannot be read says nothing.
func TestModuleInstanceSpecField(t *testing.T) {
	tests := []struct {
		name string
		objs []runtime.Object
		want FieldSupport
	}{
		{"field in a served version", []runtime.Object{moduleInstanceCRD(true, "prune", "dataPolicy")}, FieldPresent},
		{"field missing", []runtime.Object{moduleInstanceCRD(true, "prune")}, FieldAbsent},
		{"field only in a version that is not served", []runtime.Object{moduleInstanceCRD(false, "dataPolicy")}, FieldAbsent},
		{"no CRD", nil, FieldUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: fakedynamic.NewSimpleDynamicClient(runtime.NewScheme(), tt.objs...)}
			assert.Equal(t, tt.want, ModuleInstanceSpecField(context.Background(), client, "dataPolicy"))
		})
	}
}

func TestModuleInstanceSpecField_FailedReadIsUnknown(t *testing.T) {
	fake := fakedynamic.NewSimpleDynamicClient(runtime.NewScheme(), moduleInstanceCRD(true, "dataPolicy"))
	fake.PrependReactor("get", "customresourcedefinitions", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("forbidden")
	})
	client := &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: fake}
	assert.Equal(t, FieldUnknown, ModuleInstanceSpecField(context.Background(), client, "dataPolicy"))
}
