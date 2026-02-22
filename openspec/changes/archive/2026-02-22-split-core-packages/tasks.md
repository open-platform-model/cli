## 1. Consolidate errors into internal/errors

- [x] 1.1 Create `internal/errors/sentinel.go` — move `ErrValidation`, `ErrConnectivity`, `ErrPermission`, `ErrNotFound` out of `errors.go`
- [x] 1.2 Create `internal/errors/domain.go` — move `TransformError` and `ValidationError` from `internal/core/errors.go`
- [x] 1.3 Remove `internal/core/errors.go`
- [x] 1.4 Update all consumers of `core.TransformError` and `core.ValidationError` to import `internal/errors`
- [x] 1.5 Run `task build` — verify no compile errors

## 2. Create internal/core/component

- [x] 2.1 Create `internal/core/component/component.go` — move `Component`, `ComponentMetadata`, `ExtractComponents` from `internal/core/component.go`
- [x] 2.2 Move `internal/core/component_test.go` to `internal/core/component/component_test.go` and update package/imports
- [x] 2.3 Remove `internal/core/component.go` and `internal/core/component_test.go`
- [x] 2.4 Update `internal/loader/module.go` — replace `core.Component`, `core.ExtractComponents` with `component` package
- [x] 2.5 Update `internal/loader/provider.go` — replace any `core.Component` references
- [x] 2.6 Update `internal/builder/builder.go` — replace `core.ExtractComponents` with `component.ExtractComponents`
- [x] 2.7 Run `task build` — verify no compile errors

## 3. Create internal/core/module

- [x] 3.1 Create `internal/core/module/module.go` — move `Module`, `ModuleMetadata` from `internal/core/module.go`
- [x] 3.2 Move `internal/core/module_test.go` to `internal/core/module/module_test.go` and update package/imports
- [x] 3.3 Remove `internal/core/module.go` and `internal/core/module_test.go`
- [x] 3.4 Update `internal/loader/module.go` — replace `core.Module`, `core.ModuleMetadata` with `module` package
- [x] 3.5 Update `internal/builder/builder.go` and `internal/builder/values.go` — replace `core.Module` with `module.Module`
- [x] 3.6 Update `internal/pipeline/pipeline.go` and `internal/pipeline/types.go` — replace `core.Module`/`core.ModuleMetadata`
- [x] 3.7 Run `task build` — verify no compile errors

## 4. Create internal/core/modulerelease

- [x] 4.1 Create `internal/core/modulerelease/release.go` — move `ModuleRelease`, `ReleaseMetadata` from `internal/core/module_release.go`
- [x] 4.2 Create `internal/core/modulerelease/validation.go` — move `validateFieldsRecursive`, `pathRewrittenError`, `rewriteErrorPath` from `internal/core/validation.go` (keep unexported)
- [x] 4.3 Move `internal/core/validation_test.go` to `internal/core/modulerelease/validation_test.go` and update package/imports
- [x] 4.4 Remove `internal/core/module_release.go`, `internal/core/validation.go`, `internal/core/validation_test.go`
- [x] 4.5 Update `internal/builder/builder.go` — replace `core.ModuleRelease`, `core.ReleaseMetadata` with `modulerelease` package
- [x] 4.6 Update `internal/pipeline/pipeline.go` and `internal/pipeline/types.go` — replace `core.ModuleRelease`/`core.ReleaseMetadata`
- [x] 4.7 Run `task build` — verify no compile errors

## 5. Create internal/core/transformer

- [x] 5.1 Create `internal/core/transformer/transformer.go` — move `Transformer`, `TransformerMetadata`, `TransformerRequirements` from `internal/core/provider.go`
- [x] 5.2 Create `internal/core/transformer/context.go` — move `TransformerContext`, `TransformerComponentMetadata`, `NewTransformerContext`, `ToMap` from `internal/core/transformer_context.go`
- [x] 5.3 Create `internal/core/transformer/match.go` — move `TransformerMatchPlan`, `TransformerMatch`, `TransformerMatchDetail`, `MatchPlan`, `TransformerMatchOld`, `ToLegacyMatchPlan` from `internal/core/provider.go`
- [x] 5.4 Create `internal/core/transformer/execute.go` — move `Execute`, `executeMatch`, decode helpers, `errMissingTransform` from `internal/core/match.go`
- [x] 5.5 Create `internal/core/transformer/warnings.go` — move `CollectWarnings` from `internal/transformer/warnings.go`
- [x] 5.6 Move `internal/core/match_test.go` and `internal/core/transformer_context_test.go` into `internal/core/transformer/` and update package/imports
- [x] 5.7 Move `internal/transformer/warnings_test.go` into `internal/core/transformer/warnings_test.go` and update package/imports
- [x] 5.8 Delete `internal/transformer/` directory entirely
- [x] 5.9 Update `internal/loader/provider.go` — replace `core.Transformer`, `core.TransformerMetadata` with `transformer` package
- [x] 5.10 Update `internal/pipeline/pipeline.go` — replace `internal/transformer` import with `internal/core/transformer`
- [x] 5.11 Update `internal/pipeline/errors.go` — replace `core.TransformerRequirements` with `transformer.TransformerRequirements`
- [x] 5.12 Update `internal/cmdutil/output.go` — replace `core.TransformError`, `core.TransformerRequirements` with new packages
- [x] 5.13 Run `task build` — verify no compile errors

## 6. Create internal/core/provider

- [x] 6.1 Create `internal/core/provider/provider.go` — move `Provider`, `ProviderMetadata`, `Match`, `evaluateMatch`, `buildMatchReason`, `Requirements` from `internal/core/provider.go`
- [x] 6.2 Remove `internal/core/provider.go` and `internal/core/match.go`
- [x] 6.3 Update `internal/loader/provider.go` — replace `core.Provider`, `core.ProviderMetadata` with `provider` package
- [x] 6.4 Update `internal/pipeline/pipeline.go` — replace `core.Provider` with `provider.Provider`
- [x] 6.5 Run `task build` — verify no compile errors

## 7. Clean up internal/core

- [x] 7.1 Verify `internal/core` contains only `resource.go`, `labels.go`, `weights.go` (and their tests)
- [x] 7.2 Update package doc comment in `internal/core/labels.go` to accurately describe the package contents
- [x] 7.3 Update `AGENTS.md` package tree to reflect the new structure

## 8. Validation

- [x] 8.1 Run `task fmt` — verify all files are formatted
- [x] 8.2 Run `task lint` — verify linter passes
- [x] 8.3 Run `task test:unit` — verify all unit tests pass
- [x] 8.4 Run `task test` — verify full test suite passes
