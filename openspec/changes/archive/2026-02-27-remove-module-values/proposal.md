## Why

The CUE catalog has moved from v0 to v1alpha1. In v0, `#Module` had a `values` field for author-provided defaults and `#config` held constraints only. In v1alpha1, defaults now live inside `#config` (e.g., `port: int | *8096`), the `values` field on `#Module` has been replaced by `debugValues` (test/debug only), and `values.cue` provides concrete author defaults as a standalone file. The CLI's loader and builder still implement the v0 model — loading `Module.Values` from inline `values` fields (Pattern B) or from `values.cue` (Pattern A), and passing `Module.Values` through to the builder. This needs to align with the v1alpha1 schema.

## What Changes

- **BREAKING**: Remove `Module.Values` field from the Go `Module` struct. The loader no longer extracts or stores default values.
- **BREAKING**: Remove `Module.HasValuesCue` and `Module.SkippedValuesFiles` fields — these were loader concerns that move to the builder.
- Remove Pattern B (inline `values` in `module.cue`) from the loader. Only Pattern A file filtering remains (values*.cue must still be excluded from the CUE package load because `#Module` is a closed definition).
- Move all values resolution logic to the builder: the builder discovers `values.cue` from `mod.ModulePath` when no `--values` flag is provided, or loads explicit `--values` files. No values source means an error — `values.cue` or `--values` is always required.
- Simplify `pipeline.prepare()` — remove values debug logging (moves to builder).
- Update all test fixtures and tests to match the new model.

## Capabilities

### New Capabilities

_None — this is a refactor of existing capabilities._

### Modified Capabilities

- `module-loading`: Remove the requirement for extracting `Module.Values`. Loader no longer loads values; it only filters `values*.cue` from the package load.
- `core-module`: Remove `Values`, `HasValuesCue`, `SkippedValuesFiles` fields from the `Module` struct.
- `release-building`: Value selection no longer reads `mod.Values`. Builder discovers `values.cue` from `mod.ModulePath` or uses `--values` files. `values.cue` or `--values` is a hard requirement.

## Impact

- **Packages**: `internal/core/module`, `internal/loader`, `internal/builder`, `internal/pipeline`
- **Tests**: Loader tests (remove Pattern A/B value assertions), builder tests (add values.cue discovery tests), integration test `values-flow` (remove `mod.Values` assertions)
- **Test fixtures**: Remove `inline-values-module/`, update others
- **SemVer**: This is internal refactoring — no public CLI flag changes. PATCH level unless we consider `Module` struct a public API (it's in `internal/`), in which case still PATCH.
