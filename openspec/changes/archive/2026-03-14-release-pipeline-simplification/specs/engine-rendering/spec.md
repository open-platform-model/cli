## MODIFIED Requirements

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
