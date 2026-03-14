## Context

The `--module` flag on release render commands (`vet`, `build`, `apply`, `diff`) injects a local module directory into a release file's `#module` field via CUE `FillPath`. This was introduced as a development convenience for authors who haven't published their module to a registry yet.

In practice, this flag creates the most complex mutation path in the release pipeline. The `FromReleaseFile` function in `internal/workflow/render/render.go` has a ~20-line block that: validates the module path, calls `LoadModulePackage`, overwrites `rel.RawCUE` via `FillPath`, overwrites `rel.Module.Raw`, `rel.Module.Config`, and `rel.Config`, and re-decodes module metadata. These 4 field mutations on a partially-constructed `*module.Release` are the primary source of complexity that the upcoming `release-pipeline-simplification` change must work around.

Removing the flag eliminates this mutation path entirely. Module resolution becomes a single path: CUE imports in the release file. The `#module` filled check becomes a simple gate with no fallback.

## Goals / Non-Goals

**Goals:**

- Remove the `--module` flag from all four release render commands
- Remove `Module` field from `ReleaseFileFlags` and `ModulePath` from `ReleaseFileOpts`
- Remove the `--module` injection branch from `FromReleaseFile`
- Simplify the `#module` filled check to a hard error with no fallback
- Remove `--module` from command help examples
- Update tests that exercise the `--module` path
- Update `LoadReleaseFile` and `LoadModulePackage` doc comments

**Non-Goals:**

- Do not remove `LoadModulePackage` from `pkg/loader` — it is used by `opm module vet` for loading modules from a directory path
- Do not change `opm module` commands — they use module paths as positional arguments, not `--module`
- Do not change the release-file loading or rendering pipeline itself — only remove the `--module` injection branch
- Do not change CUE import resolution behavior

## Decisions

### Decision 1: Hard error when `#module` is not filled

When `#module` is not concrete after loading the release file, the CLI returns a clear error directing the user to import a module in their release file. No fallback mechanism.

**Rationale**: A single module resolution path (CUE imports) is simpler than two paths. Users who develop modules locally can use a local CUE registry or a relative import path in their `cue.mod/module.cue`.

**Error message**: `"#module is not filled in the release file — import a module to fill it"`

### Decision 2: Keep `LoadModulePackage` in `pkg/loader`

The function stays because `internal/cmd/module/vet.go` uses it to load a module from a directory for module-only validation. Only the `--module` comment on the function doc is updated.

**Rationale**: `LoadModulePackage` is a general-purpose loader, not tied to the `--module` flag. Its only consumer after this change is `opm module vet`.

### Decision 3: Restructure e2e test to use CUE import

The e2e test `TestE2E_ReleaseVet_Output` currently uses `--module` to inject a module. After removal, the test fixture needs a `cue.mod/` directory and a CUE import to fill `#module`.

**Rationale**: The test should exercise the only remaining module resolution path. This makes the test more representative of real usage.

### Decision 4: Remove `--module` from `ReleaseFileFlags`, not just hide it

Complete removal rather than deprecation. The CLI is pre-1.0 so there is no backward-compatibility obligation for flags.

**Rationale**: Deprecation adds complexity (warning messages, flag hiding). Pre-1.0 means breaking changes are expected.

## Risks / Trade-offs

**[Risk: Users relying on `--module` for local development]** → Low. The CLI is pre-1.0 with a small user base. CUE's native module resolution (local registry, relative imports) provides the same capability. The error message directs users to the import path.

**[Risk: E2e test restructure breaks CI]** → Medium. The e2e test needs a working CUE module import in its fixture directory. This requires adding `cue.mod/` with module declaration and import path to the test fixture. Must verify the test passes before merging.

**[Trade-off: Less convenient local development]** → Accepted. Module authors now must set up CUE imports rather than using `--module ./path`. This is the standard CUE workflow and aligns with how modules will work in production.
