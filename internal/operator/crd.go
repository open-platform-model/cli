package operator

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// PlatformCRDHasSubscriptionVersion reports whether the embedded operator
// manifest's Platform CRD declares the scalar `version` property on registry
// subscriptions. While it does not, the API server's structural schema would
// silently prune a written `version`, so the CLI refuses to seed the cluster
// Platform (see platform.EnsureClusterPlatform).
//
// Transitional: this probe and its caller are deleted once the embedded
// manifest carries the v2 CRD from the operator-library-retarget release.
func PlatformCRDHasSubscriptionVersion() bool {
	objs, err := EmbeddedManifest()
	if err != nil {
		// An unparseable embedded manifest cannot prove support; refuse.
		return false
	}
	for _, obj := range objs {
		if obj.GetKind() != kindCustomResourceDefinition {
			continue
		}
		kind, _, err := unstructured.NestedString(obj.Object, "spec", "names", "kind")
		if err != nil || kind != "Platform" {
			continue
		}
		versions, _, err := unstructured.NestedSlice(obj.Object, "spec", "versions")
		if err != nil {
			continue
		}
		for _, v := range versions {
			vm, ok := v.(map[string]any)
			if !ok {
				continue
			}
			_, found, err := unstructured.NestedMap(vm,
				"schema", "openAPIV3Schema", "properties", "spec",
				"properties", "registry", "additionalProperties",
				"properties", "version")
			if err == nil && found {
				return true
			}
		}
	}
	return false
}
