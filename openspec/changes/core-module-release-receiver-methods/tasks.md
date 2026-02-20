## 1. Add receiver methods to core.ModuleRelease

- [ ] 1.1 Add `ValidateValues() error` to `internal/core/module_release.go` — guard on `!rel.Module.Config.Exists() || !rel.Values.Exists()`, then move `validateValuesAgainstConfig` logic from `release/validation.go` into the method body
- [ ] 1.2 Add `Validate() error` to `internal/core/module_release.go` — iterate `rel.Components`, call `comp.Value.Validate(cue.Concrete(true))` on each, aggregate errors into `*core.ValidationError`
- [ ] 1.3 Verify both methods have no side effects: confirm neither mutates any field on `ModuleRelease`

## 2. Update release.Builder to return *core.ModuleRelease

- [ ] 2.1 Change `Builder.Build()` return type from `*BuiltRelease` to `*core.ModuleRelease` in `internal/build/release/builder.go`
- [ ] 2.2 Remove step 4b (`validateValuesAgainstConfig` call) from `Build()` — validation moves to the receiver method
- [ ] 2.3 Remove step 7 (concrete component check loop) from `Build()` — validation moves to the receiver method
- [ ] 2.4 Replace `return &BuiltRelease{...}` with `return &core.ModuleRelease{...}` — populate `Module` (from `modMeta`), `Metadata` (from `relMeta`), `Components` (converting from `build/component.Component` to `core.Component`), `Values`
- [ ] 2.5 Remove `BuiltRelease` type from `internal/build/release/types.go`

## 3. Update build/pipeline.go BUILD phase

- [ ] 3.1 Update the BUILD phase in `internal/build/pipeline.go` to accept `*core.ModuleRelease` from `release.Build()`
- [ ] 3.2 Add `rel.ValidateValues()` call immediately after `Build()` returns; return error if non-nil
- [ ] 3.3 Add `rel.Validate()` call immediately after `ValidateValues()` succeeds; return error if non-nil
- [ ] 3.4 Update downstream pipeline code (phases 3–6) that references `rel.ReleaseMetadata`, `rel.ModuleMetadata`, `rel.Components` to use the new `*core.ModuleRelease` field names (`rel.Metadata`, `rel.Module`, `rel.Components`)

## 4. Update affected tests

- [ ] 4.1 Update `internal/build/transform/executor_test.go` — replace `release.ReleaseMetadata` references with `core.ReleaseMetadata` (or construct `*core.ModuleRelease` directly)
- [ ] 4.2 Update `internal/build/transform/context_annotations_test.go` — same as 4.1; fix the missing `build/module` import error
- [ ] 4.3 Update `internal/build/pipeline_test.go` — fix `core.ModuleMetadata.Components` references once the field is confirmed present, or adjust assertions to new field layout

## 5. Cleanup

- [ ] 5.1 Remove or reduce `internal/build/release/validation.go` — `validateValuesAgainstConfig` and `validateFieldsRecursive` logic now lives in `core`; keep `pathRewrittenError`, `findSourcePosition`, and `rewriteErrorPath` helpers if still needed by `collectAllCUEErrors`
- [ ] 5.2 Confirm `release/types.go` only contains `Options`; delete file if empty after `BuiltRelease` removal

## 6. Validation gates

- [ ] 6.1 Run `task fmt` — all Go files formatted
- [ ] 6.2 Run `task test` — all tests pass with no regressions
