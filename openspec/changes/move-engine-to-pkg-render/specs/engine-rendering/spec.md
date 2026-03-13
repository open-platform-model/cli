## MODIFIED Requirements

### Requirement: Module renderer renders a ModuleRelease into resources
The `pkg/render` package SHALL export a `Module` struct (previously `ModuleRenderer`) that renders a `*render.ModuleRelease` into `[]*core.Resource` with provenance metadata. The renderer SHALL execute the transform-execution phase only: it SHALL consume a precomputed `*render.MatchPlan` and execute matched transformer-component pairs via CUE `FillPath` and `#transform` evaluation.

#### Scenario: Successful module render
- **WHEN** `render.Module.Execute(ctx, release, matchPlan)` is called with a valid release and match plan
- **THEN** it returns a `*render.ModuleResult` containing `[]*core.Resource`
- **AND** `Warnings` SHALL contain any unhandled trait messages and any metadata decode warnings

#### Scenario: Unmatched components produce error
- **WHEN** `render.Module.Execute(ctx, release, matchPlan)` is called and the match plan contains unmatched components
- **THEN** it returns an `*render.UnmatchedComponentsError`

#### Scenario: Metadata decode failure surfaces as warning
- **WHEN** a rendered resource's metadata cannot be decoded during transform execution
- **THEN** the failure SHALL be appended to `ModuleResult.Warnings` as a descriptive string
- **AND** the resource SHALL still be included in the result

### Requirement: Bundle renderer renders a BundleRelease into resources
The `pkg/render` package SHALL export a `Bundle` struct (previously `BundleRenderer`) that renders a `*render.BundleRelease` into `[]*core.Resource` by iterating child module releases and delegating to `Module.Execute`.

#### Scenario: Successful bundle render
- **WHEN** `render.Bundle.Execute(ctx, bundleRelease)` is called with a valid bundle
- **THEN** it returns a `*render.BundleResult` containing aggregated `[]*core.Resource` from all child module renders

### Requirement: No CLI logging framework dependency
The `pkg/render` package SHALL NOT import `charmbracelet/log` or any other CLI-specific logging framework. All diagnostic information SHALL be surfaced through return values (Warnings slices, error types).

#### Scenario: Render package compiled without charmbracelet/log
- **WHEN** `pkg/render/` is compiled
- **THEN** its transitive dependency tree SHALL NOT contain `github.com/charmbracelet/log`
