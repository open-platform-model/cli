## ADDED Requirements

### Requirement: Build-pipeline commands resolve namespace with 4-step precedence

Commands that use the build pipeline (`mod build`, `mod apply`, `mod diff`, `mod vet`) SHALL resolve the target namespace using the following precedence, from highest to lowest:

1. `--namespace` / `-n` flag
2. `OPM_NAMESPACE` environment variable
3. `module.metadata.defaultNamespace` from the loaded module (if set)
4. `config.kubernetes.namespace` from `.opm/config.cue`

Step 3 is only applicable when a module has been loaded. If `module.metadata.defaultNamespace` is not defined in the module (the field is optional in the CUE `#Module` schema), step 3 is skipped and resolution falls through to step 4.

#### Scenario: Flag takes highest precedence

- **WHEN** `opm mod build --namespace prod` is run on a module with `metadata.defaultNamespace: "staging"` and `OPM_NAMESPACE=dev` is set and `config.kubernetes.namespace: "default"` is configured
- **THEN** the pipeline SHALL use `"prod"` as the target namespace

#### Scenario: Env var overrides module and config defaults

- **WHEN** `opm mod build` is run without `--namespace` and `OPM_NAMESPACE=dev` is set and the module defines `metadata.defaultNamespace: "staging"`
- **THEN** the pipeline SHALL use `"dev"` as the target namespace

#### Scenario: Module defaultNamespace overrides config default

- **WHEN** `opm mod build` is run without `--namespace` and `OPM_NAMESPACE` is not set and the module defines `metadata.defaultNamespace: "staging"` and `config.kubernetes.namespace: "default"` is configured
- **THEN** the pipeline SHALL use `"staging"` as the target namespace

#### Scenario: Config default used when no higher-precedence source is set

- **WHEN** `opm mod build` is run without `--namespace` and `OPM_NAMESPACE` is not set and the module does not define `metadata.defaultNamespace` and `config.kubernetes.namespace: "production"` is configured
- **THEN** the pipeline SHALL use `"production"` as the target namespace

#### Scenario: Module without defaultNamespace falls through to config

- **WHEN** `opm mod build` is run without `--namespace` and `OPM_NAMESPACE` is not set and the module's `metadata` does not include `defaultNamespace` (optional field omitted)
- **THEN** the pipeline SHALL use the value from `config.kubernetes.namespace`

### Requirement: Non-pipeline commands are unaffected

Commands that do not use the build pipeline (`mod delete`, `mod status`) SHALL continue using the existing 3-step resolution:

1. `--namespace` / `-n` flag
2. `OPM_NAMESPACE` environment variable
3. `config.kubernetes.namespace` from `.opm/config.cue`

These commands do not load a module and therefore have no access to `module.metadata.defaultNamespace`.

#### Scenario: mod delete uses 3-step resolution

- **WHEN** `opm mod delete` is run without `--namespace` and `OPM_NAMESPACE` is not set and `config.kubernetes.namespace: "default"` is configured
- **THEN** the command SHALL use `"default"` as the target namespace
- **AND** `module.metadata.defaultNamespace` SHALL NOT be consulted

### Requirement: Pipeline receives namespace source information

The pipeline SHALL receive enough information from the command layer to distinguish whether namespace was explicitly set by the user (flag or env var) or fell through to the config default. This allows the pipeline to insert `module.metadata.defaultNamespace` at the correct position in the precedence chain after the module is loaded.

`GlobalConfig` SHALL NOT be mutated. Namespace resolution for the pipeline is transient — the effective namespace is determined per-invocation and does not persist.

#### Scenario: Pipeline overrides config-sourced namespace with module default

- **WHEN** the pipeline receives a namespace value that was resolved from `config.kubernetes.namespace` (source: config or default) and the loaded module defines `metadata.defaultNamespace: "whoopie"`
- **THEN** the pipeline SHALL use `"whoopie"` as the target namespace

#### Scenario: Pipeline does not override flag-sourced namespace

- **WHEN** the pipeline receives a namespace value that was resolved from `--namespace` flag and the loaded module defines `metadata.defaultNamespace: "whoopie"`
- **THEN** the pipeline SHALL use the flag value as the target namespace
- **AND** `module.metadata.defaultNamespace` SHALL be ignored

#### Scenario: Pipeline does not override env-sourced namespace

- **WHEN** the pipeline receives a namespace value that was resolved from `OPM_NAMESPACE` and the loaded module defines `metadata.defaultNamespace: "whoopie"`
- **THEN** the pipeline SHALL use the env var value as the target namespace
- **AND** `module.metadata.defaultNamespace` SHALL be ignored

### Requirement: ExtractMetadata CUE fallback is removed

The `module.ExtractMetadata()` function and the `module.MetadataPreview` type SHALL be removed from `internal/build/module/`. The CUE evaluation fallback for `metadata.name` is unnecessary because `metadata.name!` is a mandatory required field in the CUE `#Module` schema.

#### Scenario: Module with string-literal name loads without CUE fallback

- **WHEN** `module.Load()` is called on a module where `metadata.name` is a string literal
- **THEN** `Metadata.Name` SHALL be populated from AST inspection
- **AND** no CUE evaluation SHALL be performed during the PREPARATION phase
