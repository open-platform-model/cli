# Tasks: test-coverage-and-fixture-hygiene

## 1. Live coverage additions (kind + CRDs; existing integration tier)

- [x] 1.1 `tests/integration/ssa-ownership/main.go`: new step pair — `ApplySpec` with `SourceLocal: true` → assert `module-instance.opmodel.dev/source: local` on the live object; re-apply with `false` → assert annotation removed by SSA (closes the open C1 VERIFICATION-NOTES item; 0006 D38 gate input)
- [x] 1.2 `tests/integration/migration/main.go`: idempotence scenario — CR present + leftover legacy Secret → apply → Secret deleted without being read as inventory, no duplicate migration, single clean end state

## 2. Unit coverage additions

- [x] 2.1 Migration failure ordering: fake-client `PrependReactor` fails the status-subresource apply → assert legacy Secret retained and no delete action recorded (delete-after-status contract)
- [x] 2.2 `RunClusterGates` probe-order assertion (CRD presence → field floor → ceiling) via reactor-recorded probe sequence
- [x] 2.3 Dry-run exemption: dry-run `Execute` against a CRD-less fake → succeeds with zero gate probes recorded
- [x] 2.4 Platform-fallback banner: capture output during apply-path resolution with failing cluster getter → assert the D21 provenance warning is emitted (add a test-only writer seam in `internal/output` only if none exists; keep minimal)

## 3. Fixture hygiene

- [x] 3.1 Vendor `podinfo` module fixture from opm-operator into `tests/fixtures/modules/podinfo` with provenance header; point `tests/integration/render-parity/main.go` at it (drop the `../opm-operator` path)
- [x] 3.2 Port `tests/fixtures/valid/{simple-module,secrets-module,module-with-debug-values}` to the `core@v1` line; vet tests keep asserting the same behaviors
- [x] 3.3 Examples disposition: replaced the branded instance examples (garage/jellyfin/mc_java_fleet — real homelab modules, and none published on the `core@v1` line) with neutral `core@v1` `#ModuleInstance` examples importing the published test modules `opmodel.dev/modules/test/{hello-web,podinfo}@v0` (matching opm-operator's fixture convention); `examples/cue.mod` updated + tidied. Deviation from design LD2 — see design.md note.
- [x] 3.4 `examples/modules/mc_router` did not exist and mc_router is `core/v1alpha1`-only; swapped `mod_build_test`'s fixture to the vendored `core@v1` `tests/fixtures/modules/podinfo`
- [x] 3.5 Lineage grep gate: no `core/v1alpha1` / `modulerelease@v1` / `opm/v1alpha1` imports outside `examples/legacy/` (no legacy dir needed — branded examples deleted, not retired)
- [ ] 3.6 OPTIONAL: consolidate the duplicated integration-program helpers into a shared package under `tests/integration/` (skip without guilt on any friction)

## 4. Verification

- [x] 4.1 `task fmt`/`vet` green; `task test:unit` green; new lint issues: none (pre-existing `goconst` debt in untouched files remains); `render-parity` runs standalone (both paths byte-identical); `hello-web` example builds end-to-end; ssa-ownership (steps 5–6 annotation SSA-clear) and migration (step 5 idempotence) both PASS live on a provisioned `kind-opm-dev` cluster (CRDs-only install)
- [x] 4.2 Sync/archive per openspec flow (test-fixture-lineage synced to openspec/specs/; change archived)
