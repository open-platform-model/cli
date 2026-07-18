package inventory

import (
	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/cli/internal/kubernetes"
)

// newDynamicClient builds a *kubernetes.Client backed by a fake dynamic client
// seeded with the given unstructured objects. ModuleInstance list kind is
// registered so empty-fixture list calls do not panic.
func newDynamicClient(objs ...*unstructured.Unstructured) *kubernetes.Client {
	scheme := runtime.NewScheme()
	runtimeObjs := make([]runtime.Object, len(objs))
	for i, o := range objs {
		runtimeObjs[i] = o
	}
	listKinds := map[schema.GroupVersionResource]string{
		ModuleInstanceGVR: "ModuleInstanceList",
		PlatformGVR:       "PlatformList",
		crdGVR:            "CustomResourceDefinitionList",
	}
	return &kubernetes.Client{
		Dynamic: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds, runtimeObjs...),
	}
}

// withSSAR attaches a fake clientset whose SelfSubjectAccessReview always
// returns the given allowed verdict.
func withSSAR(client *kubernetes.Client, allowed bool) *kubernetes.Client {
	cs := k8sfake.NewClientset()
	cs.PrependReactor("create", "selfsubjectaccessreviews", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, &authorizationv1.SelfSubjectAccessReview{
			Status: authorizationv1.SubjectAccessReviewStatus{Allowed: allowed, Reason: "test"},
		}, nil
	})
	client.Clientset = cs
	return client
}

// makeModuleInstanceCRD builds a ModuleInstance CRD object with configurable
// presence of the spec.owner and status.inventory schema properties.
func makeModuleInstanceCRD(hasOwner, hasInventory bool) *unstructured.Unstructured {
	specProps := map[string]any{"module": map[string]any{"type": "object"}}
	if hasOwner {
		specProps["owner"] = map[string]any{"type": "string"}
	}
	statusProps := map[string]any{}
	if hasInventory {
		statusProps["inventory"] = map[string]any{"type": "object"}
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apiextensions.k8s.io/v1",
		"kind":       "CustomResourceDefinition",
		"metadata":   map[string]any{"name": CRDNameModuleInstances},
		"spec": map[string]any{
			"group": GroupOpmodel,
			"versions": []any{
				map[string]any{
					"name":    VersionV1Alpha1,
					"served":  true,
					"storage": true,
					"schema": map[string]any{
						"openAPIV3Schema": map[string]any{
							"properties": map[string]any{
								"spec":   map[string]any{"properties": specProps},
								"status": map[string]any{"properties": statusProps},
							},
						},
					},
				},
			},
		},
	}}
}

// makePlatform builds the cluster Platform singleton with an optional
// status.operatorVersion.
func makePlatform(operatorVersion string) *unstructured.Unstructured {
	status := map[string]any{}
	if operatorVersion != "" {
		status["operatorVersion"] = operatorVersion
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": APIVersionModuleInstance,
		"kind":       KindPlatform,
		"metadata":   map[string]any{"name": PlatformSingletonName},
		"spec":       map[string]any{"type": "kubernetes"},
		"status":     status,
	}}
}
