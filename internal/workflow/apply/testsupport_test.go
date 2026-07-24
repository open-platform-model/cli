package apply

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
)

// crdGVR is the CustomResourceDefinition GroupVersionResource the cluster gates
// probe. Mirrors the unexported constant in the inventory package.
var crdGVR = schema.GroupVersionResource{
	Group:    "apiextensions.k8s.io",
	Version:  "v1",
	Resource: "customresourcedefinitions",
}

// probeRecorder captures the ordered list of dynamic-client GET probes the
// cluster gates issue, identified by resource name.
type probeRecorder struct {
	gets []string
}

// recordingDynamicClient builds a *kubernetes.Client backed by a fake dynamic
// client seeded with the given objects, plus a probeRecorder that appends every
// GET's resource name (in order) as the gates run. The recording reactor passes
// through so the seeded objects are still served.
func recordingDynamicClient(objs ...*unstructured.Unstructured) (*kubernetes.Client, *probeRecorder) {
	scheme := runtime.NewScheme()
	runtimeObjs := make([]runtime.Object, len(objs))
	for i, o := range objs {
		runtimeObjs[i] = o
	}
	listKinds := map[schema.GroupVersionResource]string{
		inventory.ModuleInstanceGVR: "ModuleInstanceList",
		inventory.PlatformGVR:       "PlatformList",
		crdGVR:                      "CustomResourceDefinitionList",
	}
	fake := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds, runtimeObjs...)

	rec := &probeRecorder{}
	fake.PrependReactor("get", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		rec.gets = append(rec.gets, action.GetResource().Resource)
		return false, nil, nil // passthrough to the tracker
	})

	return &kubernetes.Client{Dynamic: fake}, rec
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
		"metadata":   map[string]any{"name": inventory.CRDNameModuleInstances},
		"spec": map[string]any{
			"group": inventory.GroupOpmodel,
			"versions": []any{
				map[string]any{
					"name":    inventory.VersionV1Alpha1,
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
		"apiVersion": inventory.APIVersionModuleInstance,
		"kind":       inventory.KindPlatform,
		"metadata":   map[string]any{"name": inventory.PlatformSingletonName},
		"spec":       map[string]any{"type": "kubernetes"},
		"status":     status,
	}}
}
