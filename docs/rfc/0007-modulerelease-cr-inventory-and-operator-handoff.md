# RFC-0007: ModuleRelease CR Inventory and Operator Handoff

| Field          | Value                                       |
|----------------|---------------------------------------------|
| **Status**     | Draft                                       |
| **Created**    | 2026-06-12                                  |
| **Authors**    | OPM Contributors                            |
| **Supersedes** | RFC-0001 (storage mechanism only)           |
| **Affects**    | `cli/`, `opm-operator/` (documented here)   |

## Summary

Replace the CLI's Secret-based release inventory (RFC-0001) with the operator's `ModuleRelease` custom resource. The CLI creates and updates a ModuleRelease CR — a deliberately simpler usage of the same CRD the operator owns — and stores its inventory in `status.inventory`, using inventory code imported from the operator module rather than the CLI's own copy.

The goal is not storage hygiene. It is to pave the way for a **zero-downtime handoff**: a user who starts with the CLI alone can later install the operator and transfer ownership of a deployed release to it without re-deploying anything. Because both actors would read and write the same inventory in the same CR, takeover becomes a no-op server-side apply.

This RFC also adds operator installation to the CLI (`opm install`), which becomes a prerequisite rather than a convenience: once inventory lives in a CR, the CRDs must exist in every cluster the CLI applies to.

This is a cross-repo design. The operator-side changes are documented here (not in `opm-operator/`) so the whole design can be reviewed as one unit; implementation will be sliced into per-repo OpenSpec changes (see "Implementation Slices").

## Motivation

### The learner-to-operator migration path

The expected adoption path for OPM is:

1. A user discovers OPM and tries it with only the CLI — `opm release apply` against any cluster, no operator, no CRDs, minimal footprint.
2. The user grows into GitOps / continuous reconciliation and installs the operator.
3. The user wants their already-deployed releases managed by the operator — **without downtime, without orphaned resources, without re-deploying**.

Today step 3 is impossible. The CLI records its inventory in a Secret (`opm.<releaseName>.<releaseID>`, RFC-0001); the operator records its inventory in `ModuleRelease.status.inventory`. They are two disjoint sources of truth with structurally identical content. Handing off a release would mean manually constructing a ModuleRelease CR, hoping the operator's first reconcile happens to be a no-op, and deleting the Secret — with pruning logic on each side unaware of the other's record.

### The convergence is already half-built

Three existing facts make this design cheap relative to its payoff:

1. **The inventory schemas are near-identical.** The CLI Secret stores `{Revision, Digest, Count, Entries[]}` where an entry is `{Group, Kind, Namespace, Name, Version, Component}`. The operator's `status.inventory` (`api/v1alpha1/common_types.go`) stores exactly the same shape. This is not coincidence: the operator's archived change `2026-04-12-01-cli-dependency-and-inventory-bridge` deliberately copied the CLI's inventory functions (`ComputeStaleSet`, digest computation, entry identity) into `opm-operator/internal/inventory/`, and its design doc explicitly defers "promote to `pkg/` for external reuse" to a future enhancement. This RFC is that enhancement, with the dependency direction inverted: the CLI imports the operator's code and deletes its own copy.

2. **Resource labeling already converges.** Both actors stamp `module-release.opmodel.dev/{uuid,name,namespace}` and `app.kubernetes.io/managed-by` on managed resources, and the operator's prune ownership guard already accepts both `opm-cli` and `opm-controller` as recognized managed-by values.

3. **The ownership marker exists.** The CLI inventory record carries `CreatedBy: "cli" | "controller"` and the CLI refuses to mutate controller-owned releases (`EnsureCLIMutable`). Handoff is, at its core, flipping this marker in a place both sides respect.

### Why a single source of truth matters for handoff

The operator computes its prune set as `previous entries (status.inventory) - currently rendered entries`. If the CLI has already written the deployed entry set into `status.inventory`, the operator's first reconcile after takeover sees:

- a render whose digest matches `lastAppliedRenderDigest` written by the CLI → apply is a server-side-apply no-op;
- a stale set of zero → nothing pruned.

No resource is touched. That is the zero-downtime property, and it falls out of sharing the inventory rather than being engineered separately.

## Current State

### CLI (today)

| Concern | Implementation |
|---------|----------------|
| Inventory storage | Secret `opm.<releaseName>.<releaseID>`, key `inventory`, JSON `ReleaseInventoryRecord` (`internal/inventory/secret.go`, `types.go`) |
| Record contents | `CreatedBy`, release metadata, module metadata, `{Revision, Digest, Count, Entries[]}` |
| Entry identity / stale set | `pkg/inventory/` — `InventoryEntry`, `ComputeStaleSet`, component-aware identity |
| Apply flow | `internal/workflow/apply/apply.go` — load previous inventory, compute stale set, SSA apply in weight order, prune, write Secret |
| Prune safety | never prunes Namespaces; component-rename detection; pre-apply existence check on first apply |
| Ownership guard | `pkg/ownership/` — `CreatedBy: cli|controller`; CLI refuses to mutate controller-owned releases |
| Commands touching inventory | `release apply`, `release delete`, `release status`, `release list`, `release diff` (and `module apply` via the same workflow) |
| Operator/library deps | none — `go.mod` has no dependency on `opm-operator` or `library` |

### Operator (today)

| Concern | Implementation |
|---------|----------------|
| CRDs | `ModuleRelease`, `Release`, `BundleRelease`, `Platform` in group `releases.opmodel.dev/v1alpha1` |
| ModuleRelease spec | `suspend`, `module` (OCI module path + version), `values`, `prune`, `serviceAccountName`, `rollout` |
| ModuleRelease status | `observedGeneration`, `releaseUUID`, `conditions`, `lastAttempted*` / `lastApplied*` digests and timestamps, `failureCounters`, **`inventory`**, `history`, `nextRetryAt` |
| Inventory code | `internal/inventory/` — `NewEntryFromResource`, `IdentityEqual`, `K8sIdentityEqual`, `ComputeStaleSet`, `ComputeDigest`. **Internal: not importable by the CLI today.** |
| Apply | Flux SSA `ResourceManager.ApplyAllStaged`, field manager `opm-controller` |
| Prune | per-resource delete with ownership guard (managed-by label + UUID match); never deletes Namespaces or CRDs |
| Install artifacts | `config/crd/bases/*.yaml` (4 CRDs), `dist/install.yaml` (Namespace + CRDs + RBAC + Deployment), built by `task operator:installer` |
| Library dep | `github.com/open-platform-model/library` (kernel wired into the controller, not yet read on reconcile paths) |

## Design

### Overview

```text
  CLI-only phase                      Handoff                  Operator phase
  ==============                      =======                  ==============

  opm release apply               opm release handoff          operator reconciles
        |                                |                           |
        v                                v                           v
  +------------------+          +------------------+        +------------------+
  | ModuleRelease CR |          | verify published |        | reads status.    |
  |  spec: module,   |  ----->  | module == what   | -----> | inventory as     |
  |   values,        |          | is deployed,     |        | "previous";      |
  |   owner: cli     |          | then flip        |        | SSA apply is a   |
  |  status:         |          | owner -> operator|        | no-op (digests   |
  |   inventory      |          +------------------+        | match), stale    |
  |   digests        |                                      | set empty ->     |
  |   releaseUUID    |                                      | zero downtime    |
  +------------------+                                      +------------------+
        |
        v
  resources labeled
  managed-by: opm-cli
  uuid: <release-uuid>
```

### D1 — The CLI writes a ModuleRelease CR instead of a Secret

On `opm release apply`, the CLI:

1. Ensures the ModuleRelease CRD exists (see D5 for the failure/auto-install behavior).
2. Creates or updates a `ModuleRelease` object named after the release, in the release namespace, with the CLI-ownership marker set (D3).
3. Renders and applies resources exactly as today (the CLI keeps its own apply engine — see "Out of scope: apply-engine unification").
4. Writes the inventory into `status.inventory` via the status subresource, plus the small set of status fields the CLI is responsible for (D2).

`release delete` resolves inventory from the CR, deletes resources in reverse weight order, then deletes the CR last (mirroring today's "Secret deleted last" ordering). `release list` lists ModuleRelease objects instead of inventory Secrets. `release status` and `release diff` read `status.inventory`.

The CLI field manager for CR writes is `opm-cli` (distinct from the operator's `opm-controller`), so SSA field ownership cleanly identifies which actor wrote which fields, including status.

The packages `cli/internal/inventory` (Secret marshaling, CRUD) are deleted. `cli/pkg/inventory` (entry identity, stale set) is deleted in favor of the operator's exported equivalents (D4). Prune safety semantics that exist only in the CLI today (component-rename detection, pre-apply existence check) are preserved — either by porting them into the shared package or keeping them CLI-side on top of it; the OpenSpec slice decides, with a bias toward sharing (the operator benefits from them too).

### D2 — The CLI writes a strict subset of status ("simpler version of the CR")

The CLI is a one-shot actor with no reconcile loop, so most status fields are meaningless for it to write. The contract:

| Status field | CLI writes? | Rationale |
|--------------|-------------|-----------|
| `inventory` | yes | The point of this RFC. Same `{revision, digest, count, entries}` semantics as today's Secret: revision increments per successful apply. |
| `releaseUUID` | yes | Release identity; already stamped on resources as a label. |
| `lastAppliedRenderDigest`, `lastAppliedSourceDigest`, `lastAppliedConfigDigest`, `lastAppliedAt` | yes | The CLI already computes render digests. Writing them lets the operator's first post-handoff reconcile detect a no-op. |
| `conditions` | one condition only | A single `Ready` condition reflecting the last CLI apply outcome, with reason `AppliedByCLI`. No `Reconciling`/`Stalled`/`Drifted` — those describe a control loop that is not running. |
| `observedGeneration` | no | Controller-loop semantics. |
| `lastAttempted*` | no | Attempt/retry bookkeeping belongs to the controller. |
| `failureCounters`, `history`, `nextRetryAt` | no | Same. |

The operator MUST tolerate a ModuleRelease whose status contains only this subset (it already must tolerate empty status on a fresh CR; this is the same obligation).

### D3 — Explicit CLI-ownership marker; the operator skips CLI-owned CRs

A CLI-created ModuleRelease in a cluster where the operator runs would otherwise be reconciled immediately — the operator would attempt to resolve `spec.module`, fight the CLI for resources, and potentially prune. An explicit ownership marker is required.

**Decision: a spec-level field the operator respects.**

```yaml
apiVersion: releases.opmodel.dev/v1alpha1
kind: ModuleRelease
metadata:
  name: jellyfin
  namespace: media
spec:
  owner: cli            # new field: "cli" | "operator"; default "operator"
  module:
    path: example.com/modules/jellyfin
    version: 1.2.0      # best-effort when applying from a local path (see D6)
  values: { ... }
```

Operator behavior when `spec.owner: cli`:

- Skip render/apply/prune entirely.
- Set a single condition `Ready: Unknown` with reason `ManagedExternally`, message pointing at the CLI.
- Do not touch `status.inventory` or any CLI-written status field.

Alternatives rejected:

- **`spec.suspend: true` as the marker.** Conflates "paused" with "CLI-owned". A user running `kubectl edit` to unsuspend would trigger an accidental takeover with no verification. Suspend remains available and orthogonal.
- **Status-driven skip (`status.inventory.createdBy == cli`).** Driving controller behavior from status is fragile: status can be lost on backup/restore, and spec is the user-intent surface.

The existing `CreatedBy` concept from the CLI's Secret record maps onto this field; `pkg/ownership` semantics (`EnsureCLIMutable`) are preserved: the CLI refuses to apply/delete a release whose `spec.owner` is `operator`.

> **Operator-side change.** This field, its defaulting, the skip path, and the `ManagedExternally` condition are an `opm-operator` change, documented here per the cross-repo scope of this RFC.

### D4 — The operator exports its inventory package; the CLI imports it

`opm-operator/internal/inventory` moves to `opm-operator/pkg/inventory` (the move already anticipated by the archived `cli-dependency-and-inventory-bridge` design). Exported surface:

- `InventoryEntry` (already exported via `api/v1alpha1`)
- `NewEntryFromResource(*unstructured.Unstructured) InventoryEntry`
- `IdentityEqual`, `K8sIdentityEqual`
- `ComputeStaleSet(previous, current []InventoryEntry) []InventoryEntry`
- `ComputeDigest(entries []InventoryEntry) string`

The CLI adds `github.com/open-platform-model/opm-operator` to `go.mod` and imports:

- `api/v1alpha1` — the ModuleRelease type, Inventory, InventoryEntry
- `pkg/inventory` — the functions above
- `pkg/core` — label constants (replacing the CLI's parallel definitions where they overlap)

The import is intentionally narrow: API types plus pure functions, pulling in apimachinery but not controller-runtime's manager machinery or Flux SSA. The CLI does **not** import the operator's apply/prune engine (see Out of scope).

**Accepted consequence — dependency direction.** `cli -> opm-operator` couples CLI releases to operator API churn at `v1alpha1`. The alternative (move the types into `library`, both repos import from there) is cleaner long-term but fights Kubebuilder codegen for CRD types and adds a third repo to every schema change. We accept the coupling while the API is `v1alpha1` and both repos are pre-1.0; if churn becomes painful, revisit relocating the shared types.

### D5 — `opm install`: CRDs and operator via the CLI

Once inventory lives in a CR, the CRDs are a hard prerequisite for *every* CLI apply — including clusters where the user never intends to run the operator. The install command therefore has a CRD-only mode:

```bash
# CRDs only — required for any CLI apply; no controller deployed
opm install crds

# Full operator: namespace, CRDs, RBAC, deployment (dist/install.yaml)
opm install operator

# Remove the operator (deployment, RBAC, namespace). CRDs are intentionally left in place.
opm uninstall operator
```

`opm uninstall` never deletes CRDs. Deleting a CRD cascades to every `ModuleRelease`
object in the cluster — the entire CLI and operator inventory — so it is not exposed as a
CLI operation at all. Removing CRDs is a deliberate, manual cluster-admin action
(`kubectl delete crd ...`), kept off the CLI surface to make accidental data loss impossible.

Manifest sourcing: the CLI **embeds** the CRD manifests and `dist/install.yaml` of a pinned operator version at build time (`go:embed`), with a `--version` flag to fetch a different release from GitHub instead. Embedding keeps the learner path offline-friendly and makes "this CLI build is compatible with this CRD version" a build-time fact; fetching covers skew.

Apply-time behavior when the CRD is missing: `opm release apply` fails with a one-line hint (`ModuleRelease CRD not found — run 'opm install crds'`) rather than auto-installing. Installing CRDs is a cluster-admin write; doing it implicitly inside an application apply surprises exactly the operators who care. A `--install-crds` convenience flag on `apply` is open (OQ-2).

Upgrade semantics: `opm install crds` server-side-applies with field manager `opm-cli`, so re-running upgrades CRDs in place; the command warns when the cluster's CRD version is newer than the CLI's embedded copy.

### D6 — Spec contents when applying from a local module

The operator's `spec.module` is an OCI reference it resolves from a registry. A CLI user often applies from a **local directory** the operator can never fetch. Rules:

- When the CLI applies from a published module reference, it writes that reference into `spec.module` verbatim.
- When the CLI applies from a local path, it writes the module's declared path/version as best-effort metadata and records the source kind (e.g., an annotation `module-release.opmodel.dev/source: local`). The CR is a valid inventory store but **not yet reconcilable**.
- Handoff (D7) is the gate that guarantees `spec.module` is resolvable before the operator ever owns the CR.

This means a CLI-owned CR can temporarily describe a module the operator could not fetch. That is acceptable *because* of D3: the operator does not act on CLI-owned CRs, and D7 refuses to flip ownership until the spec is reconcilable.

### D7 — `opm release handoff` (the migration feature)

The capstone command. Preconditions, verified in order, each with an actionable error:

1. Operator is installed and the controller deployment is ready (else: `opm install operator`).
2. The release's ModuleRelease CR exists with `spec.owner: cli`.
3. `spec.module` is a published, registry-resolvable reference (else: publish the module and re-apply, or pass `--module <ref>` to set it now).
4. **Digest verification:** the CLI renders from the published reference + the CR's values and compares the render digest with `status.lastAppliedRenderDigest`. Mismatch aborts the handoff: what the operator would reconcile is not what is deployed. `--force` overrides with a diff display.

Then the flip:

5. Patch `spec.owner: operator` (single SSA patch, field manager `opm-cli`).
6. Wait (bounded) for the operator's first reconcile; report the outcome — expected: `Ready: True`, inventory revision incremented, zero resources changed, zero pruned.

After handoff the CLI's mutation guard makes the release read-only to the CLI (`status`, `diff`, `list` still work). A reverse handoff (`--to cli`) is the symmetric flip and is cheap to include since the verification machinery is identical.

The label transition (`app.kubernetes.io/managed-by: opm-cli -> opm-controller`) happens naturally on the operator's subsequent applies; the prune guards on both sides already accept both values, so the transition window is safe.

### D8 — Migration of existing Secret inventories

Existing CLI deployments have Secret-based inventories. Silent breakage here is the worst outcome (orphans that never get pruned). On `opm release apply` against a release that has a Secret inventory but no ModuleRelease CR:

1. Read the Secret, create the CR, write the Secret's record into `status.inventory` (preserving revision).
2. Proceed with the normal apply against the CR.
3. Delete the Secret only after the CR status write succeeds.

`release status`/`delete`/`list` fall back to reading Secrets for one transition period (one minor release), warning that the format is deprecated. After that, the Secret path is removed entirely.

## Out of Scope

- **Apply-engine unification.** The operator applies via Flux SSA `ApplyAllStaged` (field manager `opm-controller`); the CLI keeps its existing apply path (field manager `opm-cli`). True engine parity is not required for safe handoff: digest verification (D7.4) guarantees the operator's first apply is a no-op, and SSA field-manager transfer handles ownership of individual fields. Unifying the engines (likely by both consuming `library`) is future work.
- **`Release` / `BundleRelease` handoff.** This RFC covers `ModuleRelease` only. The Flux-sourced CRs have no CLI-side equivalent today.
- **Rollback / history.** `status.history` remains operator-only; the CLI does not gain rollback here (unchanged from RFC-0001's deferral).
- **Operator lifecycle management beyond install/uninstall** (upgrades orchestration, HA configuration) — `opm install` applies manifests; it is not a package manager.

## Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| CRD requirement raises the CLI's barrier to entry (today: works on any cluster) | medium | `opm install crds` is one command, embedded manifests work offline; clear error with hint on missing CRD |
| Operator API (`v1alpha1`) churn breaks CLI builds | medium | Narrow import surface (types + pure functions); both repos pre-1.0; accepted explicitly in D4 |
| User flips `spec.owner` by hand, skipping D7 verification | medium | Operator takeover still consults `status.inventory`, so pruning stays correct; worst case is a non-no-op first apply. Document that `handoff` is the supported path. |
| RBAC: CLI users now need permissions on `modulereleases` and `modulereleases/status` | low | Documented; `opm install crds` can optionally emit an RBAC snippet for non-admin CLI users |
| Field-manager conflicts on status between `opm-cli` and `opm-controller` | low | Disjoint field sets by contract (D2); SSA resolves the overlap (`inventory`, `lastApplied*`) by manager transfer on takeover |
| Two actors apply simultaneously during the transition window | low | Ownership guards on both sides (D3 + `EnsureCLIMutable`) make the window a spec-field race, resolved by SSA; the loser errors loudly |

## Open Questions

- **OQ-1:** Should the shared inventory package also absorb the CLI-only prune safety checks (component-rename detection, pre-apply existence check), making them operator behavior too? Bias: yes, but it widens the operator slice.
- **OQ-2:** Should `opm release apply` offer `--install-crds` to collapse the learner's first-run friction, given D5 decides against silent auto-install?
- **OQ-3:** Does `opm install operator` belong under a broader `opm install` umbrella that might later cover other cluster prerequisites, or should it be `opm operator install` to leave `install` free? (Pure CLI-surface taxonomy; decide in the slice.)
- **OQ-4:** Should this RFC be promoted to a workspace `enhancements/` umbrella entry (it is cross-repo by the workspace's own routing rules), with this document as its seed? The slices below work either way.
- **OQ-5:** `release list` across namespaces previously listed Secrets; listing CRs requires list permission on `modulereleases` cluster-wide for `--all-namespaces`. Acceptable, or keep a label-based fallback?

## Implementation Slices

Ordered; each is an independently shippable OpenSpec change in its target repo.

| # | Repo | Slice | Contents | Depends on |
|---|------|-------|----------|------------|
| 1 | `opm-operator` | `export-inventory-pkg` | Move `internal/inventory` -> `pkg/inventory`; no behavior change. Pre-designed by the archived bridge change. | — |
| 2 | `opm-operator` | `cli-ownership-marker` | `spec.owner` field, defaulting, skip-reconcile path, `ManagedExternally` condition, CRD regen. | — |
| 3 | `cli` | `operator-install-command` | `opm install crds|operator`, `opm uninstall`, embedded manifests, version pinning. | 2 (ships the CRD that includes `spec.owner`) |
| 4 | `cli` | `cr-inventory-backend` | Depend on operator module; ModuleRelease CR replaces Secret in apply/delete/status/list/diff; D2 status subset; D8 Secret migration; delete `internal/inventory` Secret code and `pkg/inventory` copy. | 1, 2, 3 |
| 5 | `cli` | `release-handoff` | `opm release handoff` with D7 verification and reverse mode. | 4 |

Slices 1–2 can proceed in parallel; slice 4 is the large one and should land behind passing e2e against a kind cluster with CRDs installed via slice 3.

## References

- RFC-0001 — Release Inventory (the Secret-based design this supersedes)
- `opm-operator/openspec/changes/archive/2026-04-12-01-cli-dependency-and-inventory-bridge/` — the deliberate copy of CLI inventory code into the operator, and its stated intent to export it later
- `opm-operator/api/v1alpha1/modulerelease_types.go`, `common_types.go` — CR and inventory types
- `opm-operator/internal/inventory/` — code to be promoted in slice 1
- `cli/internal/inventory/`, `cli/pkg/inventory/`, `cli/pkg/ownership/` — code retired by slice 4
- `opm-operator/dist/install.yaml`, `config/crd/bases/` — install artifacts consumed by slice 3
