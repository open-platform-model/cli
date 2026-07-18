package inventory

import (
	pkginventory "github.com/open-platform-model/cli/pkg/inventory"
)

// Record is the CLI's view of an instance's persisted inventory, backed by the
// ModuleInstance CR. It replaces the Secret-era InstanceInventoryRecord: the
// instance identity lives in metadata, module identity in spec.module, the UUID
// in status.instanceUUID, and ownership in spec.owner.
type Record struct {
	// Name and Namespace are the CR's metadata identity.
	Name      string
	Namespace string

	// Owner is the CR's spec.owner marker ("cli", "operator", or empty). An
	// empty value on an existing CR means operator-managed by the operator's
	// defaulting contract; see ResolveOwnership.
	Owner string

	// ModulePath and ModuleVersion are the CR's spec.module reference.
	ModulePath    string
	ModuleVersion string

	// InstanceUUID is the CR's status.instanceUUID.
	InstanceUUID string

	// Inventory is the CR's status.inventory block.
	Inventory pkginventory.Inventory

	// LastApplied* mirror the CLI-owned status digest set.
	LastAppliedRenderDigest string
	LastAppliedSourceDigest string
	LastAppliedConfigDigest string
	LastAppliedAt           string

	// SourceLocal reflects the render-provenance annotation
	// (module-instance.opmodel.dev/source: local) on the CR.
	SourceLocal bool
}
