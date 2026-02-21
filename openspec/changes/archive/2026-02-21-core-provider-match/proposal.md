## Why

Transformer-component matching is currently owned by `internal/build/transform/matcher.go`, a stateless service struct with no natural home in the domain model. Since matching is inherently a question the `Provider` answers — "which of my transformers handle this set of components?" — it belongs as a receiver method on `core.Provider`. This change also completes the full retirement of the legacy `MatchPlan`/`TransformerMatchOld`/`TransformerRequirements` types, replacing them with the new `TransformerMatchPlan`/`TransformerMatch`/`core.Transformer` types throughout the pipeline and output layer.

## What Changes

- Add `Match(components map[string]*Component) *TransformerMatchPlan` receiver method to `core.Provider`
- The matching algorithm (label check, required resource/trait check, O(components × transformers)) moves from `transform/matcher.go` into this method
- `core.Provider` stores the `*cue.Context` set during construction by `transform.LoadProvider()` and passes it into the returned `TransformerMatchPlan`
- `internal/build/transform/` loader updated: `LoadProvider()` sets `provider.cueCtx` and returns `*core.Provider`
- `internal/build/transform/matcher.go` and `Matcher` struct removed (logic moved to receiver method)
- `build/pipeline.go` MATCHING phase updated to call `provider.Match(rel.Components)`
- `RenderResult.MatchPlan` field type changed from `core.MatchPlan` to `*core.TransformerMatchPlan`
- `cmdutil/output.go` updated to iterate `TransformerMatchPlan.Matches []TransformerMatch` instead of `MatchPlan.Matches map[string][]TransformerMatchOld`
- `build.UnmatchedComponentError.Available` changed from `[]core.TransformerRequirements` to `[]*core.Transformer`
- Legacy types removed from `internal/core/provider.go`: `MatchPlan`, `TransformerMatchOld`, `TransformerRequirements`
- `transform/matcher.go:ToMatchPlan()` removed (the last producer of `MatchPlan`)

## Capabilities

### New Capabilities

- `provider-match`: `Match()` receiver method on `core.Provider` encapsulating transformer-component matching logic and producing `*TransformerMatchPlan` as the canonical match output

### Modified Capabilities

- `render-pipeline`: The MATCHING phase of `pipeline.Render()` now invokes `provider.Match()` instead of constructing and calling a separate `Matcher` service; `RenderResult.MatchPlan` field type changes from `core.MatchPlan` to `*core.TransformerMatchPlan`

## Impact

- `internal/core/provider.go` — new `Match()` method on `*Provider`; add unexported `cueCtx *cue.Context` field; remove `MatchPlan`, `TransformerMatchOld`, `TransformerRequirements`
- `internal/build/transform/provider.go` — `LoadProvider()` sets `provider.cueCtx` before returning `*core.Provider`
- `internal/build/transform/matcher.go` — removed entirely; all logic migrated to `core.Provider.Match()`
- `internal/build/types.go` — `RenderResult.MatchPlan` type changes to `*core.TransformerMatchPlan`
- `internal/build/errors.go` — `UnmatchedComponentError.Available` type changes to `[]*core.Transformer`
- `internal/cmdutil/output.go` — `WriteTransformerMatches()` and `WriteVerboseMatchLog()` updated to use `TransformerMatchPlan`; `PrintRenderErrors()` updated to access `core.Transformer` fields directly
- `internal/build/pipeline.go` — MATCHING phase simplified; `collectWarnings()` updated to work from `*TransformerMatchPlan` instead of `*MatchResult`
- Tests updated across `cmdutil/`, `build/`, `build/transform/`
- SemVer: **PATCH** — internal refactor; no change to CLI behavior or user-facing output format
