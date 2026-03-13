## Purpose

Defines the public matching API in `pkg/render` that enables external consumers to invoke the component-to-transformer matching algorithm without depending on CLI internals.

## Requirements

### Requirement: Public matching API in pkg/render
The `pkg/render` package SHALL export the component-to-transformer matching algorithm so that external consumers (CLI, Kubernetes controller, other tools) can invoke matching without depending on CLI internals.

#### Scenario: External consumer calls render.Match
- **WHEN** an external Go module imports `pkg/render` and calls `render.Match(components, provider)`
- **THEN** it receives a `*render.MatchPlan` with the same matching results as the previous `internal/match.Match` function

#### Scenario: All match types are publicly accessible
- **WHEN** an external Go module imports `pkg/render`
- **THEN** it can reference `render.MatchPlan`, `render.MatchResult`, `render.MatchedPair`, and `render.NonMatchedPair` as exported types

### Requirement: No CLI dependencies in matching code
The matching implementation in `pkg/render` SHALL NOT import any CLI-specific packages (cobra, charmbracelet/log, lipgloss, internal/output, internal/cmd, internal/cmdutil).

#### Scenario: Clean dependency tree
- **WHEN** `pkg/render/match.go` is compiled
- **THEN** its transitive dependency tree contains only: standard library, `cuelang.org/go/cue`, and `pkg/provider`
