# Tasks: test-coverage-and-fixture-hygiene

## 1. Live coverage additions (kind + CRDs; existing integration tier)

- [ ] 1.1 `tests/integration/ssa-ownership/main.go`: new step pair — `ApplySpec` with `SourceLocal: true` → assert `module-instance.opmodel.dev/source: local` on the live object; re-apply with `false` → assert annotation removed by SSA (closes the open C1 VERIFICATION-NOTES item; 0006 D38 gate input)
- [ ] 1.2 `tests/integration/migration/main.go`: idempotence scenario — CR present + leftover legacy Secret → apply → Secret deleted without being read as inventory, no duplicate migration, single clean end state

## 2. Unit coverage additions

- [ ] 2.1 Migration failure ordering: fake-client `PrependReactor` fails the status-subresource apply → assert legacy Secret retained and no delete action recorded (delete-after-status contract)
- [ ] 2.2 `RunClusterGates` probe-order assertion (CRD presence → field floor → ceiling) via reactor-recorded probe sequence
- [ ] 2.3 Dry-run exemption: dry-run `Execute` against a CRD-less fake → succeeds with zero gate probes recorded
- [ ] 2.4 Platform-fallback banner: capture output during apply-path resolution with failing cluster getter → assert the D21 provenance warning is emitted (add a test-only writer seam in `internal/output` only if none exists; keep minimal)

## 3. Fixture hygiene

- [ ] 3.1 Vendor `podinfo` module fixture from opm-operator into `tests/fixtures/modules/podinfo` with provenance header; point `tests/integration/render-parity/main.go` at it (drop the `../opm-operator` path)
- [ ] 3.2 Port `tests/fixtures/valid/{simple-module,secrets-module,module-with-debug-values}` to the `core@v1` line; vet tests keep asserting the same behaviors
- [ ] 3.3 Examples disposition: port `examples/instances/jellyfin` to `core@v1` `#ModuleInstance` form; verify garage / mc_java_fleet module availability on the v1 line — port where available, else move to `examples/legacy/` with a README naming the retired line; `examples/cue.mod` follows
- [ ] 3.4 Verify `examples/modules/mc_router` (mod_build_test's fixture) against the current line; port or swap the test to a v1-line module
- [ ] 3.5 Lineage grep gate: no `core/v1alpha1` / `modulerelease@v1` / `opm/v1alpha1` imports outside `examples/legacy/`
- [ ] 3.6 OPTIONAL: consolidate the duplicated integration-program helpers into a shared package under `tests/integration/` (skip without guilt on any friction)

## 4. Verification

- [ ] 4.1 `task check` green; `task test:integration` green on kind (ssa-ownership + migration new steps executing); render-parity runs standalone
- [ ] 4.2 Sync/archive per openspec flow
