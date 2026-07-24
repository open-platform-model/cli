# Proposal: test-coverage-and-fixture-hygiene

Test and fixture change from the 2026-07-21 workspace fixture audit. No product behavior changes.

## Why

Four spec'd behaviors lack the test that proves them, and the fixture tree carries two schema lineages plus a cross-repo dependency:

1. **Provenance annotation SSA-clear is still fake-client-only.** `instance-inventory`'s "Registry apply clears the annotation" scenario is asserted only against the fake dynamic client (`patch_test.go`), which — by the `ssa-ownership` program's own header — cannot model server-side-apply's removal-on-omit. This annotation is a fail-closed handoff gate input (0006 D38); C1's VERIFICATION-NOTES flagged the gap and it remains open.
2. **Migration failure/idempotence are spec'd, untested.** `secret-inventory-migration`'s "Failure preserves the Secret" and "Idempotent re-run after partial migration" scenarios have no tests; delete-after-status ordering is the migration's entire safety property and only its happy path runs.
3. **Gate ordering, dry-run exemption, and the platform-fallback warning banner** are spec'd (`apply-preflight-gates`, `platform-resolution`) but only implicit in source — no test pins the order, the dry-run skip, or that apply actually emits the D21 provenance warning.
4. **Fixture lineage drift**: `examples/instances/*` still import the retired `modulerelease@v1`/`core/v1alpha1` line; `tests/fixtures/valid/*` pin it too; `render-parity` loads its module fixture from the **sibling opm-operator checkout** (`../opm-operator/test/fixtures/modules/podinfo`), breaking standalone clones; six integration programs copy-paste the same helper set.

## What Changes

- **ssa-ownership program** gains the annotation step: `ApplySpec` with `SourceLocal: true` → live annotation present; re-apply `false` → live annotation removed by SSA.
- **Migration coverage**: unit test with a fake-client reactor forcing the status write to fail → legacy Secret retained (delete never issued); live idempotence scenario in the migration program (CR exists + leftover Secret → normal apply, Secret cleaned, no duplicate migration).
- **Unit assertions**: `RunClusterGates` probe order (CRD presence → field floor → ceiling); dry-run apply performs zero gate probes and succeeds against a CRD-less cluster; apply emits the local-default fallback warning banner.
- **Fixture hygiene**: port `tests/fixtures/valid/*` and portable `examples/instances/*` to the `core@v1` line (retire unportable examples to a marked legacy location); vendor the podinfo module fixture into `cli/tests/fixtures/` and point `render-parity` at it; verify `examples/modules/mc_router` (used by `mod_build_test`) against the current line. Optional: consolidate the duplicated integration-program helpers.

## Capabilities

### New Capabilities

- `test-fixture-lineage`: maintained fixtures and examples track the current published schema line, and repo tests depend only on repo-local fixtures.

### Modified Capabilities

None — every test addition implements scenarios already present in `instance-inventory`, `secret-inventory-migration`, `apply-preflight-gates`, and `platform-resolution`.

## Impact

- **Packages**: `tests/integration/{ssa-ownership,migration}/main.go`, `internal/workflow/apply` tests, `internal/platform`/output-capture tests, `tests/fixtures/`, `examples/`, `tests/integration/render-parity/main.go`.
- **SemVer**: none. Commit types `test:` / `chore:`.
- **Dependencies**: annotation + migration-idempotence steps need kind + CRDs (existing integration tier); everything else pure unit.
