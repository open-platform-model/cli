# Tasks: cli-cr-inventory-backend

## 1. Foundations (CUE bump + CR store)

- [ ] 1.1 Bump `cuelang.org/go` to v0.17.1 in `go.mod`; `task check` green (design D9; trial-verified — expect zero code changes)
- [ ] 1.2 Add an integration fixture proving the loader honors `cue.mod/local-module.cue` `replaceWith` (local dir replacing a registry path, including a never-published path); fix loader resolution if it bypasses the SDK's module resolution
- [ ] 1.3 Define the `ModuleInstance` GVR/kind constants in `internal/inventory`; replace `internal/operator/uninstall.go`'s private `moduleInstanceGVR` with the shared import (design D1)
- [ ] 1.4 Implement entry↔CRD-wire-shape mapping (`group/kind/namespace/name/v/component`; `inventory.revision/digest/count/entries`) with round-trip unit tests (design D2)
- [ ] 1.5 Implement the CR store in `internal/inventory`: GET (NotFound ⇒ no inventory), SSA spec apply, SSA status-subresource apply, delete (NotFound ⇒ success), list + `status.instanceUUID` matching — all `unstructured` via the dynamic client, manager `opm-cli` (design D3)

## 2. Ownership and provenance

- [ ] 2.1 Implement the single ownership mode-resolution function (`spec.owner` → mode) replacing `EnsureCLIMutable`/`createdBy`; unit-test all owner values (design D4)
- [ ] 2.2 Loader records render provenance (main-module-local, or any local-path `replaceWith` in the main module's `local-module.cue`); apply stamps/omits `module-instance.opmodel.dev/source: local` accordingly; test SSA removal on registry re-apply (design D7)
- [ ] 2.3 Extract `status.instanceUUID` from rendered `module-instance.opmodel.dev/uuid` labels (first non-empty, omit when absent) with unit tests (design D3)

## 3. Pre-apply gate battery

- [ ] 3.1 CRD presence gate with the `opm operator install --crds-only` hint (design D5.1)
- [ ] 3.2 CRD field-presence floor (`spec.owner` + `status.inventory` in the served storage-version schema), unit-tested against the embedded B2 CRD manifest (design D5.2)
- [ ] 3.3 Operator-version ceiling reading `Platform.status.operatorVersion`: absent ⇒ skip; RBAC-denied read ⇒ skip with warning; dev build ⇒ skip with warning; semver-older CLI ⇒ refuse (design D5.3)
- [ ] 3.4 `SelfSubjectAccessReview` pre-flight for `patch moduleinstances/status` (CLI-executor mode only), aborting before any apply with the `--rbac` remedial hint (design D5.5)
- [ ] 3.5 Wire the gate battery into `internal/workflow/apply` in order (gates → ownership → SSAR → existence check), dry-run exempt; integration-test gate ordering guarantees zero writes on failure

## 4. Apply/delete/status/list/diff rewire

- [ ] 4.1 Rewire `internal/workflow/apply`: previous inventory from the CR, spec write (owner/module/values — canonical declared reference for local applies), stale set + prune via existing logic, status subset write after prune (no conditions), dry-run untouched
- [ ] 4.2 One-time Secret→CR migration in the apply path: legacy lookup, revision continuation, delete-Secret-after-status-write, idempotent re-run (design D6)
- [ ] 4.3 Rewire `internal/workflow/query` (status/diff/list health): inventory resolution and per-entry discovery read the CR record; re-home `discover.go` input types
- [ ] 4.4 Rewire `instance delete`: ownership refusal with kubectl hint, prune from CR entries, delete CR last (replace `InventorySecretName` ordering in `internal/kubernetes/delete.go`)
- [ ] 4.5 `instance list`: namespace-scoped CR list + `--all-namespaces`/`-A` cluster-wide list with actionable RBAC error; sorted by name; corrupt CRs warned and skipped
- [ ] 4.6 `instance diff`: orphan detection reads `status.inventory` from the CR

## 5. Deletions and cleanup

- [ ] 5.1 Delete `internal/inventory/{secret.go,crud.go,list.go}` and the Secret-era paths in consumers; keep `pkg/inventory` + `stale.go` untouched; retain Secret unmarshal only inside the migration path
- [ ] 5.2 Remove `pkg/ownership` `createdBy`-based enforcement (superseded by 2.1); sweep remaining `createdBy`/Secret-name references
- [ ] 5.3 `task check` (fmt, vet, lint, tests) green across the repo

## 6. Verification and docs

- [ ] 6.1 e2e: first apply creates CR with correct spec/status subset; re-apply increments revision; rename prunes; delete removes CR last (kind cluster, CRDs via `opm operator install --crds-only`)
- [ ] 6.2 e2e: migration — seed a legacy Secret inventory, apply, assert CR ported + Secret deleted + stale entry pruned; assert status/list do not see unmigrated Secrets
- [ ] 6.3 e2e: gates — missing CRD hint; SSAR denial aborts pre-apply (namespace-scoped user); ceiling inert against current operator release (field absent ⇒ skip)
- [ ] 6.4 Update command reference docs + QUICKSTART: CRD prerequisite, `--all-namespaces`, migration note, breaking Secret removal
- [ ] 6.5 Record the landing `history` event in `enhancements/0006/config.yaml` (slice `cli/<archive-date>-cli-cr-inventory-backend`)

## 7. Blocked on A6 release (do last; independent of 1–6)

- [ ] 7.1 After the opm-operator release carrying `Platform.status.operatorVersion` ships: `task operator:sync VERSION=<tag>` pin bump, then e2e the ceiling gate against a real operator (skip-when-absent + refuse-when-newer paths)
