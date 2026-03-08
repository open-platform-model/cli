## ADDED Requirements

### Requirement: ModuleRenderer renders a ModuleRelease into resources
The `pkg/engine` package SHALL export a `ModuleRenderer` struct that renders a `*modulerelease.ModuleRelease` into `[]*core.Resource` with provenance metadata. The renderer SHALL execute a two-phase pipeline: CUE-native matching via `#MatchPlan`, then per-pair transform execution via CUE `FillPath` and `#transform` evaluation.

#### Scenario: Successful module render
- **WHEN** `ModuleRenderer.Render(ctx, release)` is called with a valid release that has matching transformers for all components
- **THEN** it returns a `*RenderResult` containing `[]*core.Resource` with each resource's `Release`, `Component`, and `Transformer` fields populated, and `Warnings` containing any unhandled trait messages

#### Scenario: Unmatched components produce error
- **WHEN** `ModuleRenderer.Render(ctx, release)` is called and one or more components have no matching transformer
- **THEN** it returns an `*UnmatchedComponentsError` containing the unmatched component names and per-transformer diagnostics (missing labels, missing resources, missing traits)

#### Scenario: Transform execution errors are collected
- **WHEN** individual transform executions fail during Phase 2
- **THEN** the renderer collects all errors and returns them joined via `errors.Join`, after attempting all matched pairs (fail-slow)

#### Scenario: Context cancellation stops execution
- **WHEN** the provided `context.Context` is cancelled during Phase 2 execution
- **THEN** the renderer stops executing remaining pairs and returns the context error alongside any resources produced so far

### Requirement: BundleRenderer renders a BundleRelease into resources
The `pkg/engine` package SHALL export a `BundleRenderer` struct that renders a `*bundlerelease.BundleRelease` by iterating its `Releases` map and calling `ModuleRenderer.Render()` for each module release.

#### Scenario: Successful bundle render
- **WHEN** `BundleRenderer.Render(ctx, bundleRelease)` is called with a valid bundle release
- **THEN** it returns a `*BundleRenderResult` containing all resources from all module releases, with each resource's `Release` field set to the individual module release name

#### Scenario: Bundle render collects all release errors (fail-slow)
- **WHEN** one or more module releases in the bundle fail to render
- **THEN** the BundleRenderer continues rendering remaining releases, collects all errors, and returns them alongside any successfully rendered resources

#### Scenario: Bundle releases are processed in deterministic order
- **WHEN** a BundleRelease has multiple module releases
- **THEN** the BundleRenderer processes them in sorted key order (alphabetical by instance name) to ensure deterministic output

### Requirement: CUE-native matching via #MatchPlan
The engine SHALL perform component-to-transformer matching by loading the CUE `./core/matcher` package, filling `#provider` and `#components` into the `#MatchPlan` definition, and decoding the result. Go code SHALL NOT implement matching logic.

#### Scenario: Match plan evaluation
- **WHEN** `buildMatchPlan()` is called with a provider CUE value and schema components CUE value
- **THEN** it loads the matcher CUE package, fills `#MatchPlan.#provider` and `#MatchPlan.#components`, evaluates the CUE expression, and decodes the result into a `MatchPlan` struct with `Matches`, `Unmatched`, and `UnhandledTraits` fields

#### Scenario: MatchPlan provides structured diagnostics
- **WHEN** a transformer does not match a component
- **THEN** the `MatchResult` for that pair contains `Matched: false` and non-empty `MissingLabels`, `MissingResources`, or `MissingTraits` slices identifying exactly what was missing

#### Scenario: MatchedPairs are deterministically sorted
- **WHEN** `MatchPlan.MatchedPairs()` is called
- **THEN** the returned pairs are sorted by component name ascending, then transformer FQN ascending

#### Scenario: Warnings are deterministically sorted
- **WHEN** `MatchPlan.Warnings()` is called
- **THEN** the returned warning strings are sorted by component name then trait FQN

### Requirement: Transform execution injects context and component
The engine SHALL execute each matched pair by: (1) looking up `#transform` from the provider, (2) filling `#component` with the finalized data component, (3) filling `#context` with release and component metadata, (4) decoding the `output` field into resources.

#### Scenario: Three output forms are supported
- **WHEN** a transformer's `output` field is a CUE list
- **THEN** each list item becomes a `*core.Resource`
- **WHEN** a transformer's `output` field is a CUE struct with an `apiVersion` field
- **THEN** the entire struct becomes a single `*core.Resource`
- **WHEN** a transformer's `output` field is a CUE struct without an `apiVersion` field
- **THEN** each field value becomes a `*core.Resource`

#### Scenario: Metadata decode errors are propagated
- **WHEN** `metadata.labels` or `metadata.annotations` on a schema component cannot be decoded
- **THEN** the error MUST be logged at WARN level, NOT silently discarded

### Requirement: RenderResult contract
The `RenderResult` struct SHALL contain `Resources []*core.Resource` and `Warnings []string`. Both fields MUST be non-nil (empty slice, not nil). There is no `Pipeline` interface — consumers call `ModuleRenderer.Render()` or `BundleRenderer.Render()` directly.

#### Scenario: Empty results have non-nil slices
- **WHEN** a render produces no resources and no warnings
- **THEN** `RenderResult.Resources` is `[]*core.Resource{}` (not nil) and `RenderResult.Warnings` is `[]string{}` (not nil)
