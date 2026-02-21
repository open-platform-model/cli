## 1. Core type changes (done)

- [x] 1.1 ~~Add `CueCtx` to `core.Provider`~~ — removed; neither `Provider` nor `TransformerMatchPlan` carry the CUE context (see design Decision 2)
- [x] 1.2 `core.TransformerMatch` and `core.TransformerMatchDetail` types defined in `internal/core/provider.go`
- [x] 1.3 `core.TransformerMatchPlan` defined as a pure data type in `internal/core/provider.go`

## 2. Implement core.Provider.Match()

- [x] 2.1 Add `Match(components map[string]*Component) *TransformerMatchPlan` method to `core.Provider`
- [x] 2.2 Port label-checking logic from `transform/matcher.go:evaluateMatch()` into the new method
- [x] 2.3 Port resource-checking logic from `transform/matcher.go:evaluateMatch()` into the new method
- [x] 2.4 Port trait-checking logic (required + unhandled) from `transform/matcher.go:evaluateMatch()` into the new method
- [x] 2.5 Port reason-string building from `transform/matcher.go:buildReason()` into the new method
- [x] 2.6 Sort component names before iterating to produce a deterministic match plan (see design Open Questions)
- [x] 2.7 ~~Set cueCtx on TransformerMatchPlan~~ — not needed; pipeline passes `p.cueCtx` directly to `Execute()` in the next change

## 3. Update transform.LoadProvider()

- [x] 3.1 Rename / replace `ProviderLoader.Load()` with a package-level `LoadProvider(providers map[string]cue.Value, name string) (*core.Provider, []*LoadedTransformer, error)` function in `internal/build/transform/provider.go` — no `cueCtx` parameter needed; returns `[]*LoadedTransformer` as secondary return for the executor until next migration step
- [x] 3.2 Populate `core.Provider.Transformers` (type `map[string]*core.Transformer`) from the extracted transformer data in `LoadProvider()`
- [x] 3.3 Ensure `core.Transformer` fields (`RequiredLabels`, `RequiredResources`, `RequiredTraits`, `OptionalLabels`, `OptionalResources`, `OptionalTraits`, `Transform`) are populated from the CUE value extraction

## 4. Update pipeline.Render() MATCHING phase

- [x] 4.1 Replace `p.provider.Load(ctx, providerName)` with `transform.LoadProvider(p.providers, providerName)` in `internal/build/pipeline.go`
- [x] 4.2 Replace `p.matcher.Match(components, provider.Transformers)` with `provider.Match(rel.Components)` — removed `componentsToSlice()` conversion helper
- [x] 4.3 Update `collectWarnings()` in `pipeline.go` to read from `*core.TransformerMatchPlan` fields instead of `*transform.MatchResult.Details`
- [x] 4.4 Update unmatched component error collection to read `matchPlan.Unmatched` from `core.TransformerMatchPlan`
- [x] 4.5 Remove `p.matcher *transform.Matcher` field from the `pipeline` struct and its initialization in `NewPipeline()`
- [x] 4.6 Remove `p.provider *transform.ProviderLoader` field from the `pipeline` struct and its initialization in `NewPipeline()`
- [x] 4.7 ~~Add `cueCtx *cue.Context` field to `pipeline` struct and set it from the argument in `NewPipeline()`~~ — already present in `pipeline.go:26`; no change needed

## 5. Delete dead code

- [x] 5.1 Delete `internal/build/transform/matcher.go` (Matcher struct, Match, evaluateMatch, buildReason, ToMatchPlan)
- [x] 5.2 Remove `transform.MatchResult` and `transform.MatchDetail` types from `internal/build/transform/types.go`
- [x] 5.3 Remove `NewProviderLoader` and `ProviderLoader` struct from `internal/build/transform/provider.go` (replaced by `LoadProvider`)
- [x] 5.4 Remove `LoadedProvider.Requirements()` — `LoadedProvider` struct itself removed; `LoadedTransformer` retained (still used by executor until next change); `core.Transformer` now implements `TransformerRequirements` via `GetFQN()`, `GetRequiredLabels()`, `GetRequiredResources()`, `GetRequiredTraits()` methods

## 6. Note for core-transformer-match-plan-execute

- [x] 6.0 `TransformerMatchPlan.Execute()` will receive `cueCtx` as a parameter (`Execute(ctx, p.cueCtx, rel)`) — the pipeline passes its own `p.cueCtx` directly; update `core-transformer-match-plan-execute` design accordingly

## 7. Validation

- [x] 7.1 Run `task fmt` — all Go files formatted
- [x] 7.2 Run `task test` — all tests pass with identical RenderResult output
- [x] 7.3 Verify `task build` produces a working binary and `opm mod build` on an existing fixture produces byte-identical YAML output
