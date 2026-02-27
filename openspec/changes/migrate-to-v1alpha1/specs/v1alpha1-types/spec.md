## ADDED Requirements

### Requirement: ModuleMetadata includes ModulePath field

The `ModuleMetadata` struct SHALL include a `ModulePath string` field (replacing the former FQN-as-apiVersion pattern) that stores the value of `metadata.modulePath` from the v1alpha1 Module schema. This is a plain registry path without version (e.g., `"example.com/modules"`).

#### Scenario: ModulePath populated after load

- **WHEN** a v1alpha1 module with `metadata: modulePath: "example.com/modules"` is loaded
- **THEN** `mod.Metadata.ModulePath` SHALL equal `"example.com/modules"`

#### Scenario: ModulePath empty when absent

- **WHEN** a module without `metadata.modulePath` is loaded (e.g., lightweight test fixture)
- **THEN** `mod.Metadata.ModulePath` SHALL be an empty string

### Requirement: Workload type label constant defined

The `core` package SHALL define a constant `LabelWorkloadType` with value `"core.opmodel.dev/workload-type"` for use in component matching and label assertions.

#### Scenario: Label constant available for matching

- **WHEN** code references `core.LabelWorkloadType`
- **THEN** the value SHALL be `"core.opmodel.dev/workload-type"`

### Requirement: FQN extracted directly from CUE evaluation

The `ModuleMetadata.FQN` field SHALL be extracted directly from `metadata.fqn` in the evaluated CUE value. In v1alpha1, Module FQN is computed by CUE as `modulePath/name:version` (container-style, e.g., `"example.com/modules/my-app:0.1.0"`). The loader SHALL NOT compute FQN in Go — it is a CUE-computed field.

#### Scenario: FQN extracted from metadata.fqn

- **WHEN** a module has `metadata.modulePath: "example.com/modules"`, `metadata.name: "my-app"`, `metadata.version: "0.1.0"`
- **THEN** CUE computes `metadata.fqn: "example.com/modules/my-app:0.1.0"`
- **THEN** `mod.Metadata.FQN` SHALL equal `"example.com/modules/my-app:0.1.0"`

#### Scenario: FQN for lightweight test fixture

- **WHEN** a lightweight test fixture defines `metadata.fqn` inline (e.g., `fqn: "test.local/test-module:0.1.0"`)
- **THEN** `mod.Metadata.FQN` SHALL equal `"test.local/test-module:0.1.0"`

### Requirement: Legacy FQN extraction fallback removed

The loader SHALL NOT attempt to extract FQN from `metadata.apiVersion` as a fallback. The `metadata.apiVersion` field does not exist in v1alpha1 Module metadata. FQN is now always at `metadata.fqn`.

#### Scenario: No fallback to metadata.apiVersion

- **WHEN** a module has `metadata.fqn` present
- **THEN** the loader SHALL use `metadata.fqn` directly
- **THEN** no `metadata.apiVersion` fallback logic SHALL exist

### Requirement: Primitive FQN format uses path/name@version

Resource, Trait, Blueprint, and Transformer FQNs in v1alpha1 use the format `<modulePath>/<name>@<version>` (e.g., `"opmodel.dev/resources/workload/container@v1"`, `"opmodel.dev/traits/workload/scaling@v1"`). This format SHALL be used as keys in `#resources` and `#traits` maps.

#### Scenario: Resource FQN format

- **WHEN** a Container resource is defined with `modulePath: "opmodel.dev/resources/workload"`, `name: "container"`, `version: "v1"`
- **THEN** its FQN SHALL be `"opmodel.dev/resources/workload/container@v1"`

#### Scenario: Trait FQN format

- **WHEN** a Scaling trait is defined with `modulePath: "opmodel.dev/traits/workload"`, `name: "scaling"`, `version: "v1"`
- **THEN** its FQN SHALL be `"opmodel.dev/traits/workload/scaling@v1"`
