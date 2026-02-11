## Why

The render pipeline's overlay generation uses `fmt.Sprintf` with embedded CUE source — fragile, no compile-time safety, and invisible to `go vet`. The pipeline also loads the module twice: once in Phase 2 (`extractModuleMetadata`) to get name/namespace for overlay construction, then again in Phase 3 (`ReleaseBuilder.Build`) with the overlay injected. Both issues were identified in `experiments/ast-pipeline/` which proved that CUE's AST APIs (`ast.NewStruct`, `ast.NewIdent`, `format.Node`, `load.Instances` with `inst.Files` inspection) eliminate both problems.

## What Changes

- **AST-based overlay generation**: Replace `fmt.Sprintf` template in `release_builder.go` with typed AST construction using `ast.NewStruct`, `ast.NewIdent`, `ast.NewString`, `ast.Interpolation`, and `astutil.Resolve`. The generated CUE is byte-identical to the current string template (proven by `TestOverlayAST_MatchesStringTemplate`). New imports: `cuelang.org/go/cue/ast`, `cuelang.org/go/cue/ast/astutil`, `cuelang.org/go/cue/token`.
- **Single-load metadata extraction**: Eliminate the Phase 2 `extractModuleMetadata` call and the Phase 3 `detectPackageName` call (which each do their own `load.Instances`). Replace with a single `load.Instances` call whose `inst.PkgName` provides the package name and whose `inst.Files` AST walk extracts `metadata.name` and `metadata.defaultNamespace` — no `BuildInstance` needed for metadata (proven by `TestSingleLoad_ASTInspectVsValueLookup`).
- **No API/flag/command changes**: `Pipeline`, `RenderResult`, `RenderOptions`, and all command interfaces remain unchanged. This is an internal refactor.

**SemVer: PATCH** — no user-facing API, flag, or behavioral changes.

## Capabilities

### New Capabilities

_None — this is a refactor, not a new capability._

### Modified Capabilities

- `render-pipeline`: Phase 2 (Metadata Extraction) changes from `BuildInstance` + `LookupPath` to AST inspection of `inst.Files` without CUE evaluation. Phase 3 (ReleaseBuilder) overlay generation changes from `fmt.Sprintf` to typed AST construction. Pipeline phases and their ordering are unchanged.
- `build`: ReleaseBuilder's `generateOverlayCUE` changes from string template to AST construction. `Build()` reduces from three `load.Instances` calls (metadata + package detection + overlay build) to two (metadata+package via AST + overlay build). No functional requirement changes.

## Impact

- **Affected packages**: `internal/build/release_builder.go` (primary — AST overlay generation replaces `fmt.Sprintf`), `internal/build/pipeline.go` (Phase 2 metadata extraction changes to AST walk, `detectPackageName` removed)
- **Affected commands**: None — all commands using the render pipeline get the improvement transparently.
- **New Go imports**: `cuelang.org/go/cue/ast`, `cuelang.org/go/cue/ast/astutil`, `cuelang.org/go/cue/token` (all part of the existing `cuelang.org/go` dependency — no new module dependencies).
- **Removed code**: `detectPackageName` helper, the `fmt.Sprintf`-based overlay template string, one `load.Instances` call in `extractModuleMetadata`.
- **Performance**: Eliminates one full `load.Instances` + `BuildInstance` call. AST construction and formatting is trivially fast compared to CUE evaluation.
- **Risk**: Low. The experiment proves byte-identical output. Existing tests cover the pipeline end-to-end. The overlay AST gotchas (identifier vs quoted labels, interpolation element format, `astutil.Resolve` for scope wiring) are documented and tested in `experiments/ast-pipeline/`.
