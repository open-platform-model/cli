# Proposal: cli-cr-inventory-backend

Enhancement 0006, slice C1. Implements D1, D2 (as amended by D25), D3 (consume), D6 (write-side), D8, D14, D23 (pre-flight half), D24, D27 (as respelled by D32), D33, D36, D37; leaves OQ17 resolved here at drafting time (see design.md).

## Why

The CLI records what it deployed in a proprietary inventory Secret while the operator records the same fact in `ModuleInstance.status.inventory` — two disjoint stores for one fact, which makes safe CLI↔operator handoff (enhancement 0006's goal) structurally impossible. This slice makes the `ModuleInstance` CR the single inventory store both actors share: the operator's prune set is `previous status.inventory − current render`, so once the CLI writes its deployed entries into the same field, the operator's first post-handoff reconcile sees a zero stale set.

## What Changes

- **The CLI writes a `ModuleInstance` CR instead of a Secret** on `opm instance apply` / `opm module apply` (D1): named after the instance, in the target namespace, handled as `unstructured` via the dynamic client (no `opm-operator` Go import — D13), SSA with field manager `opm-cli`. Spec: `owner: cli` (D3), canonical declared `module.path`/`module.version` — for local applies too, never a filesystem path (D6 write-side, D37) — and the unified values blob as `spec.values` (D19). Status (via the status subresource): `inventory`, `instanceUUID` (extracted from the rendered `module-instance.opmodel.dev/uuid` label, same mechanism as the operator), and the `lastApplied*` digest/timestamp set — **no `status.conditions`** (D2/D25).
- **`apply`/`delete`/`status`/`list`/`diff` are rewired** to read/write the CR. `instance list` becomes a native CR list; `--all-namespaces` is a cluster-wide CR list that fails with a clear RBAC error, no label fallback (D29).
- **Pre-apply gates land on the apply path** (D33): missing-CRD fail-with-hint (`ModuleInstance CRD not found — run 'opm operator install --crds-only'`, D27/D32); CRD field-presence floor (`spec.owner` + `status.inventory` in the installed schema) and operator-version ceiling read from `Platform.status.operatorVersion`, absent ⇒ solo cluster, skipped (D24); `SelfSubjectAccessReview` for `moduleinstances/status` before any resource is applied, CLI-owned path only (D23b).
- **Ownership guard as one mode-resolution branch** (D3/D18): operator-owned instances are refused on `apply` and `delete` with actionable hints; D18's thin-editor apply mode and delete symmetry defer to slice C3, which replaces the refusal arm without touching callers.
- **BREAKING: one-time Secret→CR migration, no deprecation window** (D8/D14): on apply, an existing Secret inventory with no CR is migrated (create CR, port record, delete Secret only after the status write succeeds); `status`/`delete`/`list` read CRs only — Secret-tracked instances become visible again on their next apply.
- **Secret-specific code is deleted** (`internal/inventory/secret.go`, `crud.go`, `list.go`); the entry-identity/stale-set/digest/rename-safety/collision logic (`pkg/inventory`, `internal/inventory/stale.go`) is ported onto the CR flow as-is (D31), and per-entry live-resource discovery (`discover.go`) is re-homed, not removed.
- **`cuelang.org/go` v0.16.1 → v0.17.1** (D36 — trial-verified 2026-07-16: zero code changes, unit suite green), including verifying the loader honors `cue.mod/local-module.cue` replacements (D37). Render still uses the CLI's current pipeline; kernel adoption is slice C2.

## Capabilities

### New Capabilities

- `apply-preflight-gates`: the gate battery that runs before any resource is applied — CRD presence hint, CRD field-presence floor, operator-version ceiling, and the `moduleinstances/status` access pre-flight.
- `secret-inventory-migration`: the one-time, delete-after-success Secret→CR migration performed on apply.

### Modified Capabilities

- `instance-inventory`: the persisted record envelope and CRUD move from a Secret to the `ModuleInstance` CR (`status.inventory` + CLI status subset + spec write); entry identity/digest semantics unchanged.
- `inventory-ownership`: ownership derives from `spec.owner` on the CR instead of the record's `createdBy`; refusal semantics for operator-owned instances defined here.
- `inventory-listing`: listing is a native `ModuleInstance` CR list (namespace-scoped by default, `--all-namespaces` cluster-wide, clear RBAC error) instead of a Secret label-selector list.
- `resource-discovery`: per-entry live-resource discovery reads entries from the CR-backed inventory instead of the Secret record.

## Impact

- **Packages**: `internal/inventory` (Secret CRUD deleted; CR store added), `internal/workflow/apply`, `internal/workflow/query`, `internal/cmd/instance/{apply,delete,diff,list,status}.go`, `internal/cmd/module/apply.go`, `internal/kubernetes` (delete path's Secret-last ordering becomes CR-last), `internal/operator` (the hardcoded `moduleInstanceGVR` lifts to a shared home), `go.mod` (CUE bump).
- **SemVer**: breaking (`feat!`) — the Secret inventory format is dropped without a window. Accepted under D14 (single user, no external consumers).
- **Dependencies**: A2/A4/B2 shipped. **A6 gates landing, not drafting**: implemented on opm-operator branch `feat/platform-status-operator-version`, unmerged/unreleased as of 2026-07-16 — the ceiling gate's e2e needs an operator release carrying it plus a `task operator:sync` pin bump.
- **Out of scope** (later slices): kernel adoption / platform resolution / `#ModuleInstance` synthesis retirement (C2), handoff + thin-editor mode (C3), operator-side changes (A6 lands in opm-operator).
