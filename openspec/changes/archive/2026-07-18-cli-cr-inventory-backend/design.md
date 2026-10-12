# Design: cli-cr-inventory-backend

Enhancement 0006, slice C1. Decision numbers `DN` below are local to this change; enhancement decisions are cited as `0006/DN`.

## Context

Today the CLI persists "what did I deploy" in a proprietary inventory Secret (`internal/inventory`: `secret.go` marshaling, `crud.go` get/write, `list.go` label-selector listing, `discover.go` per-entry GETs), while the operator persists the same fact in `ModuleInstance.status.inventory`. The apply flow (`internal/workflow/apply/apply.go`) computes digests, loads the previous Secret, computes the stale set (`pkg/inventory` + `internal/inventory/stale.go`), SSA-applies with field manager `opm-cli`, prunes, and writes the Secret back. Slices A2 (module rename), A4 (`spec.owner` marker CRD), and B2 (`opm operator install/uninstall`, embedded pinned CRD, `opm-cli` manager rename) have shipped; A6 (`Platform.status.operatorVersion`) is implemented on an opm-operator branch but unreleased.

This slice replaces the Secret with the `ModuleInstance` CR as the single inventory store, adds the pre-apply gate battery, performs the one-time migration, and bumps CUE to v0.17.1. Render still goes through the CLI's current `pkg/render` pipeline — kernel adoption is slice C2; handoff is C3.

## Goals / Non-Goals

**Goals:**

- `apply`/`delete`/`status`/`list`/`diff` read/write the `ModuleInstance` CR; the Secret backend is deleted.
- The CLI writes exactly the enhancement 0006 ("The CLI writes a strict subset of status") status subset (as amended by enhancement 0006 ("The CLI writes no `status.conditions`; conditions are operator-exclusive (amends D2, resolves OQ11)") — no conditions) and the enhancement 0006 ("Explicit `spec.owner` marker; the operator skips CLI-owned CRs") `spec.owner: cli` marker; `spec.module` is always the canonical declared reference (enhancement 0006 ("Spec contents when applying from a local module") write-side, enhancement 0006 ("`cue.mod/local-module.cue` is the sanctioned local-module workflow; `spec.module` always carries the canonical declared reference; the CR never represents a local path")).
- Pre-apply gates: missing-CRD hint, CRD field floor, operator-version ceiling, status-RBAC pre-flight (enhancement 0006).
- One-time Secret→CR migration, delete-Secret-after-success, no deprecation window (enhancement 0006, "Migration of existing Secret inventories" and "No backwards-compatibility or deprecation burden; the CUE v0.17-alpha bump is accepted").
- `cuelang.org/go` v0.16.1 → v0.17.1 (enhancement 0006, "The CLI's CUE bump retargets to v0.17.1 and relocates from C2 to C1; library/operator pin bumps happen outside this enhancement; amends the first clause of its earlier CLI pin decision"), with the `cue.mod/local-module.cue` loader behavior verified and surfaced as render provenance (enhancement 0006).

**Non-Goals:**

- Kernel adoption, platform resolution, `--platform` flag, `#ModuleInstance` synthesis retirement (C2).
- `opm instance handoff`, the thin-editor apply mode for operator-owned CRs, delete symmetry (C3) — this slice ships refusals at the branch point C3 will extend.
- Any operator-side change (A6 lands in opm-operator).
- Prune-behavior changes: stale-set/rename-safety/existence-check logic is ported as-is (enhancement 0006, "Revert the shared `library/opm/inventory` package (the CLI and operator keep independent local inventory implementations); supersedes D13's shared-logic clause and D26"); reconciling the CLI/operator stale-set base relation is an open question of enhancement 0006.

## Decisions

### D1: The CR backend lives in `internal/inventory`, rewritten in place

`internal/inventory` keeps its name and its consumers; the Secret files (`secret.go`, `crud.go`, `list.go`) are replaced by a CR-backed store (`cr.go`, `store.go` or equivalent). The `ModuleInstance` GVR, kind, and field constants move here as the single source; `internal/operator/uninstall.go`'s private `moduleInstanceGVR` copy is replaced by an import. Per repo rule ("update existing packages over new abstractions") and enhancement 0006 ("The CLI imports `library` only; it does not import `opm-operator` (no controller-runtime, no Flux); supersedes D4's dependency direction") (no `opm-operator` Go import), the CR is handled as `unstructured` via the dynamic client. `pkg/inventory` (entry identity, stale set, digest) and `internal/inventory/stale.go` (rename safety, `PreApplyExistenceCheck`, prune) are untouched in semantics. `discover.go`'s per-entry live-resource discovery is retained (it is entry-driven, not Secret-driven) with its input re-typed to the CR-backed record.

*Alternative — a new `internal/instancecr` package:* rejected; it would leave `internal/inventory` a stub and force every consumer to re-import for no boundary gain.

### D2: Explicit wire-shape mapping, never struct-tag marshaling

The CRD serializes `InventoryEntry.Version` as `v` (`opm-operator/api/v1alpha1/common_types.go`); the CLI's `pkg/inventory.InventoryEntry` has its own JSON tags. Conversion between `pkg/inventory` types and the CR's `status.inventory` is done by explicit map-building functions that target the CRD's OpenAPI shape (`group`, `kind`, `namespace`, `name`, `v`, `component`; `inventory.revision/digest/count/entries`), with round-trip unit tests. Marshaling Go structs by their tags into the unstructured object is forbidden — the CRD schema, not the Go tags, anchors cross-actor shape parity (enhancement 0006, "Revert the shared `library/opm/inventory` package (the CLI and operator keep independent local inventory implementations); supersedes D13's shared-logic clause and D26").

### D3: Write mechanics — SSA for spec, SSA on the status subresource for status, guard-read first

Every apply: (1) GET the CR (404 ⇒ new instance); (2) resolve the ownership mode (D4); (3) SSA-apply the spec document (create-or-update) with field manager `opm-cli`: `spec.owner: cli`, `spec.module` (canonical declared path/version — local applies included, enhancement 0006 ("`cue.mod/local-module.cue` is the sanctioned local-module workflow; `spec.module` always carries the canonical declared reference; the CR never represents a local path")), `spec.values` (the unified blob, enhancement 0006 ("`spec.values` is the sole authoritative render input; the CLI unifies all value inputs into it; resolves OQ10")), plus managed labels and the provenance annotation (D7); (4) after resources are applied and pruned, SSA-apply the status subset on the **status subresource**: `inventory`, `instanceUUID`, `lastAppliedRenderDigest`, `lastAppliedSourceDigest`, `lastAppliedConfigDigest`, `lastAppliedAt` — never `conditions`, never `observedGeneration` (enhancement 0006, "The CLI writes a strict subset of status" and "The CLI writes no `status.conditions`; conditions are operator-exclusive (amends D2, resolves OQ11)"). `instanceUUID` is extracted from the rendered resources' `module-instance.opmodel.dev/uuid` label (first non-empty — the operator's `extractInstanceUUID` mechanism; deterministic UUIDv5 computed in core CUE), omitted if absent. Dry-run applies touch neither the CR nor the gates that exist to protect writes (D5 note).

### D4: One ownership branch point

A single mode-resolution function (e.g. `ResolveOwnership(cr)`) maps `spec.owner` to a mode: absent CR or `owner: cli` (or empty — a CR the CLI itself wrote always carries `cli` explicitly) ⇒ CLI-executor mode; `owner: operator` ⇒ refusal in this slice. `apply` refuses with "instance is operator-managed (`spec.owner: operator`); the CLI does not edit operator-owned instances yet". `delete` refuses with the kubectl hint (`kubectl delete moduleinstance <name>` — the operator's cleanup finalizer prunes the workloads). C3 replaces the refusal arm with the enhancement 0006 ("After handoff the CLI may still edit the CR's spec; it writes only the CR and defers execution to the operator; refines D3's `EnsureCLIMutable` guard") thin-editor mode without touching callers. This function replaces the record-based `EnsureCLIMutable`/`createdBy` guard.

### D5: Gate battery — order, mechanics, and the dev-build ceiling rule

On real (non-dry-run) applies, before any resource is written, in order:

1. **CRD present** — GET the `moduleinstances.opmodel.dev` CRD (apiextensions). NotFound ⇒ `ModuleInstance CRD not found — run 'opm operator install --crds-only'` (enhancement 0006, "No `--install-crds` flag; missing CRDs fail with a hint to run `opm install crds`; resolves OQ7" and "Noun-first `opm operator` command group (`opm operator install [--crds-only]` / `opm operator uninstall`); supersedes D28").
2. **CRD field floor** — the served storage version's schema must contain `spec.owner` and `status.inventory` properties. Missing ⇒ "ModuleInstance CRD is missing required fields — run 'opm operator install --crds-only'" (enhancement 0006 ("CRD/operator version skew: CLI-≥-cluster invariant, gate-only (CRD field-presence floor + operator-version ceiling via `Platform.status`); resolves OQ4") floor; guards the API server silently pruning `spec.owner`).
3. **Operator-version ceiling** — GET the singleton cluster `Platform`; if the CR or `status.operatorVersion` is absent ⇒ solo cluster, skip (enhancement 0006 ("CRD/operator version skew: CLI-≥-cluster invariant, gate-only (CRD field-presence floor + operator-version ceiling via `Platform.status`); resolves OQ4") semantics; requires A6-carrying operator for the field to exist). Present and semver-greater than the CLI's version ⇒ refuse ("your CLI is older than the cluster operator — upgrade the CLI"). **Dev builds** (version not valid semver, e.g. `dev`) skip the ceiling with a warning — a dev CLI is presumed current, and refusing would make every from-source build unusable against real clusters.
4. **Ownership** (D4).
5. **Status-RBAC pre-flight** — `SelfSubjectAccessReview` for `patch moduleinstances/status` in the target namespace; denial aborts before any apply with the enhancement 0006 ("Status-subresource RBAC (opt-in `--rbac` emission + a pre-flight access check, both as edge-case safety nets); resolves OQ3") remedial hint. CLI-executor mode only.
6. **Existing `PreApplyExistenceCheck`** (first-ever apply only), unchanged.

Gates 1–3 are read-only cluster probes; RBAC failures reading the Platform in gate 3 degrade to skip-with-warning rather than hard failure (a namespace-scoped user must still be able to apply — enhancement 0006 ("Both a local Platform and an in-cluster Platform CR are first-class; OPM must never require cluster-admin for normal use") accessibility).

### D6: Migration — read Secret once, write CR, delete Secret only after status write succeeds

On apply, when the CR is absent but a legacy inventory Secret exists (direct name + label fallback, today's lookup): build the CR from the Secret record (entries/revision/digest/count into `status.inventory`, record UUID into `instanceUUID`, timestamps into `lastAppliedAt`; spec comes from the *current* apply's module/values), proceed with the normal apply, and delete the Secret **only after** the CR status write succeeds. A failure anywhere leaves the Secret intact and the release discoverable by a re-run. `status`/`delete`/`list`/`diff` never read Secrets (enhancement 0006 ("Migration of existing Secret inventories") as amended by enhancement 0006 — no fallback window); an instance tracked only by a Secret reappears on its next apply.

### D7: Render provenance annotation — written here, consumed by C3's handoff as a fail-closed pre-gate (resolves an open question of enhancement 0006)

The loader records render provenance; when the module bytes did not come from pure registry resolution — the main module is a local directory, **or** the main module's `cue.mod/local-module.cue` contains any local-path `replaceWith` (conservative: any replacement marks the render, since replaced dependencies also change bytes) — the CLI stamps `module-instance.opmodel.dev/source: local` on the CR in the spec apply. When a later apply resolves fully from registries, the annotation is omitted from the applied document and SSA field ownership removes it. The annotation **only ever blocks**: C3's handoff refuses while it is present ("last apply used a local module — publish and re-apply, then hand off"), and the authoritative backstop is C3's verification render, which MUST bypass `local-module.cue` and resolve `spec.module` strictly from the registry — otherwise a local replacement would make enhancement 0006 ("`opm instance handoff`, the migration feature").4's digest check self-consistent and meaningless. Stripping the annotation by hand grants nothing (the strict-registry digest gate stands behind it); a checkout byte-identical to the published module clears naturally via the re-apply the error prescribes. The C3-side requirements are recorded in enhancement 0006 with this OQ17 resolution; this slice ships the write side plus detection.

### D8: `instance list` — native CR list (enhancement 0006)

`ListInventories` is replaced by a namespace-scoped `ModuleInstance` list; a new `--all-namespaces` flag issues a cluster-wide list. Insufficient RBAC surfaces as a clear, actionable error (no label fallback, no silent narrowing). Health evaluation (`internal/workflow/query/list.go`) keeps its per-entry discovery, fed from the CR record. `--instance-id` selectors resolve by listing and matching `status.instanceUUID`.

### D9: CUE v0.17.1 in the same slice (enhancement 0006)

`go.mod` bumps `cuelang.org/go` v0.16.1 → v0.17.1 first (trial-verified 2026-07-16: zero code changes, unit suite green). An integration fixture with a `cue.mod/local-module.cue` replacement verifies the loader honors `replaceWith` end-to-end — the same mechanism D7's detection relies on. If the loader's own resolution wiring bypasses the local module file, that is fixed here (it must go through the SDK's standard module resolution).

## Risks / Trade-offs

- **A6 is unreleased** → the ceiling gate cannot be exercised e2e until an operator release carries `Platform.status.operatorVersion` and the B2 pin is bumped (`task operator:sync`). → Unit-test the gate against fake dynamic objects now; land the e2e once the release exists. Absence semantics (skip) mean the gate is safe-but-inert against today's operator — which is exactly the enhancement 0006 ("CRD/operator version skew: CLI-≥-cluster invariant, gate-only (CRD field-presence floor + operator-version ceiling via `Platform.status`); resolves OQ4") caveat, not a regression.
- **CRD schema introspection (floor gate) is structurally fiddly** → keep it to two property-path existence checks on the served storage version; unit-test against the embedded B2 CRD manifest (which already contains both fields).
- **SSA removal semantics for the provenance annotation** (annotation must disappear when omitted) → covered by an envtest-style unit test; if granular annotation ownership proves unreliable, fall back to explicitly writing `source: registry`/removing via JSON patch — same contract, different mechanism.
- **Migration mid-failure** → delete-after-success ordering leaves the Secret authoritative until the CR status write lands; re-running apply is idempotent (CR now exists, Secret still present ⇒ migration retries the delete only).
- **Version-compare edge cases (ceiling gate)** → dev builds skip-with-warning (D5); pre-release semver compares per semver 2.0.0 via a vetted library.
- **Old CLI + migrated instance** → after migration the Secret is gone; an older CLI sees nothing. Accepted explicitly under enhancement 0006 ("No backwards-compatibility or deprecation burden; the CUE v0.17-alpha bump is accepted") (single user, no external consumers).

## Migration Plan

Single release, no flags, no window: ship, re-apply live instances once (each apply migrates its own Secret), done. Rollback = previous CLI binary, which still reads any not-yet-migrated Secrets; already-migrated instances would need a manual re-apply under the old binary to recreate Secrets (accepted, enhancement 0006). Docs: `apply`/`delete`/`list`/`status` reference pages and the QUICKSTART gain the CRD-prerequisite note and the `opm operator install --crds-only` first-run step.

## Open Questions

None blocking implementation. Landing-order note: the ceiling gate's e2e and the embedded-pin bump wait on the A6 opm-operator release; everything else in this slice is independent of it.
