## Context

See proposal.md for motivation. Handoff is the top layer of enhancement 0006's three layers: (1) the `ModuleInstance` CR as CLI inventory plus the library kernel as the single render path, (2) the ownership model (`spec.owner`, thin-editor apply, operator-owned delete, provenance annotation, render digest), (3) the `handoff` command. Only layer 3 goes. The operator has no notion of handoff; it honours `spec.owner` (`ManagedExternally` for `cli`), which is layer 2 and stays, so no operator change is involved.

Today the CLI has exactly one arrow from CLI ownership to operator ownership, the handoff flip (`inventory.ApplySpec` with `Owner: OwnerOperator` in `internal/workflow/handoff/handoff.go`). Every other writer either sets `cli` on create (`internal/workflow/apply/apply.go`) or restates the existing owner (`internal/workflow/apply/thineditor.go`). The two operator-owned e2e tests reach operator ownership by invoking the command.

## Goals / Non-Goals

**Goals:** remove the command and everything only it uses; leave layer 2 byte-for-byte in behavior; make the documentation state the resulting model plainly; keep the surviving operator-owned tests green with a setup that does not depend on the removed command.

**Non-Goals:** a replacement transfer path (explicitly declined); removing `status.lastAppliedRenderDigest` or the provenance annotation; touching `opm-operator`; rewriting history (`CHANGELOG.md`, `docs/rfc/0007`, archived changes, enhancement 0006).

## Research & Decisions

### What is handoff-only

**Context**: the removal must not take layer 2 with it.
**Explored**: every symbol the handoff package imports, and every reader of the two CR fields it verifies against.
**Options considered**:
1. Delete by grep for "handoff": over-removes, since the word appears in comments on shared code (`digest.go`, `cr.go`, `record.go`, `ownership.go`, the SSA integration proof).
2. Delete by call graph: remove only what has no caller left once the command is gone.
**Decision**: option 2. Handoff-only: `internal/workflow/handoff/` (its library imports `kernel`, `synth`, `materialize` all have other users), `internal/cmd/instance/handoff.go` plus its two command tests, `internal/inventory/drift.go` (+ test; `DescribeEntrySetDrift` has one caller, the verdict). Shared and kept: `inventory/reconcile.go` (thin editor), `OwnerOperator`/`ResolveOwnership`/`DisplayOwner`, `operator.CheckReady` (delete), `inventory.ApplySpec`, the digest algorithm, the provenance annotation (read by `thineditor.go`).
**Rationale**: the compiler and `golangci-lint` (unused) confirm the boundary mechanically.

### Setup for the surviving operator-owned e2e tests

**Context**: `TestE2E_ThinEditor_ValuesRoundTrip` and `TestE2E_Delete_OperatorOwnedDelegates` call `instance handoff` to obtain an operator-owned instance.
**Explored**: the delete test already patches `spec.prune` with `kubectl patch --type=merge`; the operator adopts CLI-applied resources on its first reconcile regardless of who set `spec.owner`.
**Options considered**:
1. `applyCLIOwned` then `kubectl patch ... -p '{"spec":{"owner":"operator"}}'`, then wait for the operator's first reconcile (`Ready=True` for the new generation): keeps CLI-applied workloads under test as adopted by the operator, one line, mirrors the existing `spec.prune` precedent.
2. `kubectl apply` an operator-owned CR from scratch: no adoption, but the CUE fixture then needs a YAML twin and the thin-editor test's "values round-trip" assumes the CLI wrote the first spec.
**Decision**: option 1, in one helper (`makeOperatorOwned` or similar) that both tests call.
**Rationale**: smallest change, and adoption of CLI-applied resources is exactly what a redesigned transfer will need as a baseline. The kubectl patch takes SSA ownership of `spec.owner` away from `opm-cli`; the thin-editor apply restates the owner with force, which is what it does today after a handoff, so nothing new is exercised.

### Naming after removal

**Context**: `handoff` appears in file, directory, identifier and comment names that outlive the command.
**Explored**: `tests/e2e/instance_handoff_test.go`, `tests/e2e/testdata/handoff/`, helpers `runHandoffOPM`/`resetHandoffInstance`/`handoffInstance`, and the fixture pin in the workspace root `.tasks/deps/fixtures.sh`.
**Options considered**:
1. Keep names, remove code: cheapest, leaves a misleading trail.
2. Rename to the surviving concept, operator-owned instances: one extra line in the root fixtures script.
**Decision**: option 2. File `instance_operator_owned_test.go`, fixture `testdata/operator-owned/`, identifiers `operatorOwnedInstance`, `runOperatorOwnedOPM`, `resetOperatorOwnedInstance` (final spelling is the implementer's). The workspace-root references move in the same PR set: the `.tasks/deps/fixtures.sh` pin and path comment, the `deps:pins:fixtures` description in `.tasks/deps.yml`, and the root `CLAUDE.md` mention.
**Rationale**: a reader landing on "handoff" a year from now would look for a command that does not exist.

### Fields that stay written with no CLI reader

**Context**: `status.lastAppliedRenderDigest` loses its only reader; the provenance annotation does not (thin editor).
**Decision**: both stay written; comments in `digest.go`, `render/types.go`, `record.go`, `cr.go` say why (a future ownership transfer verifies against them; the digest algorithm stays frozen for that reason).
**Rationale**: user decision 2026-08-29; removing and re-adding a status field churns the CR shape twice.

### Documentation framing

**Context**: README frames operator management as something an instance graduates to.
**Decision**: the "CLI-managed vs operator-managed instances" section states: the CLI creates CLI-managed instances; an operator-managed instance is one created outside the CLI (kubectl, GitOps, the operator's own manifests), and against it the CLI is a spec editor. The `instance delete` / `spec.prune` material stays in that section. The "Graduating an instance" section, the command-table row and the QUICKSTART "Graduating to the Operator" section go. No "coming back later" promise in shipped docs.

## Risks / Trade-offs

- [The kubectl-patched owner leaves `spec.owner` field-managed by `kubectl-patch`, not `opm-cli`] → the thin-editor apply force-applies the same value; the SSA integration proof (`tests/integration/ssa-ownership`) covers the co-owner case and stays.
- [Removing a command on the alpha line surprises a user with a script] → `BREAKING CHANGE` footer in the commit, changelog entry, README no longer lists it.
- [A comment mentioning handoff survives] → `grep -ri handoff` over non-history files as a final task; the allowed survivors are `CHANGELOG.md`, `docs/rfc/0007*`, `openspec/changes/archive/**`, and `docs/comparisons/**` (uses the word in its generic English sense, not the command).

## Migration Plan

Single release. Rollback is reverting the commit; no cluster state is written by the removal, and instances already operator-owned stay operator-owned.
