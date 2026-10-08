package operator

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
)

// FieldSupport says whether the ModuleInstance CRD installed in a cluster
// has a field of spec.
type FieldSupport int

const (
	// FieldUnknown means the CRD could not be read, so nothing is known.
	FieldUnknown FieldSupport = iota
	// FieldAbsent means the CRD was read and no version it serves has the
	// field: the API server drops the field from every ModuleInstance, and
	// the operator that came with those CRDs does not know it.
	FieldAbsent
	// FieldPresent means a served version of the CRD has the field. It says
	// nothing certain about the controller: CRDs can be newer than the
	// operator that runs beside them.
	FieldPresent
)

// ModuleInstanceSpecField reads the ModuleInstance CRD, the same object
// CheckReady reads, and reports whether a version it serves has spec.<field>.
// A release of the operator is not named by any object a namespaced command
// reads, but its CRDs say which fields its API has, with no version number to
// compare. Any failed read is FieldUnknown.
func ModuleInstanceSpecField(ctx context.Context, client *kubernetes.Client, field string) FieldSupport {
	crd := &unstructured.Unstructured{}
	crd.SetAPIVersion("apiextensions.k8s.io/v1")
	crd.SetKind(kindCustomResourceDefinition)
	crd.SetName(inventory.CRDNameModuleInstances)

	live, err := getObject(ctx, client, crd)
	if err != nil {
		return FieldUnknown
	}
	//nolint:errcheck // a CRD without readable versions has no field to offer
	versions, _, _ := unstructured.NestedSlice(live.Object, "spec", "versions")
	for _, v := range versions {
		version, ok := v.(map[string]any)
		if !ok {
			continue
		}
		//nolint:errcheck // an absent or wrong-typed served reads as not served
		if served, _, _ := unstructured.NestedBool(version, "served"); !served {
			continue
		}
		//nolint:errcheck // an absent or wrong-typed schema has no such field
		if _, found, _ := unstructured.NestedMap(version, "schema", "openAPIV3Schema", "properties", "spec", "properties", field); found {
			return FieldPresent
		}
	}
	return FieldAbsent
}
