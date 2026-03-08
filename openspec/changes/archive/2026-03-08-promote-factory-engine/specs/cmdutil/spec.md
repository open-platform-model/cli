## MODIFIED Requirements

### Requirement: RenderRelease orchestration
`cmdutil.RenderRelease()` SHALL orchestrate the new rendering pipeline: call `loader.LoadReleasePackage()`, `loader.DetectReleaseKind()`, `loader.LoadModuleReleaseFromValue()` (or bundle equivalent), then `engine.ModuleRenderer.Render()` (or `BundleRenderer.Render()`). It SHALL convert `[]*core.Resource` to `[]*unstructured.Unstructured` at this layer before passing to downstream packages.

#### Scenario: Module release rendering
- **WHEN** `RenderRelease()` is called for a ModuleRelease
- **THEN** it loads via `pkg/loader`, renders via `pkg/engine`, converts resources to Unstructured, and returns results compatible with `internal/kubernetes/` and `internal/inventory/`

#### Scenario: CUE boundary enforcement
- **WHEN** `RenderRelease()` passes resources to `internal/kubernetes/` or `internal/inventory/`
- **THEN** resources are `[]*unstructured.Unstructured` — no CUE types cross this boundary

### Requirement: Values file resolution stays in cmdutil
Values file resolution (--values CLI flags with fallback to values.cue) SHALL remain in `internal/cmdutil/` as a CLI-layer concern. The resolved file path is passed to `loader.LoadReleasePackage()`.

#### Scenario: Values flag resolution
- **WHEN** the user provides `--values custom-values.cue`
- **THEN** cmdutil resolves the path and passes it to `LoadReleasePackage(cueCtx, releaseFile, resolvedValuesFile)`

#### Scenario: Default values fallback
- **WHEN** no --values flag is provided
- **THEN** cmdutil passes empty string to `LoadReleasePackage()`, which defaults to `values.cue` in the release directory
