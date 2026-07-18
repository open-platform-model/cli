# Delta: inventory-ownership (cli-cr-inventory-backend)

Ownership derives from the CR's `spec.owner` marker instead of the record's `createdBy` provenance.

## ADDED Requirements

### Requirement: spec.owner defines instance ownership

Instance ownership SHALL be determined from the `ModuleInstance` CR's `spec.owner` field. A CR the CLI writes SHALL always carry `spec.owner: cli` explicitly. For reads: an absent CR or `spec.owner: cli` SHALL resolve to CLI-executor mode; `spec.owner: operator` SHALL resolve to operator-owned. (An absent or empty `spec.owner` on an existing CR means operator-managed by the operator's own defaulting contract; the CLI SHALL treat any value other than `cli` as operator-owned.)

#### Scenario: CLI-owned instance

- **WHEN** the CLI reads a `ModuleInstance` with `spec.owner: cli`
- **THEN** the instance SHALL resolve to CLI-executor mode

#### Scenario: Operator-owned instance

- **WHEN** the CLI reads a `ModuleInstance` with `spec.owner: operator` or without `spec.owner`
- **THEN** the instance SHALL resolve to operator-owned

### Requirement: Operator-owned instances are refused with actionable errors

In this slice, `opm instance apply` against an operator-owned instance SHALL refuse with an error stating the instance is operator-managed. `opm instance delete` against an operator-owned instance SHALL refuse and hint that `kubectl delete moduleinstance <name>` triggers the operator's finalizer cleanup. Refusals SHALL occur before any resource is applied, pruned, or deleted.

#### Scenario: Apply refused on operator-owned instance

- **WHEN** `opm instance apply` targets a `ModuleInstance` with `spec.owner: operator`
- **THEN** the command SHALL exit non-zero with an operator-managed error
- **AND** no resource SHALL have been applied

#### Scenario: Delete refused with kubectl hint

- **WHEN** `opm instance delete` targets an operator-owned instance
- **THEN** the command SHALL exit non-zero and the error SHALL mention `kubectl delete moduleinstance`

### Requirement: Ownership mode resolution is a single branch point

The mapping from `spec.owner` to a CLI execution mode SHALL be implemented in exactly one function that all mutating commands (`apply`, `delete`) consume, so a later slice can extend operator-owned handling (thin spec-editor mode, enhancement 0006 D18) without changing callers.

#### Scenario: Apply and delete share the resolver

- **WHEN** `apply` and `delete` evaluate ownership for the same CR
- **THEN** both SHALL obtain their mode from the same resolution function

## MODIFIED Requirements

### Requirement: Ownership is exclusive across tools

The CLI and the operator SHALL treat ownership as exclusive, coordinated through the `ModuleInstance` CR's `spec.owner` field. The CLI MUST NOT act as the resource reconciler (apply, prune, inventory write, status write) for an instance whose `spec.owner` is not `cli`. The operator skips render/apply/prune for `spec.owner: cli` instances (operator-side contract, enhancement 0006 D3/A4).

#### Scenario: CLI sees operator-owned instance

- **WHEN** the CLI loads a `ModuleInstance` whose `spec.owner` is `operator`
- **THEN** the CLI SHALL treat the instance as not mutable by the CLI's direct-resource path

#### Scenario: Operator sees CLI-owned instance

- **WHEN** the operator reconciles a `ModuleInstance` with `spec.owner: cli`
- **THEN** the operator SHALL skip render/apply/prune and never touch CLI-written status fields

## REMOVED Requirements

### Requirement: Inventory provenance defines release ownership

**Reason**: The `createdBy` record field is replaced by the CR's `spec.owner` marker (enhancement 0006 D3); ownership is user-intent spec state, not status provenance.
**Migration**: Migrated CRs are written with `spec.owner: cli`. The `pkg/ownership` `createdBy` constants and record-based enforcement are deleted with the Secret record.

### Requirement: Legacy inventories are treated as CLI-owned

**Reason**: Legacy (pre-provenance) Secret inventories are no longer read outside the one-time migration path, which ports them as CLI-owned.
**Migration**: The migration writes `spec.owner: cli` for every ported Secret, preserving the legacy assumption.
