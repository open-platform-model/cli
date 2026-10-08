package inventory

import (
	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
)

// Inventory is the status.inventory block the CLI records on a
// ModuleInstance: the current set of objects the instance owns, with the
// summary fields beside it. It carries no struct tags; wire.go maps it to the
// CRD's field names.
type Inventory struct {
	// Revision increments on every successful apply.
	Revision int
	// Digest is the library's inventory digest of Entries
	// (k8sinventory.Digest).
	Digest string
	// Count is len(Entries).
	Count int
	// Entries are the owned objects.
	Entries []k8sinventory.Entry
}

// Record is the CLI's view of an instance's persisted inventory, backed by the
// ModuleInstance CR: the
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

	// SpecValues is the CR's spec.values block — the unified values the last
	// apply consumed, recorded so a future ownership transfer can replay them
	// against the registry-resolved module (0006:D38).
	SpecValues map[string]any

	// Prune is the CR's spec.prune marker, which governs whether the operator
	// deletes an instance's workloads when the CR is removed. It has no CRD
	// default, so it is false unless someone set it — and the operator then
	// deliberately orphans the workloads ("Prune disabled, orphaning managed
	// resources on deletion"). The CLI does not write this field; it reads it
	// so an operator-owned delete can report what will actually happen.
	Prune bool

	// DataPolicy is the CR's spec.dataPolicy as it is written, empty when the
	// field is absent. The operator reads it to decide whether a prune or a
	// deletion removes PersistentVolumeClaims: only "Delete" removes them. The
	// CLI does not write this field and does not interpret it here; it reads
	// it so an operator-owned delete can report what will actually happen.
	DataPolicy string

	// InstanceUUID is the CR's status.instanceUUID.
	InstanceUUID string

	// Inventory is the CR's status.inventory block.
	Inventory Inventory

	// LastApplied* mirror the CLI-owned status digest set.
	LastAppliedRenderDigest string
	LastAppliedSourceDigest string
	LastAppliedConfigDigest string
	LastAppliedAt           string

	// SourceLocal reflects the render-provenance annotation
	// (module-instance.opmodel.dev/source: local) on the CR.
	SourceLocal bool

	// Generation is the CR's metadata.generation — the spec revision the API
	// server assigned. Compared against ObservedGeneration to tell whether the
	// operator has caught up with the latest write.
	Generation int64

	// ObservedGeneration is the CR's status.observedGeneration: the generation
	// the operator last reconciled. Operator-written; the CLI only reads it.
	ObservedGeneration int64

	// Conditions is the CR's status.conditions block, operator-written. The
	// CLI reads it to report reconcile outcomes (see ReadyFor).
	Conditions []Condition
}
