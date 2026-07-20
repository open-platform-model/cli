# Design: cli-instance-handoff

Enhancement 0006, slice C3. Local decisions are `LD n`; enhancement decisions are cited as `0006/DN`.

## Context

C1 made the `ModuleInstance` CR the shared inventory store with `spec.owner` and the `module-instance.opmodel.dev/source: local` provenance annotation; C2 made both actors render through the `library` kernel (byte-identical modulo the runtime-name label — the `tests/integration/render-parity` result), gave the CLI cluster-`Platform` resolution, and left `inventory.ComputeRenderDigest` as the shared digest algorithm. B2 built the operator readiness machinery (`internal/operator`: CRDs `Established` + Deployment rollout waits) explicitly for reuse here (0006/D35). A4's operator skips CLI-owned CRs and falls through to a normal reconcile when `spec.owner` flips — the operator side of handoff has been live since June. What's missing is the verified flip itself, and a CLI that stays useful after it (C1's guard refuses operator-owned instances outright).

0006/D40 (the C2 relabel discovery) reframed what "successful handoff" observably means; 0006/D38 added the local-provenance gate and the strict-registry verification-render requirement. Both are binding inputs here.

## Goals / Non-Goals

**Goals:**

- `opm instance handoff <name>`: precondition chain → single owner flip → bounded inventory-stable-reconcile wait with honest relabel reporting.
- D18 dual-mode: thin-editor `apply` (spec patch + wait + report) and finalizer-delegating `delete` for operator-owned instances, replacing C1's refusal arms at the existing mode-resolution branch point.
- All waits share one bounded reconcile-observation helper (built once, used by handoff, thin-editor apply, and operator-owned delete).

**Non-Goals:**

- Reverse handoff (0006/D16) — no flag, no machinery, no partial support.
- Operator-side changes. The 0006/D40 prune-guard question is a read-only verification task; a discovered gap becomes an opm-operator slice.
- Any cross-actor digest comparison (0006/D40) — explicitly designed out, not merely omitted.

## Decisions

### LD1: One reconcile-observation helper, three consumers

A single bounded-wait helper observes an instance's reconcile from CR status: it records the CR's `metadata.generation` after the triggering write, then polls until `status.observedGeneration >= generation` and a `Ready` condition for that generation exists (or timeout). Consumers interpret the outcome differently: handoff applies the 0006/D40 inventory-stable criteria (Ready `True` ∧ entry set unchanged ∧ revision incremented ∧ nothing pruned); thin-editor apply reports whatever `Ready` says; operator-owned delete waits for CR disappearance instead of a condition (finalizer completion = NotFound). Poll-based (no watch): the CLI's dynamic client usage stays list/get-only, intervals ~2s, default `--timeout` 5m matching the B2 install waits.

*Alternative — a watch-based informer:* rejected; drags in cache machinery for a one-shot CLI wait that polling serves fine.

### LD2: Handoff precondition order is cheapest-first, cluster-reads before renders

(1) operator ready — reuses `internal/operator`'s readiness checks (0006/D35 built them for this); (2) CR exists, `spec.owner: cli` via the C1 mode resolver; (3) D38 annotation gate — a metadata read, before any expensive work; (4) strict-registry module resolution (LD3) — fails fast on an unpublished reference before rendering; (5) verification render + digest self-comparison against `status.lastAppliedRenderDigest`. `--force` bypasses gate 5 only, printing both digests; gates 1–4 have no override (an unresolvable module or local provenance makes the flip *unsafe by construction*, not merely unverified). The cluster `Platform` is the only platform source — `--platform` is rejected (0006/D11), and a failure to read the cluster Platform is a hard error here (handoff is the one admin-adjacent path, 0006/D17).

### LD3: Strict-registry resolution = isolated acquisition, fresh CUE cache

The verification render MUST reproduce what the *operator* will fetch (0006/D38). Mechanics: acquire the module by `spec.module.path@version` through the same registry-acquisition call sequence the operator uses (the C2 render-parity test's second path), executed in an isolated module context — no working-directory module discovery, so no `cue.mod/local-module.cue` can inject a replacement — with a fresh per-invocation `CUE_CACHE_DIR` (tempdir, cleaned after), closing the registry-blind-cache hole. Cost: one full module download per handoff; accepted — handoff is a rare, deliberate operation. The render passes the CLI's own `RuntimeName` (`internal/workflow/render/kernel.go`) so the digest is comparable with the CLI-recorded one, and computes it via the same `inventory.ComputeRenderDigest`.

*Alternative — reuse the shared cache with a version-pin check:* rejected; the cache is keyed by `module@version` and registry-blind, so a republished same-version artifact silently satisfies the check with stale bytes — the exact trap D38 names.

### LD4: The flip changes one field, but the document is complete

**Corrected 2026-07-20 during verification; the original decision was wrong and would have shipped a broken command.** It read:

> One SSA apply document: `apiVersion`/`kind`/`metadata.name`/`namespace` + `spec.owner: operator`, manager `opm-cli` — a dedicated minimal store helper, since `inventory.ApplySpec` builds the full spec document (owner + module + values unconditionally) and is unsuitable for a single-field flip.

That inverts server-side-apply semantics. A field manager's applied document is its **complete declared intent**: a field the manager previously owned but now omits is *released*, and — since no other manager claims these fields — pruned from the live object. `opm-cli` owns `spec.owner`, `spec.module` and `spec.values` from the preceding `ApplySpec`, so a document carrying only `spec.owner` deletes the other two. Verified against a live API server:

```
$ kubectl apply --server-side --field-manager=opm-cli -f owner-only.yaml
The ModuleInstance "ssa-probe" is invalid: spec.module: Required value
```

`spec.module` is a required CRD field, so the API server rejects the whole patch — meaning the original design fails **every** handoff, on every cluster. (`spec.values` is not required, so the same mechanism deletes it silently.)

The premise was backwards in the other direction too: having the *same* manager restate a field's unchanged value is a no-op — it neither changes the value nor bumps `metadata.generation`. What risks clobbering is a *different* manager writing a field with `Force: true`, which is a separate concern.

**The decision, corrected:** the flip is a single `inventory.ApplySpec` carrying the complete CLI-owned spec — `spec.owner: operator` plus the instance's *current* `spec.module` and `spec.values`, read from the record the gates already fetched. One field changes; the rest are restated so they survive. There is no separate minimal store helper: targeted single-field writes are not expressible with this field manager, so `ApplySpec` is the single writer for the CLI-owned spec. SSA ownership of `spec.owner` still stays with `opm-cli` (what would let a future enhancement flip it back deliberately), and the operator's fall-through reconcile (A4) still sees exactly one *changed* spec field.

The provenance annotation is still omitted, which clears any stale value — safe here because gate 3 already refused a local-provenance instance, so it is absent.

*Why the original survived to implementation:* client-go's fake dynamic client does not implement apply-patch merge semantics, so unit tests asserting the minimal payload's shape passed against a bug a real API server rejects. `tests/integration/ssa-ownership/main.go` now covers this against a live cluster.

### LD4a: The verified record is re-read immediately before the flip

**Added 2026-07-20 during verification.** Gates 4-5 download and render a module, which takes real wall-clock time, and `spec.owner` is still `cli` for all of it — so a concurrent `opm instance apply` (CLI-executor mode, entirely legitimate) can move `spec.module`/`spec.values` inside the verification window.

That window is not benign. `opm-cli` is the sole SSA writer for those fields and applies with `Force`, so a flip built from the pre-verification record restates the *stale* values and silently reverts the concurrent apply while handing ownership away. LD5's verdict cannot catch it: the D40 baseline is that same stale snapshot, so the operator's reconcile of the reverted spec matches it and the command reports success. Silent state loss reported as a clean handoff, on a path with no reverse mode.

**The decision:** re-read the record immediately before the flip and compare `metadata.generation` — the API server bumps it on spec changes only, which is exactly the class of write that invalidates the verification. On a mismatch, abort with the two generations and leave ownership with the CLI; the D40 baseline is taken from the re-read.

Refuse rather than repair. Adopting the fresh module and values would flip a document nothing verified — gate 5 proved digest parity against the spec as it stood when verification began, so silently rolling forward defeats the gate it just passed. Re-running handoff re-verifies, which is the correct and cheap remedy.

This is detect-and-retry, not atomicity: a change landing between the re-read and the apply microseconds later still wins. Closing that fully needs an SSA `resourceVersion` precondition, which the inventory store does not currently express for any writer. Accepted — the residual window is orders of magnitude smaller than the render window it replaces, and the failure mode returns to "last writer wins" rather than "silent revert reported as success".

### LD5: D40 verdict computation reads status only

Success = `Ready: True` (for the post-flip generation) ∧ `status.inventory.entries` as a *set* equals the pre-flip snapshot (order-insensitive; entries are identity tuples) ∧ `revision` strictly greater ∧ no entry lost (the zero-pruned observable). The relabel is reported from the entry count ("operator adopted N resources — managed-by relabel only, no workload changes"). Failure/timeout exits non-zero with the operator's condition message and an explicit statement that ownership remains `operator` (0006/D16 — the CLI must not auto-revert; the error names `kubectl` and operator logs as the investigation path).

### LD6: Thin-editor apply branches at the C1 resolver, shares the C1 write shapes

The mode resolver's operator-owned arm (C1 left it as refusal) becomes: unify values exactly as CLI-executor mode does → SSA-apply the CLI-owned spec with `spec.module` + `spec.values` updated → LD1 wait → report `Ready`.

**Corrected 2026-07-20 alongside LD4.** This originally specified a "new edit-shaped store writer (module + values only; never `spec.owner`)", reasoning that an apply carrying an empty owner would seize the field and clobber `operator`. The instinct was right; the remedy was not. *Omitting* `spec.owner` does not leave it alone — it releases `opm-cli`'s claim and the API server prunes the marker, which is the very outcome the thin editor exists to avoid. Verified live: an apply of module-only against a CR with owner + values left `{"module":{…}}` and nothing else, exit 0, no error.

The correct move is to restate the owner **the CLI just read from the live CR** (`rec.Owner`). Writing back a value unchanged is a no-op; it is omission that deletes. An empty `rec.Owner` is passed through as omission, which is correct in that one case: the CRD reads an absent owner as operator-managed, so leaving it absent preserves the meaning, and writing `""` would fail the enum. Skipped machinery, per 0006/D18/D23: render, resource apply, prune, inventory/status writes, the status-RBAC pre-flight. Still active: the D24 ceiling gate (an old CLI writing spec for a newer operator is the unsafe skew direction) and the CRD-presence gate. Local-source modules are refused in this mode before any write — the render-provenance detection C1 built runs on the loaded module even though no render follows.

### LD7: Operator-owned delete = readiness guard, then CR delete, then wait-for-absence

Deleting a finalizer-armed CR with no running controller wedges it in `Terminating` — the same footgun B2's uninstall guards from the other side. So: operator ready (else refuse, naming the wedge and `opm operator install` / `--remove-finalizers` as remedies) → delete the CR → LD1 wait for NotFound → report. No `--force` bypass of the readiness guard; the existing delete confirmation flow applies unchanged.

### LD8: Command wiring

`internal/cmd/instance/handoff.go` stays thin cobra; orchestration in a new `internal/workflow/handoff` package following the existing workflow-package pattern. The thin-editor and delete arms live in the existing `internal/workflow/apply` / delete paths, branching at the resolver per C1's design promise ("C3 replaces the refusal arm without touching callers").

## Risks / Trade-offs

- [Operator prune guard may not tolerate still-`opm-cli`-labeled resources in the removed-resource + not-yet-relabeled window (0006/D40 drafting note)] → dedicated read-only verification task against `opm-operator`'s `apply.Prune` matching rule; a real gap spawns an operator slice and gets a risk note in 0006, not a workaround here.
- [Fresh-cache verification render needs registry reachability and adds seconds of latency] → inherent to the safety property (D38); the failure mode is a clear resolution error, and handoff is rare.
- [Post-flip failure leaves the instance operator-owned with no CLI-native undo] → by design (D16); LD5 mandates the error message say so explicitly with the manual investigation path. The alternative (auto-revert) reintroduces the reverse-handoff design surface D16 excluded.
- [Thin-editor apply against a suspended or crash-looping operator waits full timeout] → LD1 reports last observed condition on timeout rather than a bare deadline error.
- [Entry-set equality could mask an operator rendering *different but same-identity* resources] → out of scope by 0006/D40's rationale: D7.4's digest gate already proved content parity pre-flip; identity-set stability is the correct post-flip observable.

## Migration Plan

No migration — additive command plus behavior changes on paths that previously hard-refused. Docs: new `handoff` reference page, updates to `apply`/`delete` for the operator-owned behaviors, QUICKSTART gains the graduate-to-operator walkthrough (install operator → handoff → verify). Rollback = previous CLI binary (refusal semantics return; no persisted state depends on the new code).

## Open Questions

None blocking. The D40 prune-guard verification (task-tracked) is the one item that can produce follow-up work outside this repo.
