## ADDED Requirements

### Requirement: Module path is resolved before loading
The loader SHALL resolve the provided module path to an absolute, validated path before any CUE loading occurs. Relative paths SHALL be resolved against the working directory. Non-existent paths SHALL return an error immediately.

#### Scenario: Relative path is resolved
- **WHEN** a relative module path is provided
- **THEN** the loader resolves it to an absolute path and proceeds with loading

#### Scenario: Non-existent path is rejected
- **WHEN** the module path does not exist on disk
- **THEN** the loader returns an error before attempting CUE evaluation

---

### Requirement: CUE instance is loaded with registry configuration
The loader SHALL load the CUE instance from the resolved module path using `load.Instances`. When a registry string is provided, the loader SHALL set `CUE_REGISTRY` for the duration of the load.

#### Scenario: Registry is set during load
- **WHEN** a non-empty registry string is provided
- **THEN** `CUE_REGISTRY` is set before `load.Instances` is called and unset after

#### Scenario: No instances found returns error
- **WHEN** the module path contains no CUE instances
- **THEN** the loader returns a descriptive error

#### Scenario: Instance load error is surfaced
- **WHEN** `load.Instances` returns an instance with a non-nil error
- **THEN** the loader returns that error wrapped with context

---

### Requirement: CUE instance is fully evaluated
The loader SHALL call `cueCtx.BuildInstance` to fully evaluate the loaded CUE instance. Any evaluation errors SHALL be returned immediately.

#### Scenario: Evaluation error is returned
- **WHEN** `BuildInstance` produces a value with a non-nil error
- **THEN** the loader returns that error wrapped with context

---

### Requirement: Module metadata is extracted into Module.Metadata
The loader SHALL extract all scalar metadata fields from the evaluated CUE value and populate `core.ModuleMetadata`: `name`, `fqn`, `version`, `uuid`, `defaultNamespace`, and `labels`. Fields absent from the CUE value SHALL be left as zero values without error.

#### Scenario: All metadata fields are present
- **WHEN** the module defines `metadata.name`, `metadata.fqn`, `metadata.version`, `metadata.uuid`, `metadata.defaultNamespace`, and `metadata.labels`
- **THEN** all fields are populated on `core.ModuleMetadata`

#### Scenario: Partial metadata is tolerated
- **WHEN** only some metadata fields are present in the module
- **THEN** present fields are populated and absent fields remain zero values without error

---

### Requirement: Config schema is extracted into Module.Config
The loader SHALL extract the `#config` definition from the evaluated value and set it on `core.Module.Config`. If `#config` is absent, `Module.Config` SHALL remain a zero `cue.Value` without error.

#### Scenario: Config schema is present
- **WHEN** the module defines `#config`
- **THEN** `Module.Config` is set to the extracted `#config` value

#### Scenario: Absent config is not an error
- **WHEN** the module does not define `#config`
- **THEN** `Module.Config` is a zero value and no error is returned

---

### Requirement: Default values are extracted into Module.Values
The loader SHALL extract the `values` field from the evaluated value and set it on `core.Module.Values`. If `values` is absent, `Module.Values` SHALL remain a zero `cue.Value` without error.

#### Scenario: Values field is present
- **WHEN** the module defines a top-level `values` field
- **THEN** `Module.Values` is set to the extracted value

#### Scenario: Absent values is not an error
- **WHEN** the module does not define a `values` field
- **THEN** `Module.Values` is a zero value and no error is returned

---

### Requirement: Components are extracted into Module.Components
The loader SHALL extract `#components` from the evaluated value and populate `core.Module.Components`. If `#components` is absent, `Module.Components` SHALL remain nil without error.

#### Scenario: Components are present
- **WHEN** the module defines `#components`
- **THEN** `Module.Components` is populated with all extracted components

#### Scenario: Absent components is not an error
- **WHEN** the module does not define `#components`
- **THEN** `Module.Components` is nil and no error is returned

---

### Requirement: Raw CUE value is stored on Module.Raw
The loader SHALL store the fully evaluated `cue.Value` on `core.Module.Raw`. This value MUST be the complete evaluated module value, usable for injection into `#ModuleRelease` via `FillPath` in the BUILD phase.

#### Scenario: Raw is set after successful load
- **WHEN** the module is successfully loaded and evaluated
- **THEN** `Module.Raw` holds the fully evaluated `cue.Value` and is non-zero

---

### Requirement: Returned Module passes validation
The loader SHALL return a `*core.Module` that passes `mod.Validate()` — meaning all required fields (path, metadata, Raw) are populated.

#### Scenario: Loaded module is valid
- **WHEN** loading succeeds
- **THEN** the returned `*core.Module` passes `mod.Validate()` without error
