## Why

The `--module` flag on release commands (`vet`, `build`, `apply`, `diff`) allows injecting a local module directory into a release file via `FillPath`. This creates the most complex mutation path in the release pipeline — 4 field mutations on a partially-constructed `*module.Release` before rendering even begins. Removing it eliminates this complexity, simplifies the upcoming release-pipeline-simplification refactor, and enforces a single module resolution path: CUE imports. The flag was a development convenience, not a production necessity.

## What Changes

- **BREAKING**: Remove `--module <path>` flag from `opm release vet`, `opm release build`, `opm release apply`, and `opm release diff`.
- Remove `Module` field from `ReleaseFileFlags` in `internal/cmdutil/flags.go`.
- Remove `ModulePath` field from `ReleaseFileOpts` in `internal/workflow/render/types.go`.
- Remove the entire `--module` injection branch in `FromReleaseFile` (`internal/workflow/render/render.go`), including the `LoadModulePackage` call, `FillPath` mutations, and metadata re-decode.
- Simplify the `#module` filled check: if `#module` is not concrete after loading the release file, error with a message directing the user to import a module (no `--module` fallback).
- Remove `--module` examples from command help text in all four release commands.
- Keep `LoadModulePackage` in `pkg/loader` — it is still used by `opm module vet`.
- Update or remove tests that exercise `--module`: unit tests for flag registration, workflow test with `ModulePath`, and the e2e `vet_output_test.go` that uses `--module` to inject a module.
- Update `LoadReleaseFile` doc comment to remove `--module` reference.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `rel-commands`: Remove the requirement that release render commands accept `--module` flag. Remove scenarios for `--module` usage, missing `--module`, and `--module` with `BundleRelease`.
- `release-file-loading`: Remove the requirement for module injection via `--module` flag and `LoadModulePackage` + `FillPath`. Remove scenarios for `FillPath` injection and `--module` precedence. Update the `#module` not-filled error message.
- `release-building`: Remove the `--module` override scenario from release building. Update `LoadModulePackage` requirement description to remove `--module` reference (function still exists for `opm module vet`).

## Impact

- **`internal/cmdutil/flags.go`**: `ReleaseFileFlags` loses `Module` field and `--module` flag registration.
- **`internal/workflow/render/types.go`**: `ReleaseFileOpts` loses `ModulePath` field.
- **`internal/workflow/render/render.go`**: ~25 lines removed (the `--module` injection block), `#module` check simplified.
- **`internal/cmd/release/{vet,build,apply,diff}.go`**: Remove `ModulePath: rff.Module` from opts, remove `--module` from help examples.
- **`internal/cmd/release/release_test.go`**: Remove `--module` flag registration assertions.
- **`internal/workflow/render/render_test.go`**: Remove or update test using `ModulePath`.
- **`tests/e2e/vet_output_test.go`**: Restructure e2e test to use CUE import instead of `--module`.
- **`pkg/loader/release_file.go`**: Doc comment update only. `LoadModulePackage` stays.
- **SemVer**: MINOR — `--module` flag is removed but the CLI is pre-1.0.
- **Parallel change**: The `release-pipeline-simplification` change should be updated to remove `--module` references from its design, tasks, and specs after this change lands.
