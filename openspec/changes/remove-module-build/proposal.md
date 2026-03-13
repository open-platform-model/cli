## Why

The `opm module build` command maintains a separate render path (Path A: pure module source, Path B: module dir with sibling `release.cue`) that duplicates and diverges from the release-file render path. Path A uses `SynthesizeModule` which redundantly validates config and finalizes components before `ProcessModuleRelease` does the same work again. Path B has different values-precedence rules than Path A, creating subtle inconsistencies. Removing this command simplifies the render pipeline, eliminates dead-end code paths, and focuses rendering on the single well-defined `opm release build/apply` flow.

## What Changes

- **BREAKING**: Remove the `opm module build` subcommand entirely. Users must use `opm release build -r <release-file>` instead.
- Remove the `FromModule` workflow function and all supporting code (Path A and Path B preparation logic).
- Remove `SynthesizeModule` from `pkg/render/` — its only caller is `FromModule`.
- Remove `ReleaseOpts` type, `hasReleaseFile` helper, and `loadModuleReleaseForRender` helper.
- Remove associated tests.
- Keep `opm module init` and `opm module vet` untouched.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `cmd-structure`: The `opm module` command group loses the `build` subcommand.
- `render-pipeline`: The `FromModule` entrypoint and its two preparation paths (Path A, Path B) are removed. Only `FromReleaseFile` remains.
- `release-building`: This becomes the sole way to render modules. No behavioral changes to this path.

## Impact

- **Commands**: `opm module build` removed. `opm module init`, `opm module vet`, and all `opm release` commands unaffected.
- **Packages**:
  - `internal/cmd/module/` — `build.go` and `build_test.go` deleted, `mod.go` edited.
  - `internal/workflow/render/` — `render.go`, `types.go`, `values.go`, `render_test.go` edited to remove module-build-specific code.
  - `pkg/render/` — `synthesize.go` and `synthesize_test.go` deleted.
- **SemVer**: MAJOR (removes a user-facing command).
- **Dependencies**: No new dependencies. Reduces import surface.
