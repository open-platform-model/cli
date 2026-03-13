## MODIFIED Requirements

### Requirement: RenderRelease orchestration

`cmdutil.RenderRelease()` SHALL use the release-file loading path exclusively. There is no synthesis branch.

**When `release.cue` is present** (unchanged):
- Call `loader.LoadReleasePackage()`, `loader.DetectReleaseKind()`, `loader.LoadModuleReleaseFromValue()` (or bundle equivalent), then `engine.ModuleRenderer.Render()`.

In all cases, resources are converted to `[]*unstructured.Unstructured` before passing to downstream packages. No CUE types cross this boundary.

#### Scenario: CUE boundary enforcement (unchanged)
- **WHEN** `RenderRelease()` passes resources to `internal/kubernetes/` or `internal/inventory/`
- **THEN** resources are `[]*unstructured.Unstructured` — no CUE types cross this boundary

### Requirement: Values file resolution stays in cmdutil

Values file resolution SHALL remain in `internal/cmdutil/` as a CLI-layer concern. With the synthesis path removed, the resolution simplifies:

- When `--values` files are provided: pass them to `LoadReleasePackage`.
- When no `--values` files are provided: pass empty string to `LoadReleasePackage()`, which defaults to `values.cue` in the release directory (existing behavior).

#### Scenario: Values flag resolution (unchanged)
- **WHEN** the user provides `--values custom-values.cue`
- **THEN** cmdutil resolves the path and passes it to `LoadReleasePackage`

#### Scenario: Default values fallback with release.cue present
- **WHEN** no `--values` flag is provided
- **AND** `release.cue` is present
- **THEN** cmdutil passes empty string to `LoadReleasePackage()`, which defaults to `values.cue` in the release directory

## REMOVED Requirements

### Requirement: RenderRelease supports synthesis mode when release.cue is absent

**Reason**: The synthesis branch that detected the absence of `release.cue` and called `SynthesizeModuleRelease` is removed along with the `FromModule` workflow entrypoint. The `hasReleaseFile` detection helper, `DebugValues` option, and `RenderReleaseOpts` synthesis fields are also removed.

**Migration**: Always provide a `release.cue` file. Use `opm release build -r <release-file>` for rendering.

### Requirement: Refactored mod commands preserve exact behavioral equivalence (mod build scenarios)

**Reason**: The `opm mod build` command is removed entirely. The scenarios for "mod build output is identical after refactoring" no longer apply. The `opm mod apply` scenario is also removed as that command no longer exists.

**Migration**: Use `opm release build -r <release-file>` instead of `opm mod build`.
