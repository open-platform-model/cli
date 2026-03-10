## MODIFIED Requirements

### Requirement: RenderRelease orchestration

Supersedes the main spec requirement `"RenderRelease orchestration"`. The `RenderRelease` function now has two paths depending on whether `release.cue` exists.

`cmdutil.RenderRelease()` SHALL inspect whether `release.cue` is present in the module path and branch accordingly:

**When `release.cue` is present** (existing behavior, unchanged):
- Call `loader.LoadReleasePackage()`, `loader.DetectReleaseKind()`, `loader.LoadModuleReleaseFromValue()` (or bundle equivalent), then `engine.ModuleRenderer.Render()`.

**When `release.cue` is absent** (synthesis path):
- Load the module package directly.
- Extract `debugValues` when `DebugValues: true` and no `-f` flag, or load the `-f` values file.
- Resolve the release name from `opts.ReleaseName` → `module.metadata.name` → `filepath.Base(modulePath)`.
- Call `loader.SynthesizeModuleRelease()` to build the `*ModuleRelease`.
- Continue on the common tail: provider loading, engine rendering, resource conversion.

In both paths, resources are converted to `[]*unstructured.Unstructured` before passing to downstream packages. No CUE types cross this boundary.

#### Scenario: RenderRelease takes synthesis path when no release.cue
- **WHEN** `RenderRelease` is called with a module path that has no `release.cue`
- **AND** `DebugValues: true`
- **THEN** `RenderRelease` SHALL call `loader.SynthesizeModuleRelease` instead of `loader.LoadReleasePackage`
- **AND** the returned `*RenderResult` SHALL be populated identically to a release-backed render

#### Scenario: RenderRelease takes normal path when release.cue present
- **WHEN** `RenderRelease` is called with a module path that has a `release.cue`
- **THEN** `RenderRelease` SHALL use the existing `LoadReleasePackage` path
- **AND** behavior SHALL be unchanged from before this change

#### Scenario: CUE boundary enforcement (unchanged)
- **WHEN** `RenderRelease()` passes resources to `internal/kubernetes/` or `internal/inventory/`
- **THEN** resources are `[]*unstructured.Unstructured` — no CUE types cross this boundary

### Requirement: Values file resolution stays in cmdutil

Supersedes the `"Default values fallback"` scenario in the main spec.

Values file resolution SHALL remain in `internal/cmdutil/` as a CLI-layer concern. The resolution logic now accounts for the synthesis path:

- When `--values` files are provided: pass them to the appropriate loader function regardless of whether `release.cue` exists.
- When no `--values` files are provided and `release.cue` is present: pass empty string to `LoadReleasePackage()`, which defaults to `values.cue` in the release directory (existing behavior).
- When no `--values` files are provided and `release.cue` is absent: set `DebugValues: true` in `RenderReleaseOpts`, which causes the synthesis path to extract `debugValues` from the module.

#### Scenario: Values flag resolution (unchanged)
- **WHEN** the user provides `--values custom-values.cue`
- **THEN** cmdutil resolves the path and passes it to the appropriate loader (release or synthesis path)

#### Scenario: Default values fallback with release.cue present
- **WHEN** no `--values` flag is provided
- **AND** `release.cue` is present
- **THEN** cmdutil passes empty string to `LoadReleasePackage()`, which defaults to `values.cue` in the release directory

#### Scenario: Default values fallback without release.cue
- **WHEN** no `--values` flag is provided
- **AND** no `release.cue` is present
- **THEN** cmdutil sets `DebugValues: true` in `RenderReleaseOpts`
- **AND** the synthesis path extracts `debugValues` from the module as the values source
