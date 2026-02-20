## ADDED Requirements

### Requirement: Load() performs full CUE evaluation after AST inspection

`module.Load()` SHALL call `cueCtx.BuildInstance(inst)` on the same `*build.Instance` already used for AST inspection, producing a base `cue.Value` for the fully-evaluated module. This SHALL happen in a single load — `load.Instances()` is called exactly once and its result is reused for both phases.

#### Scenario: Load returns Module with non-zero CUE value
- **WHEN** `module.Load()` is called with a valid module directory
- **THEN** `mod.CUEValue().Exists()` SHALL return `true`
- **AND** `mod.CUEValue().Err()` SHALL return `nil`

#### Scenario: AST inspection and BuildInstance share the same instance
- **WHEN** `module.Load()` runs
- **THEN** `load.Instances()` SHALL be called exactly once
- **AND** `cueCtx.BuildInstance(inst)` SHALL be called on the same `*build.Instance` used for AST inspection

#### Scenario: CUE evaluation error surfaces as Load error
- **WHEN** `module.Load()` is called on a module with a CUE evaluation error (e.g., unresolvable import or invalid expression in metadata)
- **THEN** `Load()` SHALL return a non-nil error wrapping the CUE evaluation failure
- **AND** the returned `*core.Module` SHALL be `nil`

---

### Requirement: Load() populates Module.Config from #config

`module.Load()` SHALL extract the `#config` CUE definition from the evaluated base value and store it in `Module.Config` as a `cue.Value`. `Module.Config` represents the module's user-facing configuration schema and is used by `Build()` to validate user-supplied values before `FillPath`.

#### Scenario: Module with #config definition
- **WHEN** `module.Load()` is called on a module that defines `#config`
- **THEN** `Module.Config.Exists()` SHALL return `true`

#### Scenario: User values are validated against Module.Config during Build()
- **WHEN** user-supplied values are provided to `Build()`
- **THEN** they SHALL be unified with `Module.Config` to detect type mismatches before `FillPath` is called
- **AND** a unification error SHALL cause `Build()` to return a non-nil error before any components are extracted

---

### Requirement: Load() populates Module.Values from the module values field

`module.Load()` SHALL extract the module's `values` field (if present) from the evaluated base value and store it in `Module.Values` as a `cue.Value`. This field holds the module author's suggested config inputs — a plain struct such as `values: { image: "nginx:1.28.2", replicas: 1 }`.

#### Scenario: Module with values field
- **WHEN** `module.Load()` is called on a module that defines a top-level `values` field
- **THEN** `Module.Values.Exists()` SHALL return `true`

#### Scenario: Module without values field
- **WHEN** `module.Load()` is called on a module that does not define a top-level `values` field
- **THEN** `Module.Values.Exists()` SHALL return `false`
- **AND** no error SHALL be raised

#### Scenario: Module.Values used as fallback when no --values is provided
- **WHEN** `Build()` is called without any `--values` files
- **THEN** `Module.Values` SHALL be used as the config input passed to `FillPath("#config", ...)`

#### Scenario: Module.Values ignored when --values is provided
- **WHEN** `Build()` is called with one or more `--values` files
- **THEN** `Module.Values` SHALL be ignored entirely
- **AND** the user-supplied values SHALL be used as the sole config input

---

### Requirement: Load() populates Module.Components at schema level

`module.Load()` SHALL call `core.ExtractComponents()` on the `#components` value from the evaluated base value, storing the result in `Module.Components`. At this stage components are schema-level: their spec fields reference `#config` type constraints and are not required to be concrete.

#### Scenario: Components extracted and structurally validated during Load()
- **WHEN** `module.Load()` is called on a module that defines `#components`
- **THEN** `Module.Components` SHALL be a non-empty map of component name to `*core.Component`
- **AND** `comp.Validate()` SHALL return `nil` for every extracted component

#### Scenario: Load() fails if a component is structurally invalid
- **WHEN** `module.Load()` extracts a component that fails `Validate()` (e.g., missing `#resources`)
- **THEN** `Load()` SHALL return a non-nil error identifying the invalid component
- **AND** the returned `*core.Module` SHALL be `nil`

---

### Requirement: Module.CUEValue() exposes the base cue.Value

`core.Module` SHALL expose a `CUEValue() cue.Value` accessor that returns the unexported base `cue.Value` set by `module.Load()`. The base value is the fully-evaluated module used by `Build()` as the starting point for `FillPath`.

#### Scenario: CUEValue() returns zero value before Load()
- **WHEN** `CUEValue()` is called on a `*core.Module` not returned by `module.Load()`
- **THEN** `CUEValue().Exists()` SHALL return `false`

#### Scenario: Build() obtains base value from CUEValue() — no second load
- **WHEN** `Build()` is called with a `*core.Module` returned by `module.Load()`
- **THEN** `Build()` SHALL call `mod.CUEValue()` to obtain the base value
- **AND** `Build()` SHALL NOT call `load.Instances()` or `cueCtx.BuildInstance()`

#### Scenario: Base value is immutable across multiple Build() calls
- **WHEN** `Build()` is called multiple times with the same `*core.Module`
- **THEN** each call SHALL produce an independent filled value via `FillPath`
- **AND** `mod.CUEValue()` SHALL return the same unmodified base value on every call

---

### Requirement: Load() populates full Module.Metadata from CUE evaluation

`module.Load()` SHALL read `metadata.fqn`, `metadata.uuid`, `metadata.version`, and `metadata.labels` from the evaluated `cue.Value` and store them in `Module.Metadata`. This extends the existing AST-based population of `Metadata.Name` and `Metadata.DefaultNamespace`.

#### Scenario: Metadata fields populated from CUE evaluation
- **WHEN** `module.Load()` is called on a module with fully concrete metadata fields
- **THEN** `Module.Metadata.FQN` SHALL be populated from `metadata.fqn`
- **AND** `Module.Metadata.UUID` SHALL be populated from `metadata.uuid`
- **AND** `Module.Metadata.Version` SHALL be populated from `metadata.version`
- **AND** `Module.Metadata.Labels` SHALL be populated from `metadata.labels`

#### Scenario: Non-concrete metadata.uuid causes Load error
- **WHEN** `module.Load()` is called on a module where `metadata.uuid` is not a concrete string (e.g., depends on an unresolvable import)
- **THEN** `Load()` SHALL return a non-nil error
