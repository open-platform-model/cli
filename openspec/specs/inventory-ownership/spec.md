## Purpose

Defines instance ownership semantics derived from the `ModuleInstance` CR's `spec.owner` marker so the CLI and the controller can interoperate without silently taking over each other's instances.

## Requirements

### Requirement: Ownership mode resolution is a single branch point

The mapping from `spec.owner` to a CLI execution mode SHALL be implemented in exactly one function that all mutating commands (`apply`, `delete`) consume. CLI-executor mode (absent CR or `spec.owner: cli`) drives the direct render/apply/prune/status path; controller-owned mode drives the thin-editor apply and finalizer-delegating delete paths. The CLI SHALL NOT change an instance's `spec.owner`: `apply` writes `cli` when it creates an instance and restates the existing value otherwise, so a controller-owned instance is always one created outside the CLI.

#### Scenario: Apply and delete share the resolver

- **WHEN** `apply` and `delete` evaluate ownership for the same CR
- **THEN** both SHALL obtain their mode from the same resolution function

#### Scenario: Modes route to distinct paths

- **WHEN** the resolver reports controller-owned
- **THEN** `apply` SHALL take the thin-editor path and `delete` the finalizer-delegation path, with no resource-level writes from the CLI

#### Scenario: No CLI command changes the owner

- **WHEN** a user looks for a CLI command or flag that moves an instance between CLI and controller ownership
- **THEN** none exists; `spec.owner` is set at creation and changed only outside the CLI

### Requirement: spec.owner names the instance owner

Instance ownership SHALL be determined from the `ModuleInstance` CR's `spec.owner` field. A CR the CLI writes SHALL always carry `spec.owner: cli` explicitly. For reads: an absent CR or `spec.owner: cli` SHALL resolve to CLI-executor mode; `spec.owner: controller` SHALL resolve to controller-owned. (An absent or empty `spec.owner` on an existing CR means controller-managed by the controller's own defaulting contract; the CLI SHALL treat any value other than `cli` as controller-owned.) Source: 0032:D1:R1, 0032:D1:R6.

#### Scenario: CLI-owned instance

- **WHEN** the CLI reads a `ModuleInstance` with `spec.owner: cli`
- **THEN** the instance SHALL resolve to CLI-executor mode

#### Scenario: Controller-owned instance

- **WHEN** the CLI reads a `ModuleInstance` with `spec.owner: controller` or without `spec.owner`
- **THEN** the instance SHALL resolve to controller-owned

### Requirement: Thin-editor apply on controller-owned instances

`opm instance apply` against a controller-owned instance SHALL act as a spec editor only: unify the value inputs, server-side-apply the CLI-owned spec with `spec.module` and `spec.values` updated (field manager `opm-cli`), then wait, bounded, for the controller's reconcile and report its resulting status. In this mode the CLI MUST NOT render-and-apply resources, prune, write `status`, or run the status-RBAC pre-flight. The version-skew ceiling gate still applies. A module resolving from local bytes (local directory, or a local-path `replaceWith` in effect) SHALL be refused in this mode — the controller cannot fetch a local checkout.

The applied document SHALL carry the instance's current `spec.owner` value, read from the live CR, rather than omitting the field. Omitting it does not leave ownership untouched: it releases this field manager's claim, and the API server prunes the controller's ownership marker. The CLI SHALL NOT change the owner's value in this mode.

#### Scenario: The controller's ownership survives a values edit

- **WHEN** a thin-editor apply updates `spec.values` on an instance with `spec.owner: controller`
- **THEN** `spec.owner` SHALL still be `controller` afterwards
- **AND** `spec.module` SHALL be unchanged unless the apply itself changed it

#### Scenario: Values update on a controller-owned instance

- **WHEN** `opm instance apply` targets an instance with `spec.owner: controller` and new values
- **THEN** the CLI SHALL patch `spec.values`, wait for the controller's reconcile, and report the controller's `Ready` outcome
- **AND** SHALL NOT apply or prune any resource itself

#### Scenario: Local module refused in thin-editor mode

- **WHEN** the apply's module would resolve from a local directory or local replacement
- **AND** the target instance is controller-owned
- **THEN** the CLI SHALL refuse, naming publish as the remedy

#### Scenario: No status writes in thin-editor mode

- **WHEN** a thin-editor apply completes
- **THEN** the CLI SHALL NOT have written any `status` field on the CR

### Requirement: Controller-owned delete delegates to the controller's finalizer

`opm instance delete` against a controller-owned instance SHALL delete the `ModuleInstance` CR and delegate workload cleanup to the controller's `opmodel.dev/cleanup` finalizer, waiting bounded and reporting completion. Before deleting, the CLI SHALL verify the controller is ready and refuse when it is not — deleting a finalizer-armed CR with no running controller wedges the CR in terminating state. The readiness check SHALL locate the controller by its fixed names, as the `controller-lifecycle` capability's running-controller check defines.

Whether the controller removes the workloads is governed by `spec.prune`, which has no CRD default and which the CLI does not write: absent it, the controller removes the CR and deliberately orphans the workloads. The CR's disappearance therefore proves the finalizer completed, not that anything was pruned. The CLI SHALL report the outcome that actually occurred and MUST NOT claim a prune it has not established.

#### Scenario: Delete without spec.prune reports the orphaning

- **WHEN** `opm instance delete` removes a controller-owned instance whose `spec.prune` is unset
- **THEN** the CLI SHALL report that the resources were left running, and name the remedy
- **AND** SHALL NOT report that any resource was pruned

#### Scenario: Delete of a controller-owned instance

- **WHEN** `opm instance delete` targets a controller-owned instance and the controller is ready
- **THEN** the CLI SHALL delete the CR, wait for finalizer cleanup, and report the removal

#### Scenario: Delete refused while the controller is down

- **WHEN** the controller Deployment is not available
- **AND** `opm instance delete` targets a controller-owned instance
- **THEN** the CLI SHALL refuse, explaining the finalizer would wedge without a running controller

### Requirement: One tool owns an instance at a time

The CLI and the controller SHALL treat ownership as exclusive, coordinated through the `ModuleInstance` CR's `spec.owner` field. The CLI MUST NOT act as the resource reconciler (apply, prune, inventory write, status write) for an instance whose `spec.owner` is not `cli`. The controller skips render/apply/prune for `spec.owner: cli` instances (controller-side contract, enhancement 0006 D3/A4).

#### Scenario: CLI sees controller-owned instance

- **WHEN** the CLI loads a `ModuleInstance` whose `spec.owner` is `controller`
- **THEN** the CLI SHALL treat the instance as not mutable by the CLI's direct-resource path

#### Scenario: Controller sees CLI-owned instance

- **WHEN** the controller reconciles a `ModuleInstance` with `spec.owner: cli`
- **THEN** the controller SHALL skip render/apply/prune and never touch CLI-written status fields
