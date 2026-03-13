## Why

The `internal/releaseprocess` package orchestrates the full render pipeline — config validation, CUE finalization, matching, and engine invocation. It has zero CLI dependencies and is the top-level entry point that both CLI and controller need. This is the final step in consolidating the render pipeline into `pkg/render/`.

## What Changes

- Move `ProcessModuleRelease`, `ProcessBundleRelease`, `SynthesizeModuleRelease`, and `ValidateConfig` from `internal/releaseprocess/` into `pkg/render/`
- **Rename**: `SynthesizeModuleRelease` → `SynthesizeModule`
- **Keep**: `ProcessModuleRelease` and `ProcessBundleRelease` names unchanged
- Move `finalizeValue` and related unexported helpers
- Fix misplaced tests: `pkg/loader/validate_test.go` and `pkg/loader/validate_diag_test.go` import `internal/releaseprocess.ValidateConfig` — update them to import from `pkg/render`
- Update all 6 callers

## Capabilities

### New Capabilities

### Modified Capabilities

- `module-release-processing`: The pipeline orchestration functions (`ProcessModuleRelease`, `ProcessBundleRelease`, `SynthesizeModule`, `ValidateConfig`) move from `internal/releaseprocess` to `pkg/render`.

## Impact

- **Files moved**: `internal/releaseprocess/*.go` → `pkg/render/process.go`, `pkg/render/synthesize.go`, `pkg/render/validate.go`, `pkg/render/finalize.go`
- **External callers updated** (6 files):
  - `internal/workflow/render/render.go`: `releaseprocess.SynthesizeModuleRelease` → `render.SynthesizeModule`, `releaseprocess.ProcessModuleRelease` → `render.ProcessModuleRelease`
  - `internal/workflow/render/render_test.go`: `releaseprocess.ValidateConfig` → `render.ValidateConfig`
  - `internal/workflow/render/validation_test.go`: same
  - `internal/cmd/module/vet.go`: `releaseprocess.ValidateConfig` → `render.ValidateConfig`
  - `pkg/loader/validate_test.go`: `releaseprocess.ValidateConfig` → `render.ValidateConfig`
  - `pkg/loader/validate_diag_test.go`: same
- **SemVer**: MINOR (new public API surface)
