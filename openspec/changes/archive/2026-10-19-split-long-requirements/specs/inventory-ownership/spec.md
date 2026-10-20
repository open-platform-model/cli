## REMOVED Requirements

### Requirement: Thin-editor apply on operator-owned instances
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "`opm instance apply` against an operator-owned instance acts as a spec editor only", "A module resolving from local bytes is refused in thin-editor mode", "A thin-editor apply carries the live `spec.owner` value and never changes it".

### Requirement: Ownership mode resolution is a single branch point
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Ownership mode resolution is implemented in exactly one function that all mutating commands consume", "The CLI never changes an instance's `spec.owner`".

### Requirement: Operator-owned delete hands cleanup to the operator's finalizer
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Operator-owned instance delete hands cleanup to the finalizer", "`opm instance delete` verifies that the operator is ready before it deletes an operator-owned instance and refuses when it is not", "`opm instance delete` of an operator-owned instance reports the outcome that actually occurred and never claims a prune it has not established".

## ADDED Requirements

### Requirement: `opm instance apply` against an operator-owned instance acts as a spec editor only

`opm instance apply` against an operator-owned instance SHALL act as a spec editor only: unify the value inputs, server-side-apply the CLI-owned spec with `spec.module` and `spec.values` updated (field manager `opm-cli`), then wait, bounded, for the operator's reconcile and report its resulting status. In this mode the CLI MUST NOT render-and-apply resources, prune, write `status`, or run the status-RBAC pre-flight. The version-skew ceiling gate still applies.

#### Scenario: Values update on an operator-owned instance

- **WHEN** `opm instance apply` targets an instance with `spec.owner: operator` and new values
- **THEN** the CLI SHALL patch `spec.values`, wait for the operator's reconcile, and report the operator's `Ready` outcome
- **AND** SHALL NOT apply or prune any resource itself

#### Scenario: No status writes in thin-editor mode

- **WHEN** a thin-editor apply completes
- **THEN** the CLI SHALL NOT have written any `status` field on the CR

### Requirement: A module resolving from local bytes is refused in thin-editor mode

A module resolving from local bytes (local directory, or a local-path `replaceWith` in effect) SHALL be refused in this mode: the operator cannot fetch a local checkout. (Requirement: `opm instance apply` against an operator-owned instance acts as a spec editor only)

#### Scenario: Local module refused in thin-editor mode

- **WHEN** the apply's module would resolve from a local directory or local replacement
- **AND** the target instance is operator-owned
- **THEN** the CLI SHALL refuse, naming publish as the remedy

### Requirement: A thin-editor apply carries the live `spec.owner` value and never changes it

The applied document SHALL carry the instance's current `spec.owner` value, read from the live CR, rather than omitting the field. Omitting it does not leave ownership untouched: it releases this field manager's claim, and the API server prunes the operator's ownership marker. The CLI SHALL NOT change the owner's value in this mode. (Requirement: `opm instance apply` against an operator-owned instance acts as a spec editor only)

#### Scenario: The operator's ownership survives a values edit

- **WHEN** a thin-editor apply updates `spec.values` on an instance with `spec.owner: operator`
- **THEN** `spec.owner` SHALL still be `operator` afterwards
- **AND** `spec.module` SHALL be unchanged unless the apply itself changed it

### Requirement: Ownership mode resolution is implemented in exactly one function that all mutating commands consume

The mapping from `spec.owner` to a CLI execution mode SHALL be implemented in exactly one function that all mutating commands (`apply`, `delete`) consume. CLI-executor mode (absent CR or `spec.owner: cli`) drives the direct render/apply/prune/status path; operator-owned mode drives the thin-editor apply and finalizer-delegating delete paths.

#### Scenario: Apply and delete share the resolver

- **WHEN** `apply` and `delete` evaluate ownership for the same CR
- **THEN** both SHALL obtain their mode from the same resolution function

#### Scenario: Modes route to distinct paths

- **WHEN** the resolver reports operator-owned
- **THEN** `apply` SHALL take the thin-editor path and `delete` the finalizer-delegation path, with no resource-level writes from the CLI

### Requirement: The CLI never changes an instance's `spec.owner`

The CLI SHALL NOT change an instance's `spec.owner`: `apply` writes `cli` when it creates an instance and restates the existing value otherwise, so an operator-owned instance is always one created outside the CLI.

#### Scenario: No CLI command changes the owner

- **WHEN** a user looks for a CLI command or flag that moves an instance between CLI and operator ownership
- **THEN** none exists; `spec.owner` is set at creation and changed only outside the CLI

### Requirement: Operator-owned instance delete hands cleanup to the finalizer

`opm instance delete` against an operator-owned instance SHALL delete the `ModuleInstance` CR and delegate workload cleanup to the operator's `opmodel.dev/cleanup` finalizer, waiting bounded and reporting completion.

#### Scenario: Delete of an operator-owned instance

- **WHEN** `opm instance delete` targets an operator-owned instance and the operator is ready
- **THEN** the CLI SHALL delete the CR, wait for finalizer cleanup, and report the removal

### Requirement: `opm instance delete` verifies that the operator is ready before it deletes an operator-owned instance and refuses when it is not

Before deleting, the CLI SHALL verify the operator is ready and refuse when it is not: deleting a finalizer-armed CR with no running controller wedges the CR in terminating state. The readiness check SHALL locate the operator by its fixed names, as the `operator-lifecycle` capability's running-operator check defines. (Requirement: Operator-owned instance delete hands cleanup to the finalizer)

#### Scenario: Delete refused while the operator is down

- **WHEN** the operator Deployment is not available
- **AND** `opm instance delete` targets an operator-owned instance
- **THEN** the CLI SHALL refuse, explaining the finalizer would wedge without a running operator

### Requirement: `opm instance delete` of an operator-owned instance reports the outcome that actually occurred and never claims a prune it has not established

Whether the operator removes the workloads is governed by `spec.prune`, which has no CRD default and which the CLI does not write: absent it, the operator removes the CR and deliberately orphans the workloads. The CR's disappearance therefore proves the finalizer completed, not that anything was pruned. The CLI SHALL report the outcome that actually occurred and MUST NOT claim a prune it has not established. (Requirement: Operator-owned instance delete hands cleanup to the finalizer)

#### Scenario: Delete without spec.prune reports the orphaning

- **WHEN** `opm instance delete` removes an operator-owned instance whose `spec.prune` is unset
- **THEN** the CLI SHALL report that the resources were left running, and name the remedy
- **AND** SHALL NOT report that any resource was pruned
