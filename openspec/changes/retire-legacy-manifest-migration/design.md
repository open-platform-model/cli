## Context

The migration lives in `internal/operator` (proof list, proof, plan, execute, report) and reaches two shared places: the apply guard's admission set (`internal/inventory`) and the instance apply request (`internal/workflow/apply`). `cli` does not import the library's `opm/k8s/ownership`, so this change is independent of the library's own `Admit` removal.

## Goals / Non-Goals

**Goals:** delete the migration and every type, parameter, test, fixture and doc that exists only for it; keep the apply guard, the terminating wait and every other install check unchanged.

**Non-Goals:** the operator → controller rename (the next change, `rename-operator-command-to-controller`); the apply-guard drift between "skipped when inventory exists" and 0012:D8 "guard on every apply" (tracked separately); the Secret-to-CR inventory migration (`internal/inventory/legacy.go`, `tests/integration/migration/`), which is unrelated and stays.

## Research & Decisions

### The old-style tag message
**Context**: `operatorTagShape` turned a `--version v1.0.0-beta.5` miss into a message saying `--version` now takes a module version.
**Options considered**:
1. Keep it, reworded - helps users of the manifest-era CLI, of whom there are none.
2. Delete it - a tag-shaped selector is refused as an unparseable or unserved selector, exit 2, before any cluster call.
**Decision**: Delete.
**Rationale**: it exists only for users of the old install, and the owner asked for the rename to leave no trace of the manifest era.

### Admission set
**Decision**: remove the parameter entirely rather than always pass nil. `PreApplyExistenceCheck(ctx, client, entries)` and `RunPreApplyExistenceCheck(..., currentEntries)` lose one argument; `K8sIdentity`/`IdentityOf` had no other user and go with `AdmitSet`.

## Risks / Trade-offs

- An operator applied from a release manifest now fails install at the apply guard (exit 2, names the first foreign object). Accepted: no such install exists outside the owner's homelab, which moves by hand.
