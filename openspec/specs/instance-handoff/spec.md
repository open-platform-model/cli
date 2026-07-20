## Purpose

The `opm instance handoff` command: verified, forward-only transfer of a release from CLI management to operator management. Enhancement 0006 D6 (gate side), D7 as amended by D40, D16, D38 (read side).

## Requirements

### Requirement: Handoff command exists and is forward-only

The CLI SHALL expose `opm instance handoff <name>` (with `--namespace`, `--timeout`, `--force`) under the instance command group. The command transfers ownership CLI → operator only; no reverse mode exists (no `--to` flag). The `--platform` flag SHALL be rejected on this command — handoff always renders against the cluster `Platform`.

#### Scenario: Reverse mode does not exist

- **WHEN** a user seeks to transfer an operator-owned instance back to CLI ownership
- **THEN** no handoff flag or mode offers this; the command surface is forward-only

#### Scenario: --platform rejected

- **WHEN** `opm instance handoff my-app --platform ./platform.cue` is invoked
- **THEN** the command SHALL exit non-zero stating handoff uses the cluster Platform only

### Requirement: Precondition chain runs in order and aborts before the flip

Before patching ownership, the command SHALL verify, in order, aborting on the first failure with an actionable error: (1) the operator is installed and ready (CRDs `Established`, operator Deployment rolled out — reusing the operator-lifecycle readiness machinery); (2) the `ModuleInstance` exists with `spec.owner: cli`; (3) the CR does not carry `module-instance.opmodel.dev/source: local` — if it does, refuse and name the remedy (publish the module, re-apply, then hand off); (4) `spec.module` resolves strictly from the registry; (5) the verification render digest equals `status.lastAppliedRenderDigest`. Failures SHALL leave the CR unmodified.

Because gates 4-5 take real time and ownership is still `cli` throughout them, the command SHALL re-read the instance immediately before the flip and abort if its `metadata.generation` moved — the verification no longer describes what is deployed. It SHALL NOT silently adopt the newer spec, which would flip a document no gate verified.

#### Scenario: Operator not ready

- **WHEN** handoff runs against a cluster where the operator Deployment is not available
- **THEN** the command SHALL exit non-zero naming the operator readiness failure
- **AND** `spec.owner` SHALL remain `cli`

#### Scenario: Local-provenance annotation blocks handoff

- **WHEN** the CR carries `module-instance.opmodel.dev/source: local`
- **THEN** handoff SHALL refuse before any render, with a message naming publish-and-re-apply as the remedy

#### Scenario: Unresolvable module reference blocks handoff

- **WHEN** `spec.module` names a reference that does not exist in the registry
- **THEN** handoff SHALL exit non-zero identifying the unresolvable reference

#### Scenario: Digest mismatch aborts

- **WHEN** the verification render digest differs from `status.lastAppliedRenderDigest`
- **THEN** handoff SHALL abort showing both digests and stating the deployed state does not match the published module

#### Scenario: A concurrent spec change during verification aborts the flip

- **WHEN** the instance's `spec` is modified by another actor between the gate chain's read and the ownership flip
- **THEN** handoff SHALL detect the change, abort before flipping, and report both generations
- **AND** `spec.owner` SHALL remain `cli`, so the retry can re-verify the current spec

#### Scenario: --force overrides only the digest gate

- **WHEN** handoff runs with `--force` and only the digest check fails
- **THEN** the flip SHALL proceed with the mismatch displayed
- **AND** `--force` SHALL NOT bypass the ownership, provenance, or resolvability gates

### Requirement: Verification render is strict-registry and self-compared

The verification render SHALL resolve `spec.module` from the registry through module acquisition that ignores any `cue.mod/local-module.cue` and cannot be satisfied by a stale module cache entry (a fresh, isolated CUE cache directory or equivalent). It SHALL render through the kernel with the cluster `Platform` and the CLI's own runtime name, and compare the resulting digest against the CR's `status.lastAppliedRenderDigest` only. The command MUST NOT compare any CLI digest against an operator-written digest.

#### Scenario: Local replacement cannot satisfy verification

- **WHEN** the working directory's module context contains a `cue.mod/local-module.cue` replacing `spec.module`'s path with a local checkout
- **THEN** the verification render SHALL resolve the module from the registry, not the local checkout

#### Scenario: Verification renders with the CLI runtime name

- **WHEN** the verification render executes
- **THEN** it SHALL pass the CLI's own runtime identity so the digest is comparable with the CLI-recorded `lastAppliedRenderDigest`

### Requirement: Ownership flip changes only the owner

After all gates pass, the command SHALL change exactly one field — `spec.owner` to `operator` — via a single server-side apply with field manager `opm-cli`, leaving the instance's module reference, values, and status as they were.

Because a server-side-apply document is the field manager's complete declared intent, the applied document SHALL carry the instance's current `spec.module` and `spec.values` alongside the new owner. Omitting them would release this manager's claim and cause the API server to prune them: `spec.module` is a required field, so its omission makes the apply fail outright, and `spec.values` would be deleted silently. Restating an unchanged value changes nothing and does not increment `metadata.generation`.

#### Scenario: Flip changes the owner and preserves everything else

- **WHEN** the flip executes against an instance with a module reference and values
- **THEN** `spec.owner` SHALL become `operator`
- **AND** `spec.module` and `spec.values` SHALL be unchanged, not absent
- **AND** no status field SHALL be written

#### Scenario: A no-op re-apply does not advance the generation

- **WHEN** the same spec is applied twice
- **THEN** `metadata.generation` SHALL be unchanged by the second apply, so the post-flip reconcile wait cannot be satisfied by a write that changed nothing

### Requirement: Post-flip wait judges an inventory-stable reconcile

After the flip, the command SHALL wait, bounded by `--timeout`, for the operator's first reconcile, and judge success from CR status alone: the `Ready` condition is `True`, the `status.inventory` entry set equals its pre-handoff value, `status.inventory.revision` has incremented, and no entry was pruned. The expected managed-by relabel SHALL be reported as information (operator adopted N resources; no workload changes), never treated as failure. On timeout or a failed reconcile, the command SHALL exit non-zero reporting the operator's condition and state that ownership remains with the operator (reverse handoff does not exist); it MUST NOT flip ownership back.

#### Scenario: Successful handoff

- **WHEN** the operator's first reconcile completes with `Ready: True`, the same inventory entry set, an incremented revision, and nothing pruned
- **THEN** the command SHALL exit zero reporting the adoption and the relabel

#### Scenario: Reconcile fails after flip

- **WHEN** the operator's first reconcile sets `Ready: False`
- **THEN** the command SHALL exit non-zero reporting the operator's condition message
- **AND** SHALL NOT modify `spec.owner`

#### Scenario: Operator digests are not compared

- **WHEN** the operator writes its own `lastApplied*` digests after the reconcile
- **THEN** the command SHALL NOT treat their difference from the CLI-recorded digests as an error
