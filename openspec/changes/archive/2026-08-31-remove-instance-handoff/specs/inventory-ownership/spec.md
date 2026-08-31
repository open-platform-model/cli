## MODIFIED Requirements

### Requirement: Ownership mode resolution is a single branch point

The mapping from `spec.owner` to a CLI execution mode SHALL be implemented in exactly one function that all mutating commands (`apply`, `delete`) consume. CLI-executor mode (absent CR or `spec.owner: cli`) drives the direct render/apply/prune/status path; operator-owned mode drives the thin-editor apply and finalizer-delegating delete paths. The CLI SHALL NOT change an instance's `spec.owner`: `apply` writes `cli` when it creates an instance and restates the existing value otherwise, so an operator-owned instance is always one created outside the CLI.

#### Scenario: Apply and delete share the resolver

- **WHEN** `apply` and `delete` evaluate ownership for the same CR
- **THEN** both SHALL obtain their mode from the same resolution function

#### Scenario: Modes route to distinct paths

- **WHEN** the resolver reports operator-owned
- **THEN** `apply` SHALL take the thin-editor path and `delete` the finalizer-delegation path, with no resource-level writes from the CLI

#### Scenario: No CLI command changes the owner

- **WHEN** a user looks for a CLI command or flag that moves an instance between CLI and operator ownership
- **THEN** none exists; `spec.owner` is set at creation and changed only outside the CLI
