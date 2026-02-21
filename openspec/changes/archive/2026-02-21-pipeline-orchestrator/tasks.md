## 1. Prerequisites

- [x] 1.1 Confirm changes `pipeline-legacy-move`, `pipeline-core-raw`, `pipeline-loader`, `pipeline-provider`, `pipeline-builder`, `pipeline-transformer` are all merged
- [x] 1.2 Confirm `core-transformer-match-plan-execute` is merged
- [x] 1.3 Run `task test` to confirm baseline green before any changes

## 2. Create internal/pipeline/types.go

- [x] 2.1 Create `internal/pipeline/types.go` and copy `Pipeline` interface verbatim from `internal/legacy/types.go`
- [x] 2.2 Copy `RenderOptions` struct (all fields: ModulePath, Values, Name, Namespace, Provider, Strict, Registry) to `types.go`
- [x] 2.3 Copy `RenderResult` struct and its helper methods (`HasErrors`, `HasWarnings`, `ResourceCount`) to `types.go`
- [x] 2.4 Run `go build ./internal/pipeline/...` — types file must compile cleanly

## 3. Create internal/pipeline/pipeline.go

- [x] 3.1 Create `internal/pipeline/pipeline.go` with `pipeline` struct and `NewPipeline(config) Pipeline` constructor
- [x] 3.2 Implement PREPARATION: call `loader.Load(ctx, opts)` → `*core.Module`; return fatal error on failure
- [x] 3.3 Implement PROVIDER LOAD: call `provider.Load(ctx, module, opts)` → loaded provider + transformers; return fatal error on failure
- [x] 3.4 Implement BUILD: call `builder.Build(ctx, module, opts)` → `*core.ModuleRelease`; call `rel.ValidateValues()` then `rel.Validate()`; return fatal error on either failure
- [x] 3.5 Implement MATCHING: call `transformer.Match(rel.Components, loadedProvider.Transformers)` → `*core.TransformerMatchPlan`; return fatal error on failure
- [x] 3.6 Implement warning/error collection from `core.TransformerMatchPlan.Matches`: unhandled traits → `RenderResult.Warnings` (non-strict) or `RenderResult.Errors` (strict)
- [x] 3.7 Implement GENERATE: call `matchPlan.Execute(ctx, rel)` → resources + errors; append errors to `RenderResult.Errors` (not fatal)
- [x] 3.8 Sort resources by weight → group → kind → namespace → name before populating `RenderResult.Resources`
- [x] 3.9 Assemble and return `*RenderResult` with Resources, Module metadata, Release metadata, MatchPlan, Errors, Warnings
- [x] 3.10 Run `go build ./internal/pipeline/...` — full file must compile cleanly

## 4. Test internal/pipeline

- [x] 4.1 Write test: successful end-to-end render with a valid module produces non-nil `RenderResult` with resources
- [x] 4.2 Write test: `loader.Load()` failure returns fatal error and nil `RenderResult`; no downstream phase called
- [x] 4.3 Write test: generate errors land in `RenderResult.Errors`; `pipeline.Render()` returns nil error
- [x] 4.4 Write test: unhandled trait with `Strict: false` → warning in `RenderResult.Warnings`, no error
- [x] 4.5 Write test: unhandled trait with `Strict: true` → error in `RenderResult.Errors`, no warning
- [x] 4.6 Write test: context cancellation during GENERATE returns cancellation error (not in `RenderResult.Errors`)
- [x] 4.7 Run `go test ./internal/pipeline/...` — all tests pass

## 5. Update cmdutil imports

- [x] 5.1 In `internal/cmdutil/render.go`: replace `internal/legacy` import with `internal/pipeline`; replace `legacy.NewPipeline(...)` with `pipeline.NewPipeline(...)`
- [x] 5.2 In `internal/cmdutil/output.go`: replace `internal/legacy` import with `internal/pipeline`
- [x] 5.3 In `internal/cmd/mod/verbose_output_test.go`: replace `internal/legacy` import with `internal/pipeline`
- [x] 5.4 Run `go build ./internal/cmdutil/... ./internal/cmd/...` — all packages compile cleanly
- [x] 5.5 Run `task test` — all tests pass with the new import paths

## 6. Delete internal/legacy/ and final validation

- [x] 6.1 Delete the `internal/legacy/` directory
- [x] 6.2 Run `grep -r "internal/legacy" .` — confirm zero results
- [x] 6.3 Run `task test` — all tests pass after legacy deletion
- [x] 6.4 Run `task check` — fmt, vet, and test all pass
