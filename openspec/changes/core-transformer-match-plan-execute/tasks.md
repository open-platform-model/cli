## 1. Move TransformerContext to internal/core

- [ ] 1.1 Create `internal/core/transformer_context.go` with `TransformerContext`, `TransformerComponentMetadata`, `NewTransformerContext(*ModuleRelease, *Component)`, and `ToMap()` — parameter types change from `*release.BuiltRelease`/`*component.Component` to `*ModuleRelease`/`*Component`; field mapping is identical
- [ ] 1.2 Verify `internal/core` has no import of any `internal/build/*` package: `go build ./internal/core/...`

## 2. Add cueCtx field and Execute() to TransformerMatchPlan

- [ ] 2.1 Add unexported `cueCtx *cue.Context` field to `core.TransformerMatchPlan` in `internal/core/provider.go`
- [ ] 2.2 Add `Execute(ctx context.Context, rel *ModuleRelease) ([]*Resource, []error)` receiver method on `*TransformerMatchPlan` — port execution logic from `Executor.ExecuteWithTransformers` + `executeJob`, iterating `m.Matches` (each match already holds `*Transformer` and `*Component`); use `m.cueCtx` for `Encode()` calls; use `core.NewTransformerContext` from step 1.1
- [ ] 2.3 Verify `Execute` returns a non-nil empty slice (not nil) when `m.Matches` is empty

## 3. Update build/pipeline.go GENERATE phase

- [ ] 3.1 In `pipeline.Render()`: after `Matcher.Match()` returns, build a `*core.TransformerMatchPlan` from the match result (populate `Matches` with matched pairs and `Unmatched` names; set `cueCtx` from the provider), and call `matchPlan.Execute(ctx, rel)` in place of `executor.ExecuteWithTransformers`
- [ ] 3.2 Remove `executor *transform.Executor` field from the `pipeline` struct and remove the `transform.NewExecutor()` call in `NewPipeline`
- [ ] 3.3 Confirm `collectWarnings` still reads from `matchResult` (no change needed — it stays in `pipeline.go` for this change)

## 4. Remove transform/executor.go and transform/context.go

- [ ] 4.1 Delete `internal/build/transform/executor.go`
- [ ] 4.2 Delete `internal/build/transform/context.go`
- [ ] 4.3 Remove `Job`, `JobResult`, `ExecuteResult` from `internal/build/transform/types.go` (these were only used by the executor); keep `MatchResult`, `MatchDetail`

## 5. Update tests

- [ ] 5.1 Delete `internal/build/transform/executor_test.go` (tests the removed `Executor`; covered by pipeline-level tests after the change)
- [ ] 5.2 Move or rewrite `internal/build/transform/context_test.go` and `context_annotations_test.go` as `internal/core/transformer_context_test.go`; update imports from `build/release.BuiltRelease`/`build/component.Component` to `core.ModuleRelease`/`core.Component`
- [ ] 5.3 Add a unit test for `TransformerMatchPlan.Execute()` in `internal/core/`: empty plan returns non-nil empty slice; context cancellation stops after current match
- [ ] 5.4 Confirm existing `matcher_test.go` still compiles and passes (no changes to `Matcher` itself)

## 6. Validation gates

- [ ] 6.1 `task fmt` passes (no formatting issues)
- [ ] 6.2 `task test` passes (all tests green, no regressions)
