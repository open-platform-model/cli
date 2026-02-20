## Why

`core.ModuleRelease` is built by a procedural `Builder.Build()` in `internal/build/release/` but its two validation steps — checking user values against the module schema and checking the full release is concrete — are standalone functions with no clear ownership. Promoting them to receiver methods on `core.ModuleRelease` makes the BUILD phase of the pipeline self-evident and keeps validation logic co-located with the type it validates.

## What Changes

- Add `ValidateValues() error` receiver method to `core.ModuleRelease` — validates user-supplied `Values` against `Module.Config` schema using CUE field walking (moves logic from `release/validation.go:validateValuesAgainstConfig`)
- Add `Validate() error` receiver method to `core.ModuleRelease` — validates that all components are concrete and the release is ready for matching (moves logic from `release/builder.go` validation steps)
- Update `internal/build/release/` builder to return `*core.ModuleRelease` (currently returns `BuiltRelease`, a build-internal type)
- `build/pipeline.go` calls `rel.ValidateValues()` then `rel.Validate()` after `release.Build()` returns

## Capabilities

### New Capabilities

- `module-release-receiver-methods`: Receiver methods on `core.ModuleRelease` for values validation and release readiness validation

### Modified Capabilities

- `render-pipeline`: The BUILD phase of `pipeline.Render()` now delegates validation to receiver methods on `core.ModuleRelease` rather than inline calls to standalone functions

## Impact

- `internal/core/module_release.go` — new `ValidateValues()` and `Validate()` methods; these carry `cue.Value` fields so no external CUE context is needed for the field-walking validation
- `internal/build/release/builder.go` — `Build()` returns `*core.ModuleRelease` instead of `*BuiltRelease`; internal `BuiltRelease` type removed or collapsed
- `internal/build/release/validation.go` — `validateValuesAgainstConfig` and component concreteness check move to core receiver methods; file may be removed or reduced
- `internal/build/pipeline.go` — BUILD phase updated to call receiver methods
- SemVer: **PATCH** — internal refactor, no change to CLI behavior or public-facing pipeline interface
