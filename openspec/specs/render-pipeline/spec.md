# Render Pipeline Interface

## Purpose

This spec defines the shared interface for the OPM render pipeline. The interface enables multiple CLI commands to use the same rendering logic while maintaining clear boundaries between rendering and consumption.

## Design Decisions

1. **Interface-based design**: Consumers depend on `Pipeline` interface, not implementation.
2. **RenderResult as contract**: Single struct contains all render output, enabling type-safe consumption.
3. **Fail-on-end in results**: Aggregated errors in `RenderResult.Errors` rather than failing immediately.
4. **Unstructured resources**: Platform-agnostic resource representation using `*unstructured.Unstructured`.

## Clarifications

- **Fatal vs Render errors**: Fatal errors (module not found, invalid config) return from `Render()`. Render errors (unmatched components) are in `RenderResult.Errors`.
- **Resource ordering**: Resources in `RenderResult.Resources` are ordered for sequential apply (respecting dependencies).
- **MatchPlan purpose**: Debugging and verbose output only; consumers SHOULD NOT depend on its structure for logic.

---

## User Stories

### User Story 1 - Build Command Uses Pipeline (Priority: P1)

The build command needs to render a module and output manifests.

**Independent Test**: Build command calls Pipeline.Render() and formats RenderResult.Resources as YAML.

**Acceptance Scenarios**:

1. **Given** a valid module, **When** build calls Pipeline.Render(), **Then** RenderResult contains rendered resources.
2. **Given** a module with render errors, **When** build calls Pipeline.Render(), **Then** RenderResult.Errors contains the errors.
3. **Given** RenderResult with resources, **When** build formats output, **Then** YAML contains all resources in order.

### User Story 2 - Apply Command Uses Pipeline (Priority: P1)

The apply command needs to render a module and deploy to Kubernetes.

**Independent Test**: Apply command calls Pipeline.Render() and passes RenderResult.Resources to kubernetes.Apply().

**Acceptance Scenarios**:

1. **Given** a valid module, **When** apply calls Pipeline.Render(), **Then** it receives same RenderResult as build.
2. **Given** RenderResult.Resources, **When** apply calls kubernetes.Apply(), **Then** resources are deployed in order.
3. **Given** RenderResult with Errors, **When** apply processes result, **Then** it can decide whether to proceed or abort.

---

## Functional Requirements

### Pipeline Interface

| ID | Requirement |
|----|-------------|
| FR-RP-001 | Pipeline MUST expose a `Render(ctx, opts)` method returning `(*RenderResult, error)`. |
| FR-RP-002 | Pipeline MUST return fatal errors (module not found, config invalid) as the error return value. |
| FR-RP-003 | Pipeline MUST return render errors (unmatched components, transform failures) in `RenderResult.Errors`. |
| FR-RP-004 | Pipeline MUST support context cancellation for long-running operations. |

### RenderResult

| ID | Requirement |
|----|-------------|
| FR-RP-020 | RenderResult.Resources MUST be ordered for sequential apply (dependencies first). |
| FR-RP-021 | RenderResult.Module MUST contain source module metadata. |
| FR-RP-022 | RenderResult.MatchPlan MUST describe transformer-component matches. |
| FR-RP-023 | RenderResult.Errors MUST aggregate all render errors (fail-on-end). |
| FR-RP-024 | RenderResult.Warnings MUST contain non-fatal warnings. |

### RenderResult Metadata Structure

#### Requirement: RenderResult carries both module and instance metadata

`RenderResult` SHALL carry two distinct metadata fields: `Module ModuleMetadata`
for canonical module-level identity (name, FQN, version, module UUID) and
`Instance InstanceMetadata` for instance-level identity (instance name, namespace,
instance UUID, labels). These correspond to FR-RP-021 in the base spec.

The `Module` field MUST reflect the canonical module name from
`#Module.metadata.name`, which may differ from the instance name when `--name`
overrides the default. The `Instance` field MUST reflect the deployed instance
identity (name, namespace, computed UUID).

##### Scenario: Module metadata populated on RenderResult

- **WHEN** the pipeline produces a `RenderResult`
- **THEN** `RenderResult.Module` SHALL contain the module's canonical name, FQN, version, UUID (module identity), labels, and components

##### Scenario: Instance metadata populated on RenderResult

- **WHEN** the pipeline produces a `RenderResult`
- **THEN** `RenderResult.Instance` SHALL contain the instance name, namespace, UUID (instance identity), labels, and components

##### Scenario: Module-level fields not duplicated on instance metadata

- **WHEN** a consumer accesses `RenderResult.Instance`
- **THEN** the `InstanceMetadata` type SHALL NOT contain module-level fields (Version, FQN, ModuleName, module UUID)

##### Scenario: Module and instance names differ when --name is overridden

- **WHEN** a module with `metadata.name: "my-app"` is rendered with `--name my-app-staging`
- **THEN** `RenderResult.Module.Name` SHALL equal `"my-app"` (canonical module name)
- **AND** `RenderResult.Instance.Name` SHALL equal `"my-app-staging"` (instance name)

##### Scenario: Module UUID and instance UUID are distinct

- **WHEN** any module is rendered
- **THEN** `RenderResult.Module.UUID` SHALL be the module identity UUID (from `metadata.identity`)
- **AND** `RenderResult.Instance.UUID` SHALL be the instance UUID (deterministic UUID5 from FQN+name+namespace)
- **AND** the two UUID values SHALL be different values

##### Scenario: Instance identity is preserved after refactor

- **WHEN** a module is rendered with the same `--name` and `--namespace` flags before and after this change
- **THEN** `RenderResult.Instance.UUID` SHALL be the same value as before
- **AND** all `module-instance.opmodel.dev/*` labels SHALL have the same values

#### Requirement: ModuleMetadata type definition

The `module.ModuleMetadata` struct SHALL contain the following fields, each with a corresponding `json:"..."` struct tag:
- `Name string` — canonical module name from `module.metadata.name`
- `DefaultNamespace string` — default namespace from the module
- `FQN string` — fully qualified module name
- `Version string` — module version (semver)
- `UUID string` — module identity UUID from `#Module.metadata.identity`
- `Labels map[string]string` — module labels
- `Annotations map[string]string` — module annotations (may be empty)
- `Components []string` — component names in the module

##### Scenario: ModuleMetadata populated from CUE evaluation

- **WHEN** a instance is built from a module
- **THEN** `ModuleMetadata.Name` SHALL be the canonical module name (from `module.metadata.name`)
- **AND** `ModuleMetadata.FQN` SHALL be the fully qualified module name
- **AND** `ModuleMetadata.Version` SHALL be the module semver version
- **AND** `ModuleMetadata.UUID` SHALL be the module identity UUID
- **AND** `ModuleMetadata.Labels` SHALL contain the module labels
- **AND** `ModuleMetadata.Components` SHALL list the component names

##### Scenario: ModuleMetadata JSON serialization

- **WHEN** `ModuleMetadata` is marshaled to JSON
- **THEN** all fields SHALL use their defined `json:"..."` tag names

#### Requirement: InstanceMetadata type definition

The `instance.InstanceMetadata` struct SHALL contain the following fields, each with a corresponding `json:"..."` struct tag:
- `Name string` — instance name (from `--name` flag or module name)
- `Namespace string` — target namespace
- `UUID string` — instance identity UUID (deterministic UUID5 computed from fqn+name+namespace)
- `Labels map[string]string` — instance labels
- `Annotations map[string]string` — instance annotations (may be empty)
- `Components []string` — component names rendered in this instance

##### Scenario: InstanceMetadata populated from CUE evaluation

- **WHEN** a instance is built from a module
- **THEN** `InstanceMetadata.Name` SHALL be the instance name (from `--name` or module name)
- **AND** `InstanceMetadata.Namespace` SHALL be the target namespace
- **AND** `InstanceMetadata.UUID` SHALL be the deterministic instance identity UUID
- **AND** `InstanceMetadata.Labels` SHALL contain the instance labels
- **AND** `InstanceMetadata.Components` SHALL list the component names

##### Scenario: InstanceMetadata JSON serialization

- **WHEN** `InstanceMetadata` is marshaled to JSON
- **THEN** all fields SHALL use their defined `json:"..."` tag names

#### Requirement: TransformerContext uses ModuleMetadata and InstanceMetadata

The `TransformerContext` SHALL hold references to both `ModuleMetadata` and `InstanceMetadata` instead of the former `TransformerModuleReleaseMetadata` type.

##### Scenario: TransformerContext populated from both metadata types

- **WHEN** `NewTransformerContext()` is called with a `*core.ModuleInstance` and `LoadedComponent`
- **THEN** the resulting `TransformerContext` SHALL have `ModuleMetadata` populated with module-level fields (Name, FQN, Version, UUID, Labels)
- **AND** `InstanceMetadata` populated with instance-level fields (Name, Namespace, UUID, Labels)

##### Scenario: CUE output map unchanged

- **WHEN** `TransformerContext.ToMap()` is called
- **THEN** the resulting map SHALL contain a `#moduleReleaseMetadata` key with the same field names as before: `name`, `namespace`, `fqn`, `version`, `identity`, and optionally `labels`
- **AND** the `identity` value SHALL be the instance UUID (from `InstanceMetadata.UUID`)
- **AND** the `name` value SHALL be the instance name (from `InstanceMetadata.Name`)
- **AND** the `fqn` value SHALL be the module FQN (from `ModuleMetadata.FQN`)
- **AND** the `version` value SHALL be the module version (from `ModuleMetadata.Version`)

#### Requirement: BuiltRelease carries typed metadata directly

`Builder.Build()` SHALL return `*core.ModuleInstance` instead of the build-internal
`BuiltRelease` type. The `core.ModuleInstance` type SHALL carry the same typed
metadata fields (`Module ModuleMetadata`, `Metadata *InstanceMetadata`,
`Components map[string]*Component`, `Values cue.Value`). The build-internal
`BuiltRelease` type SHALL be removed.

`TransformerContext` SHALL read module name from `core.ModuleInstance.Module.Metadata.Name`
(the canonical module name) rather than from the instance name field.

##### Scenario: Builder populates ModuleMetadata with FQN and version

- **WHEN** `instance.Build()` is called on a module that defines `metadata.fqn` and `metadata.version`
- **THEN** the returned `core.ModuleInstance.Module.FQN` SHALL equal the module's `metadata.fqn` value
- **AND** `core.ModuleInstance.Module.Version` SHALL equal the module's `metadata.version` value
- **AND** `core.ModuleInstance.Module.DefaultNamespace` SHALL equal the module's `metadata.defaultNamespace` value

##### Scenario: Builder populates InstanceMetadata with instance-level fields

- **WHEN** `instance.Build()` is called with `Name: "my-instance"` and `Namespace: "production"`
- **THEN** `core.ModuleInstance.Metadata.Name` SHALL equal `"my-instance"`
- **AND** `core.ModuleInstance.Metadata.Namespace` SHALL equal `"production"`
- **AND** `core.ModuleInstance.Metadata.UUID` SHALL be the computed instance UUID

##### Scenario: TransformerContext uses canonical module name

- **WHEN** a module with `metadata.name: "my-app"` is rendered with `--name my-app-staging`
- **THEN** `TransformerContext.ModuleMetadata.Name` SHALL equal `"my-app"` (canonical module name)
- **AND** `TransformerContext.InstanceMetadata.Name` SHALL equal `"my-app-staging"` (instance name)

#### Requirement: Pipeline BUILD phase delegates validation to ModuleInstance receiver methods

The `pipeline.Render()` BUILD phase SHALL call `rel.ValidateValues()` and then
`rel.Validate()` on the returned `*core.ModuleInstance` rather than calling
standalone validation functions directly. The pipeline SHALL NOT call validation
functions that bypass these receiver methods.

##### Scenario: BUILD phase calls ValidateValues before Validate

- **WHEN** `pipeline.Render()` executes the BUILD phase
- **THEN** `rel.ValidateValues()` SHALL be called immediately after `instance.Build()` returns
- **AND** `rel.Validate()` SHALL be called immediately after `rel.ValidateValues()` returns `nil`

##### Scenario: Validation errors surfaced identically to previous behavior

- **WHEN** user-supplied values fail schema validation after this change
- **THEN** the error returned from `pipeline.Render()` SHALL be the same type and contain the same message as before this change

##### Scenario: Instance output is identical before and after this change

- **WHEN** a module that rendered successfully before this change is rendered after
- **THEN** `RenderResult.Resources` SHALL contain the same resources with identical content
- **AND** `RenderResult.Module` and `RenderResult.Instance` SHALL contain the same metadata values
- **AND** `RenderResult.Errors` and `RenderResult.Warnings` SHALL be identical

#### Requirement: Render workflow execution

The CLI MUST share the same execution tail for instance-file rendering via `renderPreparedModuleInstance`.

##### Scenario: Shared render tail execution

- **WHEN** the `FromInstanceFile` entrypoint is executed
- **THEN** it MUST call `renderPreparedModuleInstance` with a fully resolved `*provider.Provider` and namespace override to process the instance and build the workflow `Result`

### Resource

| ID | Requirement |
|----|-------------|
| FR-RP-030 | Resource.Object MUST be `*unstructured.Unstructured`. |
| FR-RP-031 | Resource.Component MUST identify the source component. |
| FR-RP-032 | Resource.Transformer MUST identify the transformer FQN. |
| FR-RP-033 | Resources MUST include OPM tracking labels (set by transformer). |

### Error Handling

| ID | Requirement |
|----|-------------|
| FR-RP-040 | UnmatchedComponentError MUST include available transformers list. |
| FR-RP-042 | TransformError MUST include both component and transformer identification. |

### Output Consistency

#### Requirement: RenderResult.Module MUST contain source module metadata

RenderResult SHALL carry module metadata in a `Module` field of type `ModuleMetadata`. The `ModuleMetadata` type SHALL include `Name`, `DefaultNamespace`, `FQN`, `Version`, `UUID`, `Labels`, `Annotations`, and `Components` fields. This replaces the previous `ModuleMetadata` definition which had only `Name`, `Namespace`, `Version`, `Labels`, and `Components`.

##### Scenario: Module metadata available to all consumers

- **WHEN** a consumer receives a `RenderResult`
- **THEN** `result.Module.Name` SHALL be the canonical module name
- **AND** `result.Module.Version` SHALL be the module semver version
- **AND** `result.Module.FQN` SHALL be the fully qualified module name
- **AND** `result.Module.UUID` SHALL be the module identity UUID
- **AND** `result.Module.Labels` SHALL contain the module labels
- **AND** `result.Module.Components` SHALL list the component names

#### Requirement: Pipeline output is identical before and after AST refactor

The render pipeline SHALL produce byte-identical `RenderResult` output after the AST-based refactor. No user-facing behavior, resource content, metadata values, labels, or ordering SHALL change.

##### Scenario: Existing module renders identically

- **WHEN** a module that rendered successfully before the refactor is rendered after
- **THEN** the `RenderResult.Resources` SHALL contain the same resources with identical content
- **AND** `RenderResult.Module` SHALL contain the same metadata values
- **AND** `RenderResult.Errors` and `RenderResult.Warnings` SHALL be identical

##### Scenario: Instance identity is preserved

- **WHEN** a module is rendered with the same `--name` and `--namespace` flags
- **THEN** `RenderResult.Instance.UUID` SHALL be the same UUID as before the refactor
- **AND** all `module-instance.opmodel.dev/*` labels SHALL have the same values

##### Scenario: Matching phase produces identical results via Provider.Match

- **WHEN** `provider.Match(components)` is called with the same components and transformers as the previous `Matcher.Match()` call
- **THEN** the resulting `TransformerMatchPlan` SHALL contain the same matched pairs and unmatched components
- **AND** `RenderResult.MatchPlan` SHALL reflect the same transformer-component assignments

##### Scenario: Path resolution error is a fatal error

- **WHEN** `Pipeline.Render()` is called with a `ModulePath` that does not exist or is not a CUE module
- **THEN** `Render()` SHALL return a non-nil `error` (fatal error, not a render error)
- **AND** `RenderResult` SHALL be `nil`

##### Scenario: Module structural validation error is a fatal error

- **WHEN** the loaded `core.Module` fails `Validate()` (e.g., missing `Metadata.Name`)
- **THEN** `Pipeline.Render()` SHALL return a non-nil `error` (fatal error)
- **AND** `RenderResult` SHALL be `nil`

#### Requirement: Pipeline GENERATE phase delegates to TransformerMatchPlan

The `build/pipeline.go` GENERATE phase SHALL call `matchPlan.Execute(ctx, rel)` instead of constructing and invoking an `Executor` service. The `pipeline` struct SHALL NOT hold an executor field after this change.

##### Scenario: Pipeline renders without Executor field

- **WHEN** `pipeline.Render()` executes the GENERATE phase
- **THEN** it SHALL invoke `matchPlan.Execute(ctx, rel)` on the `*core.TransformerMatchPlan` returned by the MATCHING phase
- **AND** the `pipeline` struct SHALL NOT hold an `executor` field

##### Scenario: Context cancellation propagated through Execute

- **WHEN** the context passed to `pipeline.Render()` is cancelled during the GENERATE phase
- **THEN** `matchPlan.Execute()` SHALL honour the cancellation
- **AND** `pipeline.Render()` SHALL return a cancellation error (not in `RenderResult.Errors`)

### Pipeline Orchestration

#### Requirement: Pipeline delegates each phase to its named package

The `pipeline.Render()` method SHALL orchestrate the render pipeline by calling
these packages in order:

1. `loader.Load()` — PREPARATION → `*core.Module`
2. `provider.Load()` — PROVIDER LOAD → loaded provider + transformers
3. `builder.Build()` — BUILD → `*core.ModuleInstance`
4. `coreProvider.Match(rel.Components)` — MATCHING → `*core.TransformerMatchPlan`
5. `matchPlan.Execute(ctx, rel)` — GENERATE → `[]*core.Resource` + `[]error`

A fatal error from any phase SHALL abort the pipeline immediately; no subsequent
phase SHALL be called. Only `matchPlan.Execute()` errors are render errors
(collected in `RenderResult.Errors`); all prior phases produce fatal errors.

##### Scenario: Phases called in dependency order

- **WHEN** `pipeline.Render()` is called with valid options
- **THEN** `loader.Load()` is called first and its result passed to subsequent phases
- **AND** `builder.Build()` is not called until `loader.Load()` succeeds
- **AND** the MATCHING phase is not called until `builder.Build()` succeeds
- **AND** `matchPlan.Execute(ctx, rel)` is called last with the built instance

##### Scenario: Phase failure is immediately fatal

- **WHEN** `loader.Load()` returns a non-nil error
- **THEN** `pipeline.Render()` SHALL return that error as the fatal error value
- **AND** `RenderResult` SHALL be `nil`
- **AND** no subsequent phase (provider load, build, match, generate) SHALL be called

##### Scenario: Generate errors are render errors not fatal errors

- **WHEN** `matchPlan.Execute(ctx, rel)` returns one or more errors
- **THEN** those errors SHALL appear in `RenderResult.Errors`
- **AND** `pipeline.Render()` SHALL return `nil` as the error return value
- **AND** `RenderResult` SHALL be non-nil

#### Requirement: Warnings collected from TransformerMatchPlan.Matches

After the MATCHING phase, the pipeline SHALL collect unhandled-trait warnings by
inspecting `core.TransformerMatchPlan.Matches`. Each component match entry that
contains unhandled traits SHALL produce a warning string in `RenderResult.Warnings`.
The pipeline SHALL NOT read warnings from any legacy `MatchResult.Details` slice.

##### Scenario: Unhandled trait produces warning

- **WHEN** a component has a trait that no transformer handles
- **THEN** `RenderResult.Warnings` SHALL contain an entry naming the component and trait FQN
- **AND** `RenderResult.Errors` SHALL NOT contain an error for that unhandled trait

##### Scenario: No warnings when all traits are handled

- **WHEN** every trait on every component is handled by at least one matched transformer
- **THEN** `RenderResult.Warnings` SHALL be an empty slice (not nil)
- **AND** no unhandled-trait errors SHALL appear in `RenderResult.Errors`

#### Requirement: Pipeline constructor located in `internal/pipeline`

`internal/pipeline` SHALL expose a `NewPipeline()` constructor. Callers SHALL
import `internal/pipeline` to obtain a `Pipeline` implementation. No caller SHALL
import `internal/legacy` to construct a pipeline.

##### Scenario: NewPipeline returns a working Pipeline

- **WHEN** `pipeline.NewPipeline()` is called with a valid CUE context, providers, and registry
- **THEN** the returned value SHALL satisfy the `Pipeline` interface
- **AND** calling `Render()` on it SHALL execute the full phase sequence

##### Scenario: cmdutil uses internal/pipeline constructor

- **WHEN** `cmdutil.RenderRelease()` constructs a pipeline
- **THEN** it SHALL call `pipeline.NewPipeline()` from `internal/pipeline`
- **AND** it SHALL NOT import or reference `internal/legacy`

---

## Non-Functional Requirements

| ID | Requirement |
|----|-------------|
| NFR-RP-001 | Interface MUST be stable for at least one major version. |
| NFR-RP-002 | Interface MUST support future Bundle rendering without breaking changes. |

---

## Success Criteria

| ID | Criteria |
|----|----------|
| SC-RP-001 | Build command can implement using only Pipeline interface. |
| SC-RP-002 | Apply command can consume RenderResult without knowledge of rendering internals. |
| SC-RP-003 | Different implementations can satisfy Pipeline interface (for testing/future). |

---

## Edge Cases

| Case | Handling |
|------|----------|
| No resources rendered | RenderResult.Resources is empty slice, not nil. |
| All components failed | RenderResult.Resources is empty, Errors contains all failures. |
| Partial success | RenderResult contains both Resources and Errors. |
| Context cancelled | Return error from Render(), not in RenderResult. |
| Empty module (no components) | RenderResult with empty Resources, no errors. |

---

## Component Annotation Propagation

### Requirement: LoadedComponent carries annotations
The `LoadedComponent` struct SHALL include an `Annotations map[string]string` field that stores component-level annotations extracted during instance building.

#### Scenario: Annotations field initialized for all components

- **WHEN** the instance builder creates a `LoadedComponent`
- **THEN** the `Annotations` field SHALL be initialized to an empty map (not nil), regardless of whether the component has annotations

### Requirement: TransformerComponentMetadata carries annotations

The `TransformerComponentMetadata` struct SHALL include an `Annotations map[string]string` field that propagates component annotations into the transformer execution context.

#### Scenario: Annotations propagated from LoadedComponent to TransformerComponentMetadata

- **WHEN** `NewTransformerContext()` is called with a `LoadedComponent` that has annotations
- **THEN** the resulting `TransformerComponentMetadata.Annotations` SHALL contain the same key-value pairs

#### Scenario: Annotations included in CUE context map when present

- **WHEN** `ToMap()` is called on a `TransformerContext` whose component has annotations
- **THEN** the `#componentMetadata` map SHALL include an `annotations` key with the annotation map

#### Scenario: Annotations omitted from CUE context map when empty

- **WHEN** `ToMap()` is called on a `TransformerContext` whose component has no annotations
- **THEN** the `#componentMetadata` map SHALL NOT include an `annotations` key

---

## Removed Requirements

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

- **WHEN** the pipeline orchestration change is fully applied
- **THEN** no Go source file in the repository SHALL import `github.com/opmodel/cli/internal/legacy`
- **AND** the `internal/legacy/` directory SHALL NOT exist in the repository

---

## Removed Requirements (promote-factory-engine)

### Requirement: Pipeline interface
**Reason**: Replaced by concrete `ModuleRenderer` and `BundleRenderer` structs in `pkg/engine/`. The interface provided no value with only one implementation.

**Migration**: Replace `pipeline.NewPipeline().Render()` calls with `engine.NewModuleRenderer().Render()` or `engine.NewBundleRenderer().Render()`.

### Requirement: RenderOptions struct
**Reason**: Loading is no longer driven by the pipeline — the loader handles instance loading separately, and the engine receives a pre-loaded `*ModuleInstance`.

**Migration**: Callers load the instance via `pkg/loader/` and pass it to the engine. CLI flags and values resolution stay in `internal/cmdutil/`.

### Requirement: Five-phase pipeline orchestration
**Reason**: The five phases (preparation, provider load, build, matching, generate) are replaced by two distinct steps: load (via `pkg/loader/`) and render (via `pkg/engine/`). Orchestration moves to `internal/cmdutil/render.go`.

**Migration**: `cmdutil.RenderRelease()` calls `loader.LoadInstancePackage()`, `loader.LoadModuleInstanceFromValue()`, then `engine.ModuleRenderer.Render()`.

### Requirement: RenderResult with Unstructured resources
**Reason**: `RenderResult.Resources` changes from `[]*core.Resource` wrapping `*unstructured.Unstructured` to `[]*core.Resource` wrapping `cue.Value`. The new `Resource` has conversion methods.

**Migration**: Call `resource.ToUnstructured()` at the cmdutil boundary before passing to K8s or inventory packages.
