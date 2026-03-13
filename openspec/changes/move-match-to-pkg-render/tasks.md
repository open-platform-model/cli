## 1. Create pkg/render package with match code

- [ ] 1.1 Create `pkg/render/match.go` by copying `internal/match/match.go`, change package declaration to `render`, remove `//nolint:revive` stutter comments from `MatchResult` and `MatchPlan`
- [ ] 1.2 Create `pkg/render/match_test.go` by copying `internal/match/match_test.go`, change package declaration to `render` (or `render_test`)

## 2. Update internal callers (engine and releaseprocess)

- [ ] 2.1 Update `internal/engine/execute.go`: change import from `internal/match` to `pkg/render`, update all references (`match.MatchPlan` → `render.MatchPlan`, `match.MatchedPair` → `render.MatchedPair`)
- [ ] 2.2 Update `internal/engine/module_renderer.go`: change import from `internal/match` to `pkg/render`, update `match.MatchPlan` → `render.MatchPlan`
- [ ] 2.3 Update `internal/engine/bundle_renderer.go`: change import from `internal/match` to `pkg/render`, update `match.Match` → `render.Match`
- [ ] 2.4 Update `internal/releaseprocess/module.go`: change import from `internal/match` to `pkg/render`, update `match.Match` → `render.Match`

## 3. Delete match_alias.go and update its consumers

- [ ] 3.1 Delete `internal/engine/match_alias.go` entirely (no aliases allowed)
- [ ] 3.2 Update `internal/workflow/render/types.go`: replace import of `internal/engine` with `pkg/render`, change `engine.MatchPlan` → `render.MatchPlan`
- [ ] 3.3 Update `internal/cmd/module/verbose_output_test.go`: replace import of `internal/engine` with `pkg/render`, change `engine.MatchPlan` → `render.MatchPlan`, `engine.MatchResult` → `render.MatchResult`
- [ ] 3.4 Verify no remaining references to `engine.MatchPlan`, `engine.MatchResult`, `engine.MatchedPair`, or `engine.NonMatchedPair` in the codebase

## 4. Remove old package

- [ ] 4.1 Delete `internal/match/` directory entirely

## 5. Validation

- [ ] 5.1 Run `task build` — confirm compilation succeeds
- [ ] 5.2 Run `task test` — confirm all tests pass
- [ ] 5.3 Run `task lint` — confirm linter passes

## 6. Commits

- [ ] 6.1 Commit tasks 1.1, 1.2, 2.1–2.4, 4.1: `refactor(render): move match algorithm from internal/match to pkg/render`
- [ ] 6.2 Commit tasks 3.1–3.4: `refactor(engine): delete match type aliases and update consumers to pkg/render`
