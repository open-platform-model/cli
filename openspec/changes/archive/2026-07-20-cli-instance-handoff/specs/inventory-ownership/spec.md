# Delta: inventory-ownership (cli-instance-handoff)

The C1 refusal arm for operator-owned instances is replaced by D18's dual-mode contract: thin-editor `apply`, finalizer-delegating `delete`.

## ADDED Requirements

### Requirement: Thin-editor apply on operator-owned instances

`opm instance apply` against an operator-owned instance SHALL act as a spec editor only: unify the value inputs, server-side-apply the CLI-owned spec with `spec.module` and `spec.values` updated (field manager `opm-cli`), then wait, bounded, for the operator's reconcile and report its resulting status. In this mode the CLI MUST NOT render-and-apply resources, prune, write `status`, or run the status-RBAC pre-flight. The version-skew ceiling gate still applies. A module resolving from local bytes (local directory, or a local-path `replaceWith` in effect) SHALL be refused in this mode — the operator cannot fetch a local checkout.

The applied document SHALL carry the instance's current `spec.owner` value, read from the live CR, rather than omitting the field. Omitting it does not leave ownership untouched: it releases this field manager's claim, and the API server prunes the operator's ownership marker. The CLI SHALL NOT change the owner's value in this mode.

#### Scenario: The operator's ownership survives a values edit

- **WHEN** a thin-editor apply updates `spec.values` on an instance with `spec.owner: operator`
- **THEN** `spec.owner` SHALL still be `operator` afterwards
- **AND** `spec.module` SHALL be unchanged unless the apply itself changed it

#### Scenario: Values update on an operator-owned instance

- **WHEN** `opm instance apply` targets an instance with `spec.owner: operator` and new values
- **THEN** the CLI SHALL patch `spec.values`, wait for the operator's reconcile, and report the operator's `Ready` outcome
- **AND** SHALL NOT apply or prune any resource itself

#### Scenario: Local module refused in thin-editor mode

- **WHEN** the apply's module would resolve from a local directory or local replacement
- **AND** the target instance is operator-owned
- **THEN** the CLI SHALL refuse, naming publish as the remedy

#### Scenario: No status writes in thin-editor mode

- **WHEN** a thin-editor apply completes
- **THEN** the CLI SHALL NOT have written any `status` field on the CR

### Requirement: Operator-owned delete delegates to the operator's finalizer

`opm instance delete` against an operator-owned instance SHALL delete the `ModuleInstance` CR and delegate workload cleanup to the operator's `opmodel.dev/cleanup` finalizer, waiting bounded and reporting completion. Before deleting, the CLI SHALL verify the operator is ready and refuse when it is not — deleting a finalizer-armed CR with no running controller wedges the CR in terminating state.

Whether the operator removes the workloads is governed by `spec.prune`, which has no CRD default and which the CLI does not write: absent it, the operator removes the CR and deliberately orphans the workloads. The CR's disappearance therefore proves the finalizer completed, not that anything was pruned. The CLI SHALL report the outcome that actually occurred and MUST NOT claim a prune it has not established.

#### Scenario: Delete without spec.prune reports the orphaning

- **WHEN** `opm instance delete` removes an operator-owned instance whose `spec.prune` is unset
- **THEN** the CLI SHALL report that the resources were left running, and name the remedy
- **AND** SHALL NOT report that any resource was pruned

#### Scenario: Delete of an operator-owned instance

- **WHEN** `opm instance delete` targets an operator-owned instance and the operator is ready
- **THEN** the CLI SHALL delete the CR, wait for finalizer cleanup, and report the removal

#### Scenario: Delete refused while the operator is down

- **WHEN** the operator Deployment is not available
- **AND** `opm instance delete` targets an operator-owned instance
- **THEN** the CLI SHALL refuse, explaining the finalizer would wedge without a running operator

## MODIFIED Requirements

### Requirement: Ownership mode resolution is a single branch point

The mapping from `spec.owner` to a CLI execution mode SHALL be implemented in exactly one function that all mutating commands (`apply`, `delete`, `handoff`) consume. CLI-executor mode (absent CR or `spec.owner: cli`) drives the direct render/apply/prune/status path; operator-owned mode drives the thin-editor apply and finalizer-delegating delete paths. `handoff` requires CLI-executor mode as a precondition.

#### Scenario: Apply and delete share the resolver

- **WHEN** `apply` and `delete` evaluate ownership for the same CR
- **THEN** both SHALL obtain their mode from the same resolution function

#### Scenario: Modes route to distinct paths

- **WHEN** the resolver reports operator-owned
- **THEN** `apply` SHALL take the thin-editor path and `delete` the finalizer-delegation path, with no resource-level writes from the CLI

## REMOVED Requirements

### Requirement: Operator-owned instances are refused with actionable errors

**Reason**: The C1 refusal arm was an explicit placeholder for D18's dual-mode handling, which this change ships — refusing is replaced by thin-editor apply and finalizer-delegating delete.
**Migration**: No user migration; previously-refused invocations now perform the D18 behaviors above.
