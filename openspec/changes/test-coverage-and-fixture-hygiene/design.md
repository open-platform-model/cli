# Design: test-coverage-and-fixture-hygiene

## Context

The audit's cli findings split into "spec'd but untested" (annotation SSA-clear, migration failure/idempotence, gate ordering/dry-run, fallback banner) and "fixture debt" (two coexisting schema lineages, a sibling-checkout dependency in render-parity, sixfold helper duplication). The fake dynamic client cannot express SSA removal-on-omit — established by the `ssa-ownership` program's header — which is why the annotation and any omit-semantics coverage must be live; conversely the migration *failure* path is unreachable live (no clean way to make a real status write fail) and belongs to the fake client, whose reactor mechanism can inject exactly that failure.

## Goals / Non-Goals

**Goals:** close the four test gaps at their correct tiers; one schema lineage across maintained fixtures; render-parity self-contained in this repo.

**Non-Goals:** blueprint/primitive testing (enhancements#6 framework); the live refuse-when-newer ceiling e2e (rides the e2e suite against the dev operator — already tracked as C1 task 7.1's residual); rewriting the integration-program architecture (helper consolidation is optional, not a goal).

## Decisions

### LD1: Tier assignment follows what each harness can express

- **Live (kind)**: annotation write + SSA-clear-on-omit (ssa-ownership program — one new step pair using its existing live-object assertion helpers); migration idempotence (migration program — stage CR-present + leftover-Secret, run apply, assert single clean state after).
- **Unit (fake + reactor)**: migration failure — a `PrependReactor` on the status-subresource apply returning an error; assert the legacy Secret survives in the tracker and no delete action was recorded. This asserts the *ordering contract* (status success gates deletion) without pretending the fake models SSA.
- **Unit (probe recording)**: gate order — a fake client whose reactors append probe identities to a slice; assert exact sequence CRD-presence → field-floor → ceiling. Dry-run — `Execute` with dry-run against a fake with **no** CRD registered: success and zero recorded probes.
- **Unit (output capture)**: the fallback banner — run the apply-path platform resolution with no flag and a failing cluster getter, capturing the CLI's output sink; assert the D21 warning text (source provenance) is emitted. Uses/extends whatever capture hook `internal/output` already provides for tests; if none exists, add the minimal writer-swap seam (test-only).

### LD2: Fixture lineage — port when the target line exists, retire visibly when not

`tests/fixtures/valid/*` port to `core@v1` data-file form (the shape the handoff/inst-tree fixtures established); their vet tests keep asserting the same behaviors (valid module, secrets discovery, debug values). `examples/instances/jellyfin` ports (jellyfin@v2 is published on the v1 line); examples whose modules never moved lines (verify garage, mc_java_fleet against the registry) move to `examples/legacy/` with a README stating they document the retired v0 line — visible retirement, not deletion, since they remain the only worked examples of those modules. `examples/cue.mod` follows the port. `mc_router` (mod_build_test's registry-backed module) is verified against the current line and ported or the test's fixture swapped to a v1-line module.

### LD3: Vendor the podinfo fixture, note provenance

Copy `test/fixtures/modules/podinfo` from opm-operator into `cli/tests/fixtures/modules/podinfo` with a header comment naming the origin and that drift is acceptable — render-parity's correctness comes from comparing the CLI and kernel paths over the *same* fixture, not from matching the operator's copy byte-for-byte. `render-parity/main.go` drops the `../opm-operator` path.

### LD4: Helper consolidation is opportunistic

The six copy-pasted helper sets (`opmLabels`/`buildCM`/`writeInventoryCR`/wait helpers) consolidate into a shared non-ignored package under `tests/integration/` only if the `//go:build ignore` mains can import it without losing their run-directly property (they can — ignore-tagged mains may import normal packages). Attempt it; if any friction appears, skip without guilt — duplication is annoying, not dangerous.

## Risks / Trade-offs

- [Output-capture seam doesn't exist and adding one touches production code] → keep it to a test-only writer swap in `internal/output`; if it grows beyond trivial, split the banner test out rather than expanding scope.
- [Old-line example modules unresolvable during port verification] → that *is* the retirement signal; record per-example disposition in the tasks.
- [Vendored podinfo drifts from operator's copy] → accepted by design (LD3); parity compares paths, not repos.
