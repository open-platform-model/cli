## Why

Delete `tests/fixtures/valid/secrets-module`. It is built on the legacy catalog secret mechanism (`res.#Secret` with `$secretName` / `$dataKey`) that `catalog_opm` is removing (`catalog-remove-legacy-secrets`, shipping as `opmodel.dev/catalogs/opm@v3`).

Nothing reads it: no Go test or program (the vet tests read `simple-module` only), and it sits outside the published-fixture flow (`tests/fixtures/modules`, `hack/fixtures.sh`, PR CI). A secrets fixture returns with enhancement 0013's `#Secret` and the CLI's secrets surface (`opm secrets template`, secrets-aware `opm module vet`).

## What Changes

1. Delete `tests/fixtures/valid/secrets-module/` (`module.cue`, `cue.mod/module.cue`).
2. The `test-fixture-lineage` capability stops naming "secrets discovery" as a behaviour the vet fixtures exercise. The valid-module and debug-values fixtures stay.
3. No Go change. No CI change. Release class `test(fixtures)`: hidden from the changelog, no release.

Not in scope: `openspec/specs/auto-secrets-injection/spec.md` (already superseded; only a removed requirement), the removed-requirement reason in `openspec/specs/build/spec.md`, the archived `2026-02-28-cli-auto-secrets` change that created the fixture, and the design history that describes the legacy secret model (`docs/rfc/0002`, `0003`, `0005`, `adr/006`). All stay as history. The `library` doc comment at `opm/helper/synth/instance.go:134` still describing `opm-secrets` injection is a `library` concern.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `test-fixture-lineage`: the "Vet fixtures exercise current schema" scenario of "Maintained fixtures track the current schema line" no longer lists secrets discovery among the behaviours the ported fixtures cover.

## Impact

1. Files removed: `tests/fixtures/valid/secrets-module/module.cue`, `tests/fixtures/valid/secrets-module/cue.mod/module.cue`.
2. Tests: none reference the fixture; `go test ./internal/cmd/module/...` is unaffected.
3. Registry: the fixture declared `example.com/modules/secrets_module@v0` and was never published. Nothing on GHCR changes.
4. Downstream: none. `opm-operator` carries no equivalent fixture.
5. No enhancement decision is implemented here; the removal follows `enhancements/0013` D9 landing in `catalog_opm`, recorded there.
