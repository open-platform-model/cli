# Engine Rendering

## Purpose

Defines the contract for `pkg/render` — the rendering engine that consumes processed releases and precomputed match plans to produce `[]*core.Resource`.

## Requirements

### Requirement: Module renderer renders a ModuleRelease into resources
The `pkg/render` package SHALL export a `Module` struct that renders a `*render.ModuleRelease` into `[]*core.Resource` with provenance metadata. The renderer SHALL execute the transform-execution phase only: it SHALL consume a precomputed `*render.MatchPlan` and execute matched transformer-component pairs via CUE `FillPath` and `#transform` evaluation.

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

#### Scenario: Transform execution errors are collected
- **WHEN** individual transform executions fail during rendering
- **THEN** the renderer collects all errors and returns them joined via `errors.Join`, after attempting all matched pairs (fail-slow)

#### Scenario: Context cancellation stops execution
- **WHEN** the provided `context.Context` is cancelled during transform execution
- **THEN** the renderer stops executing remaining pairs and returns the context error alongside any resources produced so far

### Requirement: Bundle renderer renders a BundleRelease into resources
The `pkg/render` package SHALL export a `Bundle` struct that renders a `*render.BundleRelease` into `[]*core.Resource` by iterating child module releases and delegating to `Module.Execute`.

#### Scenario: Successful bundle render
- **WHEN** `render.Bundle.Execute(ctx, bundleRelease)` is called with a valid bundle
- **THEN** it returns a `*render.BundleResult` containing aggregated `[]*core.Resource` from all child module renders

#### Scenario: Bundle render collects all release errors (fail-slow)
- **WHEN** one or more module releases in the bundle fail to render
- **THEN** the Bundle continues rendering remaining releases, collects all errors, and returns them alongside any successfully rendered resources

#### Scenario: Bundle releases are processed in deterministic order
- **WHEN** a BundleRelease has multiple module releases
- **THEN** the Bundle processes them in sorted key order (alphabetical by instance name) to ensure deterministic output

### Requirement: Transform execution injects context and component
The engine SHALL execute each matched pair by: (1) looking up `#transform` from the provider, (2) filling `#component` with the finalized data component, (3) filling `#context` with release and component metadata, and (4) decoding the `output` field into resources.

#### Scenario: Three output forms are supported
- **WHEN** a transformer's `output` field is a CUE list
- **THEN** each list item becomes a `*core.Resource`
- **WHEN** a transformer's `output` field is a CUE struct with both `apiVersion` and `kind`
- **THEN** the entire struct becomes a single `*core.Resource`
- **WHEN** a transformer's `output` field is a CUE struct without top-level `apiVersion` and `kind`
- **THEN** each field value becomes a `*core.Resource`

### Requirement: No CLI logging framework dependency
The `pkg/render` package SHALL NOT import `charmbracelet/log` or any other CLI-specific logging framework. All diagnostic information SHALL be surfaced through return values (Warnings slices, error types).

#### Scenario: Render package compiled without charmbracelet/log
- **WHEN** `pkg/render/` is compiled
- **THEN** its transitive dependency tree SHALL NOT contain `github.com/charmbracelet/log`

### Requirement: ModuleResult contract
The `ModuleResult` struct SHALL contain `Resources []*core.Resource`, `MatchPlan *MatchPlan`, `Components []ComponentSummary`, and `Warnings []string`. `Resources`, `Components`, and `Warnings` MUST be non-nil slices on successful renders, even when empty.

#### Scenario: Empty results have non-nil slices
- **WHEN** a render produces no resources, no component summaries, and no warnings
- **THEN** `ModuleResult.Resources` SHALL be `[]*core.Resource{}`
- **AND** `ModuleResult.Components` SHALL be `[]ComponentSummary{}`
- **AND** `ModuleResult.Warnings` SHALL be `[]string{}`
