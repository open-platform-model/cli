# Proposal: cli-instance-handoff

Enhancement 0006, slice C3 — the final feature slice. Implements the handoff decisions of enhancement 0006: the gate side, the amended inventory-stable verdict, forward-only handoff, the thin-editor mode with delete symmetry (deferred from C1) and the read-side annotation gate. Scope widened from the plan's handoff-only row by user decision (2026-07-18): C3 also replaces the C1 refusal arm with enhancement 0006's operator-owned handling.

## Why

Everything 0006 built so far converges here: the CLI and operator share one inventory store (C1), one render pipeline (C2), and one ownership marker (A4) — but there is still no safe way to move a release from CLI management to operator management. The manual path (edit `spec.owner` by hand) is exactly the unverified flip the enhancement exists to eliminate: nothing checks that the operator can resolve the module, that the published module matches what is deployed, or that the operator's takeover is non-destructive. And after any handoff, the CLI goes dark — C1's guard refuses operator-owned instances outright, so the tool that deployed a release cannot even update its values once the operator owns it.

## What Changes

- **New `opm instance handoff <name>`** — forward-only (CLI → operator), no reverse mode (enhancement 0006). Verified precondition chain, in order: operator installed and ready (reusing B2's readiness machinery); CR exists with `spec.owner: cli`; **fail-closed local-provenance gate** — refuse while the CR carries `module-instance.opmodel.dev/source: local`, remedy named (enhancement 0006); `spec.module` resolves **strictly from the registry** — the verification path bypasses `cue.mod/local-module.cue` and the registry-blind CUE cache (enhancement 0006); verification render of the resolved module + the CR's `spec.values` against the **cluster Platform** (forced, no `--platform` — enhancement 0006) with the CLI's own runtime name, whose digest must equal the CR's `status.lastAppliedRenderDigest` — mismatch aborts, `--force` overrides with the mismatch shown (enhancement 0006).
- **Then the flip**: one SSA patch of `spec.owner: operator` (manager `opm-cli`), followed by a bounded wait for the operator's first reconcile, judged by **enhancement 0006's inventory-stable criterion** — `Ready: True`, `status.inventory` entry set unchanged, revision incremented, zero pruned — read from CR status only. The managed-by relabel is expected and reported ("operator adopted N resources — managed-by relabel, no workload changes"), never a failure. **No cross-actor digest comparison anywhere** (enhancement 0006).
- **enhancement 0006 thin-editor mode replaces the C1 refusal arm on `apply`**: against an operator-owned instance, `opm instance apply` unifies values and SSA-patches `spec.module` + `spec.values` only, then waits bounded for the operator's reconcile and reports its status. No render-and-apply of resources, no prune, no inventory/status writes, no status-RBAC pre-flight (enhancement 0006). A module that would resolve from local bytes is refused in this mode — the operator cannot fetch a local checkout.
- **Delete symmetry decided and shipped (enhancement 0006's open half)**: `opm instance delete` against an operator-owned instance deletes the CR and lets the operator's `opmodel.dev/cleanup` finalizer prune the workloads, with a bounded wait reporting completion — but refuses when the operator is not ready, because deleting a finalizer-armed CR with no controller running wedges it (the B2 uninstall footgun, now guarded on the other side).
- The version-skew ceiling gate (enhancement 0006) applies to all three new paths; handoff additionally hard-requires cluster `Platform` read access (it is the one admin-adjacent operation — enhancement 0006).

## Capabilities

### New Capabilities

- `instance-handoff`: the `opm instance handoff` command — precondition chain, local-provenance gate, strict-registry verification render, ownership flip, and the inventory-stable reconcile wait/report.

### Modified Capabilities

- `inventory-ownership`: the operator-owned refusal requirements are replaced by enhancement 0006's dual-mode contract — thin-editor `apply`, finalizer-delegating `delete` with operator-readiness guard; the mode-resolution branch point gains its second arm.

## Impact

- **Packages**: new `internal/cmd/instance/handoff.go` (thin cobra) + handoff workflow (likely `internal/workflow/handoff/`); `internal/workflow/apply` (thin-editor branch); `internal/cmd/instance/delete.go` (operator-owned path); `internal/inventory` (spec-patch helpers, condition/observedGeneration reads); reuse of `internal/operator` readiness waits and the C2 render/acquisition path for the strict-registry verification render.
- **SemVer**: MINOR (new command, new behavior on previously-refused paths; no breaking surface).
- **Dependencies**: C1 ✅, C2 ✅ (verified against origin/main: `apply.go` runs `RunClusterGates` + `ApplySpec`/`ApplyStatus`, `inventory.ComputeRenderDigest` defined, Secret backend deleted), A6 ✅ (operator v1.0.0-alpha.3+; embedded pin at v1.0.0-alpha.4). No open gates. (A transient false blocker recorded here on 2026-07-18 was an artifact of a corrupted local git clone presenting a stale working tree; retracted after verification against the remote.)
- **Out of scope**: reverse handoff (enhancement 0006 — permanently, absent a future enhancement); operator-side changes (the enhancement 0006 prune-guard tolerance check is a read-only verification task here — if it finds a real operator gap, that spawns an opm-operator slice, not scope creep here).
