# Tasks: cli-instance-handoff

## 1. Reconcile-observation helper (design LD1)

- [x] 1.1 Implement the bounded reconcile wait in a shared home (poll `status.observedGeneration >= generation` + `Ready` condition for that generation; NotFound mode for delete; last-observed-condition on timeout); unit tests with fake dynamic objects
- [x] 1.2 Add CR `status.conditions` reading to `internal/inventory` (Ready extraction by generation) — nothing reads conditions today

## 2. Handoff workflow (design LD2–LD5)

- [x] 2.1 New `internal/workflow/handoff` package: precondition chain — operator readiness via `operator.Wait`/predicates (`internal/operator/wait.go:74`), CR + `spec.owner: cli` via `inventory.GetRecord`/`ResolveOwnership`, D38 annotation gate (`inventory.AnnotationSource`)
- [x] 2.2 Strict-registry verification render: `kernel.AcquireModuleFromRegistry` + `SynthesizeInstance` + `Compile` with `render.RuntimeName` (the render-parity Path B sequence), isolated from CWD module context, fresh per-invocation `CUE_CACHE_DIR` (tempdir, cleaned); cluster `Platform` forced (non-nil `ClusterSpecGetter`, no flag source, hard error on read failure)
- [x] 2.3 Digest self-comparison against `record.LastAppliedRenderDigest`; `--force` bypasses only this gate, printing both digests
- [x] 2.4 The flip: single SSA apply changing `spec.owner: operator` (manager `opm-cli`) — **corrected 2026-07-20**: the document carries the current `spec.module`/`spec.values` too, because SSA prunes fields this manager owns but omits (the original 'minimal document' design failed every handoff — `spec.module` is required). `ApplySpec` is the single spec writer; the minimal helpers were removed. Covered live by `tests/integration/ssa-ownership`
- [x] 2.5 D40 verdict: pre-flip inventory snapshot → LD1 wait → judge Ready=True ∧ entry-set equality (order-insensitive) ∧ revision incremented; report "operator adopted N resources — managed-by relabel, no workload changes"; on failure/timeout exit non-zero with the operator condition and the no-reverse statement
- [x] 2.6 `internal/cmd/instance/handoff.go` thin cobra command (`--namespace`, `--timeout` default 5m, `--force`; `--platform` rejected), registered in `instance.go`

## 3. Thin-editor apply (design LD6)

- [x] 3.1 Replace the operator-owned refusal arm in the apply workflow: unify values → refuse local-source modules (loader provenance detection, `pkg/loader/provenance.go`) → SSA-patch `spec.module` + `spec.values` via a new edit-shaped store writer (module+values only; never `spec.owner`/annotation — `ApplySpec` writes owner unconditionally) → LD1 wait → report operator `Ready`
- [x] 3.2 Confirm skip set: no render-apply, no prune, no status writes, no status-SSAR; ceiling + CRD gates still run; unit + integration coverage for both skips and gates

## 4. Operator-owned delete (design LD7)

- [x] 4.1 Replace the delete refusal arm: operator-readiness guard (refuse with wedge explanation + remedies when not ready) → delete CR → LD1 wait-for-absence → report; existing confirmation flow unchanged
- [x] 4.2 Tests: refusal when operator absent; successful delegation with finalizer cleanup observed

## 5. Verification

> Re-verified 2026-07-20 after the verification-pass changes (LD4a pre-flip re-read, the thin-editor `resolveThinEditRef` extraction, the dry-run ownership probe, and the centralized `inventory.ResolveTimeout`) — all of which touch the paths 5.2/5.3 cover, so the earlier green was stale. Full `task test:e2e` green in 163s against the live kind operator (`TestE2E_Handoff_Adoption` re-run on its own to confirm, since the first log was tail-truncated); the two SKIPs are unrelated and pre-existing (`ModBuild_FromExampleModule` missing fixture, `ModInit_ThenVet` builder TODO). `task test:integration` also green, including `ssa-ownership` (flip preserves module+values, no-op does not bump generation, thin editor preserves owner) and `render-parity` (byte-identical across local staging and registry acquisition).

- [x] 5.1 D40 prune-guard check (read-only, opm-operator repo): confirm `apply.Prune`'s ownership matching tolerates resources still labeled `opm-cli` post-flip; record the finding in 0006 (gap ⇒ new operator slice, not scope here)
- [x] 5.2 e2e (kind + operator installed): apply CLI-owned → handoff → assert inventory-stable reconcile, relabel visible, workloads never restarted (pod UIDs stable across handoff) — `TestE2E_Handoff_Adoption`, green against a live reconciling operator brought up by `task cluster:operator`
- [x] 5.3 e2e negative paths — `TestE2E_Handoff_DigestGate` (mismatch aborts showing both digests + `--force` override), `TestE2E_Handoff_PreconditionRefusals` (local-provenance refusal, already-operator-owned, `--platform` rejection, not-found), `TestE2E_Delete_OperatorOwnedDelegates` (both the orphan and prune outcomes), `TestE2E_ThinEditor_ValuesRoundTrip` (values update post-handoff, owner preserved, operator scales the Deployment). All green against a live operator
- [~] 5.4 `task check` **NOT green** — `task lint` fails with pre-existing `goconst` findings that also fail on `main` (verified by stashing: 86 on this branch vs 87 on `main`); this change adds zero new lint findings and removes one, and `task fmt`/`task vet`/unit/e2e are all green. Fixing the baseline is out of scope for this slice. Docs — the handoff reference, the operator-owned `apply`/`delete` contract, and the graduate-to-operator walkthrough all live in `README.md` ("CLI-managed vs operator-managed instances" + "Graduating an instance to the operator"); `QUICKSTART.md` carries a short "Graduating to the Operator" section that points there rather than duplicating it, since divergent copies of the `spec.prune` and forward-only semantics would be worse than one home for them
- [x] 5.5 Record the landing in `enhancements/0006/config.yaml` history (slice C3); with C3 and A5 done, evaluate promoting 0006 to `implemented`
