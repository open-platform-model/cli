## Context

`internal/releaseprocess` is the top-level orchestrator of the render pipeline. It calls `match.Match`, constructs renderers, and invokes them. After changes 1-4, all its dependencies are in `pkg/render/`. This is the final merge step — after this, the entire render pipeline is public.

## Goals / Non-Goals

**Goals:**
- Move `ProcessModuleRelease`, `ProcessBundleRelease`, `SynthesizeModule`, `ValidateConfig` to `pkg/render/`
- Move unexported helpers (`finalizeValue`, etc.)
- Fix the misplaced `pkg/loader/validate_test.go` and `pkg/loader/validate_diag_test.go` imports
- Update all 6 callers

**Non-Goals:**
- Change pipeline behavior
- Refactor `ValidateConfig` into a separate package

## Decisions

### Rename SynthesizeModuleRelease to SynthesizeModule
`render.SynthesizeModuleRelease` is redundant — `render.SynthesizeModule` is clear enough. `ProcessModuleRelease` and `ProcessBundleRelease` keep their full names as requested.

### Same-package simplification
`ProcessModuleRelease` currently calls `engine.NewModuleRenderer` and `match.Match`. After the merge, these are all same-package calls — no imports needed. Internal type references simplify: `*engine.ModuleRenderResult` → `*ModuleResult`, `*modulerelease.ModuleRelease` → `*ModuleRelease`.

### Fix pkg/loader test imports
`pkg/loader/validate_test.go` and `pkg/loader/validate_diag_test.go` import `internal/releaseprocess.ValidateConfig`. After the move, they import `pkg/render.ValidateConfig`. This is a clean fix — no test logic changes.

## Risks / Trade-offs

- **[Low risk]**: 6 callers to update. The intra-group references all collapse to same-package.
- **[Low risk]**: `finalizeValue` is an unexported helper with CUE AST manipulation. Moving it is mechanical — it has no external callers.
