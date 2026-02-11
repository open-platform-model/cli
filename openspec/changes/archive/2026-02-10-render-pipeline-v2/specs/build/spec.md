## MODIFIED Requirements

### Requirement: ReleaseBuilder uses CUE #ModuleRelease for release building

ReleaseBuilder SHALL construct a `core.#ModuleRelease` instance via `load.Config.Overlay` instead of manually calling `FillPath(#config, values)` and `Validate(cue.Concrete(true))` in Go. CUE SHALL handle value validation, config injection, component concreteness, and metadata computation.

#### Scenario: ReleaseBuilder constructs release via #ModuleRelease

- **WHEN** the ReleaseBuilder receives a loaded module and release options (name, namespace)
- **THEN** it SHALL create an overlay file that constructs `core.#ModuleRelease` with the module's fields
- **AND** it SHALL call `load.Instances` + `BuildInstance` with the overlay
- **AND** it SHALL extract `BuiltRelease` (components, metadata) from `_opmRelease`

#### Scenario: Invalid values produce CUE validation error

- **WHEN** values do not satisfy `#module.#config` (e.g., wrong type, missing required field)
- **THEN** the ReleaseBuilder SHALL return a fatal error containing the CUE evaluator's error message
- **AND** the error SHALL include file path, line number, and column from CUE

#### Scenario: ReleaseBuilder no longer calls FillPath or Validate directly

- **WHEN** the ReleaseBuilder builds a release
- **THEN** it SHALL NOT call `moduleValue.FillPath(cue.ParsePath("#config"), values)`
- **AND** it SHALL NOT call `component.Value.Validate(cue.Concrete(true))`
- **AND** concreteness SHALL be guaranteed by the CUE `#ModuleRelease` evaluation

### Requirement: Executor uses isolated CUE contexts for parallel execution

The Executor SHALL create a fresh `*cue.Context` per job and re-materialize transformer and component values from CUE source text. No CUE state SHALL be shared across goroutines.

#### Scenario: Parallel execution of multi-component module

- **WHEN** a module has multiple components that match transformers
- **THEN** the executor SHALL process all jobs in parallel using worker goroutines
- **AND** each job SHALL use its own `*cue.Context` created via `cuecontext.New()`
- **AND** no runtime panic SHALL occur regardless of worker count or job count

#### Scenario: Transformer and component values serialized as CUE text

- **WHEN** the executor prepares for parallel execution
- **THEN** transformer values SHALL be serialized to CUE source text via `format.Node(value.Syntax())`
- **AND** component values SHALL be serialized to CUE source text via `format.Node(value.Syntax())`
- **AND** serialized text SHALL be cached per unique transformer FQN and component name

#### Scenario: Re-materialized values produce identical output

- **WHEN** a transformer processes a component in an isolated context
- **THEN** the output resources SHALL be identical to what the same transformer would produce in a single-threaded execution
- **AND** deterministic output (FR-B-053) SHALL be preserved

### Requirement: Pipeline executes transformers in parallel

The pipeline SHALL execute transformers in parallel using worker goroutines. This preserves the existing FR-B-021 requirement while fixing the concurrency mechanism.

#### Scenario: Worker count matches available CPUs

- **WHEN** the pipeline creates an Executor
- **THEN** the worker count SHALL be set to `runtime.NumCPU()`
- **AND** the worker count SHALL be capped at the number of jobs (no idle workers)

#### Scenario: Fail-on-end aggregation preserved

- **WHEN** some jobs succeed and some jobs fail
- **THEN** all jobs SHALL execute (no early termination on first failure)
- **AND** all errors SHALL be collected in `ExecuteResult.Errors`
- **AND** all successful resources SHALL be collected in `ExecuteResult.Resources`
