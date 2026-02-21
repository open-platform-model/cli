## ADDED Requirements

### Requirement: Pipeline delegates each phase to its named package

The `pipeline.Render()` method SHALL orchestrate the render pipeline by calling
these packages in order:

1. `loader.Load(ctx, opts)` — PREPARATION → `*core.Module`
2. `provider.Load(ctx, module)` — PROVIDER LOAD → loaded provider + transformers
3. `builder.Build(ctx, module, opts)` — BUILD → `*core.ModuleRelease`
4. `transformer.Match(rel.Components, provider.Transformers)` — MATCHING → `*core.TransformerMatchPlan`
5. `matchPlan.Execute(ctx, rel)` — GENERATE → `[]*core.Resource` + `[]error`

A fatal error from any phase SHALL abort the pipeline immediately; no subsequent
phase SHALL be called. Only `matchPlan.Execute()` errors are render errors
(collected in `RenderResult.Errors`); all prior phases produce fatal errors.

#### Scenario: Phases called in dependency order

- **WHEN** `pipeline.Render()` is called with valid options
- **THEN** `loader.Load()` is called first and its result passed to subsequent phases
- **AND** `builder.Build()` is not called until `loader.Load()` succeeds
- **AND** `transformer.Match()` is not called until `builder.Build()` succeeds
- **AND** `matchPlan.Execute(ctx, rel)` is called last with the built release

#### Scenario: Phase failure is immediately fatal

- **WHEN** `loader.Load()` returns a non-nil error
- **THEN** `pipeline.Render()` SHALL return that error as the fatal error value
- **AND** `RenderResult` SHALL be `nil`
- **AND** no subsequent phase (provider load, build, match, generate) SHALL be called

#### Scenario: Generate errors are render errors not fatal errors

- **WHEN** `matchPlan.Execute(ctx, rel)` returns one or more errors
- **THEN** those errors SHALL appear in `RenderResult.Errors`
- **AND** `pipeline.Render()` SHALL return `nil` as the error return value
- **AND** `RenderResult` SHALL be non-nil

### Requirement: Warnings collected from TransformerMatchPlan.Matches

After the MATCHING phase, the pipeline SHALL collect unhandled-trait warnings by
inspecting `core.TransformerMatchPlan.Matches`. Each component match entry that
contains unhandled traits SHALL produce a warning string in `RenderResult.Warnings`
(non-strict mode) or an error in `RenderResult.Errors` (strict mode). The pipeline
SHALL NOT read warnings from any legacy `MatchResult.Details` slice.

#### Scenario: Unhandled trait produces warning in non-strict mode

- **WHEN** a component has a trait that no transformer handles
- **AND** `RenderOptions.Strict` is `false`
- **THEN** `RenderResult.Warnings` SHALL contain an entry naming the component and trait FQN
- **AND** `RenderResult.Errors` SHALL NOT contain an error for that unhandled trait

#### Scenario: Unhandled trait produces error in strict mode

- **WHEN** a component has a trait that no transformer handles
- **AND** `RenderOptions.Strict` is `true`
- **THEN** `RenderResult.Errors` SHALL contain an error naming the component and trait FQN
- **AND** `RenderResult.Warnings` SHALL NOT contain a warning for that unhandled trait

#### Scenario: No warnings when all traits are handled

- **WHEN** every trait on every component is handled by at least one matched transformer
- **THEN** `RenderResult.Warnings` SHALL be an empty slice (not nil)
- **AND** no unhandled-trait errors SHALL appear in `RenderResult.Errors`

### Requirement: Pipeline constructor located in `internal/pipeline`

`internal/pipeline` SHALL expose a `NewPipeline(config)` constructor. Callers
SHALL import `internal/pipeline` to obtain a `Pipeline` implementation. No caller
SHALL import `internal/legacy` to construct a pipeline after this change is applied.

#### Scenario: NewPipeline returns a working Pipeline

- **WHEN** `pipeline.NewPipeline(config)` is called with a valid OPM config
- **THEN** the returned value SHALL satisfy the `Pipeline` interface
- **AND** calling `Render()` on it SHALL execute the full phase sequence

#### Scenario: cmdutil uses internal/pipeline constructor

- **WHEN** `cmdutil.RenderRelease()` constructs a pipeline
- **THEN** it SHALL call `pipeline.NewPipeline(config)` from `internal/pipeline`
- **AND** it SHALL NOT import or reference `internal/legacy`

## REMOVED Requirements

### Requirement: Legacy pipeline package

**Reason**: Replaced by `internal/pipeline/`. All phase logic now lives in
`internal/loader/`, `internal/builder/`, `internal/provider/`, and
`internal/transformer/`. The monolithic `internal/legacy/` package served as a
transitional holder while those packages were built out; it is no longer needed.

**Migration**: Replace any import of `github.com/opmodel/cli/internal/legacy` with
`github.com/opmodel/cli/internal/pipeline`. Replace `legacy.NewPipeline(config)`
with `pipeline.NewPipeline(config)`. The `Pipeline` interface, `RenderOptions`,
`RenderResult`, and helper methods (`HasErrors`, `HasWarnings`, `ResourceCount`)
are available at the new import path with identical signatures.

#### Scenario: No files import internal/legacy after this change

- **WHEN** the `pipeline-orchestrator` change is fully applied
- **THEN** no Go source file in the repository SHALL import `github.com/opmodel/cli/internal/legacy`
- **AND** the `internal/legacy/` directory SHALL NOT exist in the repository
