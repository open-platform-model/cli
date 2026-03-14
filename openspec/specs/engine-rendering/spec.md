# Engine Rendering

## Purpose

Defines the contract for `pkg/render` — the rendering engine that consumes prepared releases and precomputed match plans to produce `[]*core.Resource`.

## Requirements

### Requirement: Module renderer renders a Release into resources
The `pkg/render` package SHALL export a `Module` struct that renders a `*module.Release` into `[]*core.Resource` with provenance metadata. The renderer SHALL execute the transform-execution phase only: it SHALL consume a precomputed `*render.MatchPlan` and execute matched transformer-component pairs via CUE `FillPath` and `#transform` evaluation.

#### Scenario: Successful module render
- **WHEN** `render.Module.Execute(ctx, rel, schemaComponents, dataComponents, plan)` is called with a valid release, component views, and match plan
- **THEN** it returns a `*render.ModuleResult` containing `[]*core.Resource`
- **AND** `Warnings` SHALL contain any unhandled trait messages and any metadata decode warnings

#### Scenario: Execute reads metadata from Release
- **WHEN** `render.Module.Execute` accesses release metadata during transform execution
- **THEN** it SHALL read `rel.Metadata` for release-level identity
- **AND** it SHALL read `rel.Module.Metadata` for module-level identity

#### Scenario: Execute receives components as arguments
- **WHEN** `render.Module.Execute` is called
- **THEN** it SHALL receive schema-preserving components and finalized data components as function arguments
- **AND** it SHALL NOT read `rel.ExecuteComponents()` or `rel.DataComponents`

#### Scenario: Unmatched components produce error
- **WHEN** `render.Module.Execute` is called and the match plan contains unmatched components
- **THEN** it returns an `*render.UnmatchedComponentsError`

#### Scenario: Transform execution errors are collected
- **WHEN** individual transform executions fail during rendering
- **THEN** the renderer collects all errors and returns them joined via `errors.Join`, after attempting all matched pairs (fail-slow)

#### Scenario: Context cancellation stops execution
- **WHEN** the provided `context.Context` is cancelled during transform execution
- **THEN** the renderer stops executing remaining pairs and returns the context error alongside any resources produced so far

### Requirement: Bundle renderer renders a BundleRelease into resources
The `pkg/render` package SHALL export a `Bundle` struct that renders a `*bundle.Release` into `[]*core.Resource` by iterating child module releases and delegating to `Module.Execute`.

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

### Requirement: Transform execution functions accept Release
The internal `executeTransforms`, `executePair`, and `injectContext` functions SHALL accept `*module.Release`. Metadata access SHALL use `rel.Metadata` and `rel.Module.Metadata` directly.

#### Scenario: executeTransforms receives Release
- **WHEN** `executeTransforms` is called during rendering
- **THEN** it SHALL accept `rel *module.Release` as its release parameter
- **AND** it SHALL pass the same `*module.Release` to `executePair`

#### Scenario: injectContext reads metadata from Release
- **WHEN** `injectContext` builds `#moduleReleaseMetadata` for the transformer context
- **THEN** it SHALL read `rel.Metadata.Name`, `rel.Metadata.Namespace`, `rel.Metadata.UUID`, `rel.Metadata.Labels`, `rel.Metadata.Annotations`
- **AND** it SHALL read `rel.Module.Metadata.FQN`, `rel.Module.Metadata.Version`

### Requirement: Bundle renderer uses Release map
The `Bundle.Execute` method SHALL iterate `bundle.Release.Releases` which contains `*module.Release` entries. It SHALL call `MatchComponents()` on each `Release` for matching.

#### Scenario: Bundle renderer iterates releases
- **WHEN** `render.Bundle.Execute(ctx, bundleRelease)` is called
- **THEN** it SHALL iterate `bundleRelease.Releases` which are `*module.Release`
- **AND** it SHALL call `modRel.MatchComponents()` on each entry

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
