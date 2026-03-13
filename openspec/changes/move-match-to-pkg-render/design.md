## Context

The `internal/match` package contains the component-to-transformer matching algorithm. It has zero CLI dependencies — it only imports `cuelang.org/go/cue` and `pkg/provider`. It is consumed by `internal/engine` and `internal/releaseprocess`, both of which will also move to `pkg/render/` in subsequent changes.

This is the first change in a series that creates a unified `pkg/render/` package. Moving `match` first is safe because it is a leaf node in the dependency graph — nothing in the group depends on it except `engine` and `releaseprocess`.

## Goals / Non-Goals

**Goals:**

- Create the `pkg/render/` package with the matching algorithm as its first content
- Move all types and functions from `internal/match/` to `pkg/render/`
- Update all internal callers to use the new import path
- Remove the `nolint:revive` stutter annotations (types no longer stutter in `render` package)

**Non-Goals:**

- Rename any types or functions (names are already correct for `pkg/render/`)
- Change any matching behavior
- Move any other packages in this change

## Decisions

### Move files, don't create wrappers

Move the actual source files to `pkg/render/` and update the package declaration. Do not create forwarding aliases or wrapper packages in `internal/match/` — clean break.

**Rationale**: All callers are internal and can be updated atomically. Wrappers add indirection for no benefit.

### Remove stutter lint suppressions

`match.MatchResult` stutters, `render.MatchResult` does not. Remove the `//nolint:revive` comments on `MatchResult` and `MatchPlan`.

**Rationale**: The stutter was only present because of the old package name. The new package name eliminates it.

### Delete `engine/match_alias.go` and update all alias consumers

`internal/engine/match_alias.go` re-exports match types as type aliases under the `engine` package. Type aliases are strictly forbidden — delete the file and update all callers that reference `engine.MatchPlan`, `engine.MatchResult`, etc. to use `render.MatchPlan`, `render.MatchResult` directly.

This expands the blast radius to include 2 additional files outside the render group:

- `internal/workflow/render/types.go` — uses `engine.MatchPlan`
- `internal/cmd/module/verbose_output_test.go` — uses `engine.MatchPlan`, `engine.MatchResult`

**Rationale**: Aliases are ugly, add indirection, and mask the true type location. All callers are internal — update them now.

## Risks / Trade-offs

- **[Low risk] Import path churn**: 7 files need import updates (5 original + 2 alias consumers), all internal. Purely mechanical.
- **[Low risk] Test fixtures**: Match tests use CUE test fixtures. Ensure they are moved alongside the test files or remain accessible from the new location.
