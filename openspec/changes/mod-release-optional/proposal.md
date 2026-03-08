## Why

`opm mod build` and `opm mod apply` fail when no `release.cue` exists in the module directory, even though the module defines `debugValues` for exactly this purpose. Module authors should be able to build and apply their module locally without writing a separate release file — `debugValues` already carries the contract for local testing.

## What Changes

- `opm mod build .` and `opm mod apply .` now work without a `release.cue` file
- When no `-f` flag is given, both commands default to `debugValues` from the module (consistent with how `opm mod vet` already behaves in release mode)
- When `release.cue` is absent and no `-f` is given and `debugValues` is not defined, a clear error is returned: `"no release.cue found — add debugValues to module or use -f <values-file>"`
- A new loader function `SynthesizeModuleRelease` constructs a `*modulerelease.ModuleRelease` from a module + values without a release file, by filling `#config` with values and exposing `#components` through the standard render pipeline
- `DebugValues: len(rf.Values) == 0` is set for both `build` and `apply`; `vet` retains existing behavior unchanged

## Capabilities

### New Capabilities

- `mod-release-optional`: `opm mod build` and `opm mod apply` work on a bare module directory (no `release.cue`), using `debugValues` or an explicit `-f` values file as the values source

### Modified Capabilities

- `build`: Default-to-`debugValues` behavior added; `values.cue` no longer required; no behavior change when `release.cue` is present and `-f` is given
- `release-building`: `RenderRelease` gains a synthesis path for the no-release-file case; value-fallback chain updated to include `debugValues` tier
- `loader-api`: `SynthesizeModuleRelease` added as a new exported function in `pkg/loader`
- `cmdutil`: `RenderRelease` orchestration updated to branch on `release.cue` presence; values-fallback documentation updated

## Impact

- **`pkg/loader/`** — new file `module_as_release.go` with `SynthesizeModuleRelease`
- **`internal/cmdutil/render.go`** — `RenderRelease` restructured with a synthesis branch; new `hasReleaseFile` helper
- **`internal/cmd/mod/build.go`** — `DebugValues: len(rf.Values) == 0` added to opts
- **`internal/cmd/mod/apply.go`** — `DebugValues: len(rf.Values) == 0` added to opts
- SemVer: **MINOR** (new capability, no breaking changes; existing flag/values workflows unchanged)
