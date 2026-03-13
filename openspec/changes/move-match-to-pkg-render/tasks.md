## 1. Create pkg/render package with match code

- [x] 1.1 Create `pkg/render/match.go` by copying `internal/match/match.go`, change package declaration to `render`, remove `//nolint:revive` stutter comments from `MatchResult` and `MatchPlan`
- [x] 1.2 Create `pkg/render/match_test.go` by copying `internal/match/match_test.go`, change package declaration to `render` (or `render_test`)
- [x] 1.3 Move 6 match-type tests from `internal/engine/matchplan_test.go` into `pkg/render/match_test.go`: `TestMatchedPairs_Sorted`, `TestMatchedPairs_Empty`, `TestMatchedPairs_NilMatches`, `TestWarnings_Deterministic`, `TestWarnings_Empty`, `TestWarnings_NilUnhandledTraits`. These test `MatchPlan` methods and belong with the type definition.
- [x] 1.4 Delete `TestSortMatchedPairs` from `internal/engine/matchplan_test.go` — it tests the `sortMatchedPairs` helper from `match_alias.go` which is redundant with `TestMatchedPairs_Sorted` (same sort order verified via `MatchedPairs()`)

## 2. Update internal callers (engine and releaseprocess)

- [x] 2.1 Update `internal/engine/execute.go`: change import from `internal/match` to `pkg/render`, update all references (`match.MatchPlan` → `render.MatchPlan`, `match.MatchedPair` → `render.MatchedPair`)
- [x] 2.2 Update `internal/engine/module_renderer.go`: change import from `internal/match` to `pkg/render`, update `match.MatchPlan` → `render.MatchPlan`
- [x] 2.3 Update `internal/engine/bundle_renderer.go`: change import from `internal/match` to `pkg/render`, update `match.Match` → `render.Match`
- [x] 2.4 Update `internal/releaseprocess/module.go`: change import from `internal/match` to `pkg/render`, update `match.Match` → `render.Match`

## 3. Delete match_alias.go and update its consumers

- [x] 3.1 Delete `internal/engine/match_alias.go` entirely (no aliases allowed, `sortMatchedPairs` is dead code — covered by `MatchedPairs()`)
- [x] 3.2 Update `internal/engine/matchplan_test.go`: the 2 remaining renderer tests (`TestModuleRenderer_RenderReturnsNonNilEmptySlices`, `TestBundleRenderer_RenderReturnsNonNilEmptySlices`) use alias types — add `import render "github.com/opmodel/cli/pkg/render"`, qualify `MatchPlan` → `render.MatchPlan`, `MatchResult` → `render.MatchResult`
- [x] 3.3 Update `internal/workflow/render/types.go`: replace import of `internal/engine` with `pkg/render`, change `engine.MatchPlan` → `render.MatchPlan`
- [x] 3.4 Update `internal/cmd/module/verbose_output_test.go`: replace import of `internal/engine` with `pkg/render`, change `engine.MatchPlan` → `render.MatchPlan`, `engine.MatchResult` → `render.MatchResult`
- [x] 3.5 Verify no remaining references to `engine.MatchPlan`, `engine.MatchResult`, `engine.MatchedPair`, or `engine.NonMatchedPair` in the codebase

## 4. Remove old package

- [x] 4.1 Delete `internal/match/` directory entirely

## 5. Validation

- [x] 5.1 Run `task build` — confirm compilation succeeds — confirm compilation succeeds
- [x] 5.2 Run `task test` — confirm all tests pass
- [x] 5.3 Run `task lint` — confirm linter passes

## 6. Commits

- [x] 6.1 Single commit (groups 1–4 combined): `refactor(render): move match algorithm from internal/match to pkg/render` — two-commit split not feasible because deleting `internal/match/` (4.1) breaks the alias file, forcing alias deletion into the same atomic commit
- [x] ~~6.2~~ Merged into 6.1
