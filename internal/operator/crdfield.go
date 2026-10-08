package operator

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/inventory"
)

// FieldSupport says whether the ModuleInstance CRD installed in a cluster
// has a field of spec.
type FieldSupport int

const (
	// FieldUnknown means the CRD's schema could not be read, so nothing is
	// known.
	FieldUnknown FieldSupport = iota
	// FieldAbsent means the CRD's schema has no such field: the API it
	// defines does not carry the field, and the operator that came with those
	// CRDs does not know it.
	FieldAbsent
	// FieldPresent means the CRD's schema has the field. It says nothing
	// certain about the controller: CRDs can be newer than the operator that
	// runs beside them, or older.
	FieldPresent
)

// SpecFieldSupport reports whether a live ModuleInstance CRD, as
// ReadyModuleInstanceCRD returns it, has spec.<field>. The CRD says which
// fields the operator's API has without any release number to compare. The
// schema is read by inventory.ModuleInstanceCRDHasField, the reading the apply
// gate uses, so the version rule is the same everywhere. A nil CRD, or one
// whose schema cannot be read, is FieldUnknown.
func SpecFieldSupport(crd *unstructured.Unstructured, field string) FieldSupport {
	if crd == nil {
		return FieldUnknown
	}
	has, err := inventory.ModuleInstanceCRDHasField(crd, "spec", field)
	switch {
	case err != nil:
		return FieldUnknown
	case has:
		return FieldPresent
	default:
		return FieldAbsent
	}
}
