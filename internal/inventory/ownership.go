package inventory

// OwnershipMode is the CLI's execution mode for an instance, derived from the
// ModuleInstance CR's spec.owner marker. It is the single branch point every
// mutating command (apply, delete) consumes.
type OwnershipMode int

const (
	// ModeCLIExecutor is the CLI's direct-resource executor mode: it renders,
	// applies, prunes, and writes inventory/status. Resolved for an absent CR
	// or an explicit spec.owner: cli.
	ModeCLIExecutor OwnershipMode = iota
	// ModeControllerOwned means the controller reconciles the instance. Any owner
	// value other than "cli" on an existing CR resolves here (including an
	// empty owner, which is controller-managed by the controller's defaulting
	// contract). In this mode the CLI edits the spec and observes, but writes
	// no resources of its own: apply becomes a thin spec editor and delete
	// delegates to the controller's cleanup finalizer (0006:D18).
	ModeControllerOwned
)

// ResolveOwnership maps a record's spec.owner to the CLI execution mode. A nil
// record (no CR exists) is CLI-executor mode — a first apply. An existing CR is
// CLI-executor only when it explicitly carries spec.owner: cli; every other
// value is controller-owned.
func ResolveOwnership(rec *Record) OwnershipMode {
	if rec == nil {
		return ModeCLIExecutor
	}
	if rec.Owner == OwnerCLI {
		return ModeCLIExecutor
	}
	return ModeControllerOwned
}

// DisplayOwner returns a human-facing owner label for a record's spec.owner. An
// empty owner on an existing CR is controller-managed by the controller's
// defaulting contract.
func DisplayOwner(owner string) string {
	if owner == "" {
		return OwnerController
	}
	return owner
}
