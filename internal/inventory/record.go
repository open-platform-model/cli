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

	// SpecValues is the CR's spec.values block — the unified values the last
	// apply consumed. The handoff verification render replays them against the
	// registry-resolved module (enhancement 0006 D7.4/D38).
	SpecValues map[string]any

	// Prune is the CR's spec.prune marker, which governs whether the operator
	// deletes an instance's workloads when the CR is removed. It has no CRD
	// default, so it is false unless someone set it — and the operator then
	// deliberately orphans the workloads ("Prune disabled, orphaning managed
	// resources on deletion"). The CLI does not write this field; it reads it
	// so an operator-owned delete can report what will actually happen.
	Prune bool

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
