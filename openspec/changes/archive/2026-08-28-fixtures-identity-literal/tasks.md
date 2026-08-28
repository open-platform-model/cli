## 1. tests/fixtures/modules/podinfo

- [x] 1.1 `identity/identity.cue`: `Version: "0.1.5"` literal, drop `#VersionType` and its comment; keep the hand-managed note.
- [x] 1.2 `module.cue`: `version: id.Version`, drop the interpolation comment.
- [x] 1.3 `opm module version set 0.1.6 ./tests/fixtures/modules/podinfo`.

## 1b. Shared fixture reader (mirrored in opm-operator)

- [x] 1.4 `tests/fixtures/fixtures.go`: drop the `Default()` fallback in `Load`; comment names the literal form.
- [x] 1.5 `tests/fixtures/fixtures_test.go`: `TestIdentityIsLiteral` checks every fixture identity for a plain literal and no `#VersionType`.

## 2. Consumers

- [x] 2.1 Re-pin `tests/e2e/testdata/handoff/cue.mod/module.cue` and `examples/cue.mod/module.cue` to `v0.1.6` (root `task deps:pins:fixtures` or by hand).

## 3. Validation gates

- [x] 3.1 `hack/fixtures.sh check` passes (gates plus "version not yet on GHCR").
- [x] 3.2 `task test:fixtures` (seed + render parity) passes.
- [x] 3.3 `opm module build ./tests/fixtures/modules/podinfo --name podinfo -n default` passes the loader gate.
- [x] 3.4 From the workspace root, `task fixtures:lint` stays green (shared-file edits applied to both copies).
