## Why

`tests/fixtures/modules/podinfo` carries the two-part workaround the fleet just retired: `identity/identity.cue` declares `Version: #VersionType | *"0.1.5"` and `module.cue` interpolates `version: "\(id.Version)"` so the kernel's shape gate sees a concrete string. The fixture is the reference module the CLI's e2e tests build against and the shape new modules are compared with; it should carry the form the CLI now emits (sibling change `scaffold-identity-literal`), a plain literal referenced plainly.

## What Changes

- `podinfo/identity/identity.cue`: `Version: "0.1.6"` (the edit is a republish, so the version advances), no local `#VersionType`.
- `podinfo/module.cue`: `version: id.Version`, interpolation comment removed.
- Consumers re-pinned to `0.1.6`: `tests/e2e/testdata/handoff/cue.mod/module.cue`, `examples/cue.mod/module.cue` (the workspace root `task deps:pins:fixtures` does this sweep; run it or edit the two pins by hand, since this bump has no core/catalog dep change).
- `tests/fixtures/fixtures.go`: `Load` no longer falls back to `Default()`; a non-concrete `Version` is an error, as at build time. `tests/fixtures/fixtures_test.go`: `TestIdentityIsLiteral` scans every fixture's `identity.cue` for the literal form. Both are byte-identical copies with `opm-operator`, so the same bytes land there in its own `fixtures-identity-literal` change (`task fixtures:lint` stays green). `hack/fixtures.sh` untouched.

Not in scope: any Go code outside `tests/fixtures`; the operator's fixtures (their own change in `opm-operator`).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `test-fixture-lineage`: "Maintained fixtures track the current schema line" gains the requirement that fixture identity is a plain literal and loads through the kernel without a metadata workaround.

## Impact

- Files: `tests/fixtures/modules/podinfo/{identity/identity.cue,module.cue}`, `tests/fixtures/{fixtures.go,fixtures_test.go}` (mirrored in `opm-operator`), two `cue.mod` pins.
- CI: `pr.yml` `fixtures` job seeds `0.1.6` into the job-local registry and runs render parity against it; `hack/fixtures.sh check` requires the new version not to exist on GHCR (it does not). On merge `publish-fixtures.yml` publishes `0.1.6`.
- SemVer: none (`test(fixtures)`, no release).
- Enhancement: none.
