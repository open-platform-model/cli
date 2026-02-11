## 1. AST Overlay Generation (`internal/build/release_builder.go`)

- [x] 1.1 Add imports: `cuelang.org/go/cue/ast`, `cuelang.org/go/cue/ast/astutil`, `cuelang.org/go/cue/token`, `cuelang.org/go/cue/format` to `release_builder.go`.
- [x] 1.2 Implement `generateOverlayAST(pkgName string, opts ReleaseOptions) *ast.File` method on `ReleaseBuilder`. Port the `generateOverlayAST` function from `experiments/ast-pipeline/overlay_test.go` — build `#opmReleaseMeta` as `*ast.File` with `ast.NewStruct`, `ast.NewIdent` for scoped labels, `ast.NewString` for special-char labels, `ast.Interpolation` for the uuid argument, and `ast.NewCall` for `uuid.SHA1`. Call `astutil.Resolve(file, errFn)` before returning.
- [x] 1.3 Update `Build()` to call `generateOverlayAST` instead of `generateOverlayCUE`. Format the `*ast.File` to bytes via `format.Node` and pass to `load.Config.Overlay` as `load.FromBytes`.
- [x] 1.4 Remove the old `generateOverlayCUE(pkgName string, opts ReleaseOptions) []byte` method.
- [x] 1.5 Remove the `"fmt"` import if it's no longer used after removing `generateOverlayCUE` (check other usages first).

## 2. Module Inspection via AST (`internal/build/release_builder.go`)

- [x] 2.1 Add `PkgName string` field to `ReleaseOptions` struct with a comment marking it as internal (set by `InspectModule`).
- [x] 2.2 Implement `InspectModule(modulePath string) (*ModuleInspection, error)` method on `ReleaseBuilder`. This does a single `load.Instances(["."])`with `&load.Config{Dir: modulePath}` (setting `CUE_REGISTRY` if configured), reads `inst.PkgName`, and walks `inst.Files` to extract `metadata.name` and `metadata.defaultNamespace` as string literals. Return a `ModuleInspection` struct with `Name`, `DefaultNamespace`, and `PkgName` fields.
- [x] 2.3 Implement the AST walk helper `extractMetadataFromAST(files []*ast.File) (name, defaultNamespace string)`. Walk each file's top-level declarations looking for a `metadata` struct field, then within it look for `name` and `defaultNamespace` fields with `*ast.BasicLit` string values. Strip quotes from the literal value. Stop at first match per field.
- [x] 2.4 Update `Build()` to use `opts.PkgName` when set (skipping `detectPackageName`). If `opts.PkgName` is empty, fall back to calling `detectPackageName` for backward compatibility.
- [x] 2.5 Demote `detectPackageName` to backward-compatibility fallback — retained for direct `Build()` callers that don't use `InspectModule`. Pipeline path passes `PkgName` via `InspectModule`.

## 3. Pipeline Integration (`internal/build/pipeline.go`)

- [x] 3.1 Update `pipeline.Render()` to call `p.releaseBuilder.InspectModule(modulePath)` instead of `p.extractModuleMetadata(modulePath, opts)`. Map `ModuleInspection.Name` → `moduleMeta.name` and `ModuleInspection.DefaultNamespace` → `moduleMeta.defaultNamespace`.
- [x] 3.2 Pass `ModuleInspection.PkgName` into `ReleaseOptions.PkgName` when calling `p.releaseBuilder.Build()`.
- [x] 3.3 Add fallback: if `InspectModule` returns empty `Name`, call the old `extractModuleMetadata` path (BuildInstance + LookupPath) to handle computed metadata expressions.
- [x] 3.4 Retain `extractModuleMetadata` and `moduleMetadataPreview` as fallback for computed metadata expressions (per Design Decision 3). No longer the primary path — only called when AST inspection returns empty `Name`.
- [x] 3.5 Move `CUE_REGISTRY` environment variable setup from `extractModuleMetadata` into `InspectModule` (it's already in `Build`, so `InspectModule` needs it too for modules with registry imports).

## 4. Tests

- [x] 4.1 Write `TestGenerateOverlayAST_ProducesValidCUE` — generate overlay with test inputs, `format.Node`, `parser.ParseFile`, assert no errors.
- [x] 4.2 Write `TestGenerateOverlayAST_ContainsRequiredFields` — parse the formatted overlay, walk AST to verify `#opmReleaseMeta` contains `name`, `namespace`, `fqn`, `version`, `identity`, `labels`.
- [x] 4.3 Write `TestGenerateOverlayAST_MatchesStringTemplate` — generate both AST and old `fmt.Sprintf` overlay, load both with test module, assert `#opmReleaseMeta.identity` UUIDs match. (Port from `experiments/ast-pipeline/overlay_test.go:TestOverlayAST_InterpolationExpr`.)
- [x] 4.4 Write `TestInspectModule_StaticMetadata` — create a test module with static `metadata.name` and `metadata.defaultNamespace`, call `InspectModule`, assert correct values and `PkgName` returned.
- [x] 4.5 Write `TestInspectModule_MissingMetadata` — call `InspectModule` on a module without `metadata.name` literal, assert empty `Name` returned (fallback path).
- [x] 4.6 Write `TestExtractMetadataFromAST` — unit test the AST walk helper with hand-crafted `*ast.File` inputs covering: static literals, nested `metadata` struct, missing fields, non-string expressions.
- [x] 4.7 Verify existing `TestReleaseBuilder_Build_*` tests still pass unchanged (the output should be byte-identical).
- [x] 4.8 Verify existing pipeline integration tests pass (`TestPipeline_Render_*` if they exist, or manual `opm mod build` against test fixtures).

## 5. Cleanup and Validation

- [x] 5.1 Remove unused imports from `release_builder.go` and `pipeline.go` (check: `"fmt"` may still be needed in `release_builder.go` for error wrapping).
- [x] 5.2 Update comments in `pipeline.go` to reflect the new phase flow: Phase 1 (InspectModule via AST) → Phase 2 (Build with overlay) → Phase 3 (ProviderLoader) → etc.
- [x] 5.3 Run `task fmt` — all Go files formatted.
- [x] 5.4 Run `task check` — fmt + vet + test all pass.
- [x] 5.5 Manually test `opm mod build` against the jellyfin example module — verify identical YAML output and no panics.
- [x] 5.6 Manually test `opm mod build` against a multi-component module — verify no regressions.
