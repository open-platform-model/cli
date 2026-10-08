package operator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	fakedynamic "k8s.io/client-go/dynamic/fake"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"
)

// establishedCRD is an Established operator CRD; with specFields it carries
// one storage version whose spec has those fields, without them no version.
func establishedCRD(name string, withSchema bool, specFields ...string) *unstructured.Unstructured {
	crd := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apiextensions.k8s.io/v1", "kind": "CustomResourceDefinition",
		"metadata": map[string]any{"name": name},
		"status":   map[string]any{"conditions": []any{map[string]any{"type": "Established", "status": "True"}}},
	}}
	if !withSchema {
		return crd
	}
	props := map[string]any{}
	for _, f := range specFields {
		props[f] = map[string]any{"type": "string"}
	}
	crd.Object["spec"] = map[string]any{"versions": []any{map[string]any{
		"name": "v1alpha1", "served": true, "storage": true,
		"schema": map[string]any{"openAPIV3Schema": map[string]any{
			"properties": map[string]any{"spec": map[string]any{"properties": props}},
		}},
	}}}
	return crd
}

// The CRD's schema says whether the operator's API has a field, with no
// operator version to compare. A CRD whose schema cannot be read, and no CRD
// at all, say nothing.
func TestSpecFieldSupport(t *testing.T) {
	name := inventory.CRDNameModuleInstances
	tests := []struct {
		name string
		crd  *unstructured.Unstructured
		want FieldSupport
	}{
		{"field in the storage version", establishedCRD(name, true, "prune", "dataPolicy"), FieldPresent},
		{"field missing", establishedCRD(name, true, "prune"), FieldAbsent},
		{"no version and no schema", establishedCRD(name, false), FieldUnknown},
		{"no CRD", nil, FieldUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, SpecFieldSupport(tt.crd, "dataPolicy"))
		})
	}
}

func readyCluster(moduleInstances *unstructured.Unstructured) *kubernetes.Client {
	objs := []runtime.Object{moduleInstances}
	for _, name := range CRDNames() {
		if name != inventory.CRDNameModuleInstances {
			objs = append(objs, establishedCRD(name, false))
		}
	}
	objs = append(objs, &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apps/v1", "kind": "Deployment",
		"metadata": map[string]any{"name": ControllerDeploymentName, "namespace": OperatorNamespace, "generation": int64(1)},
		"spec":     map[string]any{"replicas": int64(1)},
		"status": map[string]any{
			"observedGeneration": int64(1), "replicas": int64(1), "updatedReplicas": int64(1),
			"readyReplicas": int64(1), "availableReplicas": int64(1),
			"conditions": []any{map[string]any{"type": "Available", "status": "True"}},
		},
	}})
	return &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: fakedynamic.NewSimpleDynamicClient(runtime.NewScheme(), objs...)}
}

// The gate hands back the ModuleInstance CRD it read, so a caller asks about
// the operator's API without a second read; it refuses as CheckReady does.
func TestReadyModuleInstanceCRD(t *testing.T) {
	client := readyCluster(establishedCRD(inventory.CRDNameModuleInstances, true, "dataPolicy"))
	crd, err := ReadyModuleInstanceCRD(context.Background(), client)
	require.NoError(t, err)
	require.NotNil(t, crd)
	assert.Equal(t, inventory.CRDNameModuleInstances, crd.GetName())
	assert.Equal(t, FieldPresent, SpecFieldSupport(crd, "dataPolicy"))
	require.NoError(t, CheckReady(context.Background(), client))

	empty := &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: fakedynamic.NewSimpleDynamicClient(runtime.NewScheme())}
	crd, err = ReadyModuleInstanceCRD(context.Background(), empty)
	var notReady *NotReadyError
	require.ErrorAs(t, err, &notReady)
	assert.Nil(t, crd)
	assert.Equal(t, err.Error(), CheckReady(context.Background(), empty).Error())
}
