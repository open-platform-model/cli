## MODIFIED Requirements

### Requirement: ModuleRenderer renders a ModuleRelease into resources
The `pkg/engine` package SHALL export a `ModuleRenderer` struct that renders a `*modulerelease.ModuleRelease` into `[]*core.Resource` with provenance metadata. The renderer SHALL execute the transform-execution phase only: it SHALL consume a precomputed Go match plan and execute matched transformer-component pairs via CUE `FillPath` and `#transform` evaluation.

#### Scenario: Successful module render
- **WHEN** `ModuleRenderer.Render(ctx, release, matchPlan)` is called with a valid release and a match plan whose components all have matching transformers
- **THEN** it returns a `*RenderResult` containing `[]*core.Resource` with each resource's `Release`, `Component`, and `Transformer` fields populated
- **AND** `Warnings` SHALL contain any unhandled trait messages derived from the supplied match plan

#### Scenario: Unmatched components produce error
- **WHEN** `ModuleRenderer.Render(ctx, release, matchPlan)` is called and the supplied match plan contains one or more unmatched components
- **THEN** it returns an `*UnmatchedComponentsError` containing the unmatched component names and per-transformer diagnostics (missing labels, missing resources, missing traits)

#### Scenario: Transform execution errors are collected
- **WHEN** individual transform executions fail during rendering
- **THEN** the renderer collects all errors and returns them joined via `errors.Join`, after attempting all matched pairs (fail-slow)

#### Scenario: Context cancellation stops execution
- **WHEN** the provided `context.Context` is cancelled during transform execution
- **THEN** the renderer stops executing remaining pairs and returns the context error alongside any resources produced so far

### Requirement: BundleRenderer renders a BundleRelease into resources
The `pkg/engine` package SHALL export a `BundleRenderer` struct that renders a `*bundlerelease.BundleRelease` by iterating its `Releases` map and calling `ModuleRenderer.Render()` for each module release using the appropriate precomputed match plan for each release.

#### Scenario: Successful bundle render
- **WHEN** `BundleRenderer.Render(ctx, bundleRelease)` is called with a valid processed bundle release
- **THEN** it returns a `*BundleRenderResult` containing all resources from all module releases, with each resource's `Release` field set to the individual module release name

#### Scenario: Bundle render collects all release errors (fail-slow)
- **WHEN** one or more module releases in the bundle fail to render
- **THEN** the BundleRenderer continues rendering remaining releases, collects all errors, and returns them alongside any successfully rendered resources

#### Scenario: Bundle releases are processed in deterministic order
- **WHEN** a BundleRelease has multiple module releases
- **THEN** the BundleRenderer processes them in sorted key order (alphabetical by instance name) to ensure deterministic output

### Requirement: Transform execution injects context and component
The engine SHALL execute each matched pair by: (1) looking up `#transform` from the provider, (2) filling `#component` with the finalized data component, (3) filling `#context` with release and component metadata, and (4) decoding the `output` field into resources.

#### Scenario: Three output forms are supported
- **WHEN** a transformer's `output` field is a CUE list
- **THEN** each list item becomes a `*core.Resource`
- **WHEN** a transformer's `output` field is a CUE struct with both `apiVersion` and `kind`
- **THEN** the entire struct becomes a single `*core.Resource`
- **WHEN** a transformer's `output` field is a CUE struct without top-level `apiVersion` and `kind`
- **THEN** each field value becomes a `*core.Resource`

#### Scenario: Metadata decode errors are propagated
- **WHEN** `metadata.labels` or `metadata.annotations` on a schema component cannot be decoded
- **THEN** the error MUST be logged at WARN level, NOT silently discarded

### Requirement: RenderResult contract
The `RenderResult` struct SHALL contain `Resources []*core.Resource`, `MatchPlan *match.MatchPlan`, `Components []ComponentSummary`, and `Warnings []string`. `Resources`, `Components`, and `Warnings` MUST be non-nil slices on successful renders, even when empty.

#### Scenario: Empty results have non-nil slices
- **WHEN** a render produces no resources, no component summaries, and no warnings
- **THEN** `RenderResult.Resources` SHALL be `[]*core.Resource{}`
- **AND** `RenderResult.Components` SHALL be `[]ComponentSummary{}`
- **AND** `RenderResult.Warnings` SHALL be `[]string{}`
