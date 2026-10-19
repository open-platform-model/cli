## REMOVED Requirements

### Requirement: Single GlobalConfig type in config package
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The `internal/config` package exports a single `GlobalConfig` struct and no `Config` or `OPMConfig` type", "`GlobalConfig` contains the fields `Kubernetes`, `Log`, `Registry`, `ConfigPath` and `Flags`", "`GlobalConfig` contains no `Providers` field and no `CueContext` field".

### Requirement: Skew policy configuration key
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "`config.cue` accepts an optional `skewPolicy` key with the values `warn` and `refuse`, defaulting to `warn`", "The `skewPolicy` value governs how a render responds to a module that requires a newer build than the platform pins", "The `skewPolicy` key applies to the local default or a `--platform` directory and the cluster Platform CR takes precedence".

## ADDED Requirements

### Requirement: The `internal/config` package exports a single `GlobalConfig` struct and no `Config` or `OPMConfig` type

The `internal/config` package SHALL export a single `GlobalConfig` struct that contains all CLI configuration: config file values, resolved runtime state, and CLI flags. The types `Config` and `OPMConfig` SHALL NOT exist.

#### Scenario: Config and OPMConfig types do not exist

- **WHEN** the project is compiled
- **THEN** there SHALL be no exported type named `Config` in the `config` package
- **THEN** there SHALL be no exported type named `OPMConfig` in the `config` package

### Requirement: `GlobalConfig` contains the fields `Kubernetes`, `Log`, `Registry`, `ConfigPath` and `Flags`

`GlobalConfig` SHALL contain the following fields:

- `Kubernetes` (`KubernetesConfig`): Kubernetes settings from config file
- `Log` (`LogConfig`): Logging settings from config file
- `Registry` (`string`): Resolved registry URL (after flag > env > config precedence)
- `ConfigPath` (`string`): Resolved config file path
- `Flags` (`GlobalFlags`): Raw CLI flag values

#### Scenario: GlobalConfig contains only surviving fields

- **WHEN** `GlobalConfig` is fully populated by the loader
- **THEN** `cfg.Kubernetes.Namespace` SHALL be accessible directly
- **THEN** `cfg.Log.Kubernetes.APIWarnings` SHALL be accessible directly
- **THEN** `GlobalConfig` SHALL NOT have a field named `Providers`
- **THEN** `GlobalConfig` SHALL NOT have a field named `CueContext`

### Requirement: `GlobalConfig` contains no `Providers` field and no `CueContext` field

`GlobalConfig` SHALL NOT contain a `Providers` field or a `CueContext` field. CUE evaluation contexts are owned by the per-invocation kernel, not by configuration.

#### Scenario: GlobalConfig has no Providers field and no CueContext field

- **WHEN** `GlobalConfig` is fully populated by the loader
- **THEN** `GlobalConfig` SHALL NOT have a field named `Providers`
- **THEN** `GlobalConfig` SHALL NOT have a field named `CueContext`

### Requirement: `config.cue` accepts an optional `skewPolicy` key with the values `warn` and `refuse`, defaulting to `warn`

`config.cue` SHALL accept an optional `skewPolicy` key with the values `"warn"` and `"refuse"`, defaulting to `"warn"` when absent (enhancement 0019, "Skew policy defaults to warn-and-render; the comparison reads the two committed resolutions; older-than-platform is data").
Any other value SHALL fail config validation naming the allowed values. No flag SHALL exist for it.

#### Scenario: Default is warn

- **WHEN** `config.cue` carries no `skewPolicy`
- **THEN** renders against a local or flag platform use the warn policy

#### Scenario: Invalid value rejected

- **WHEN** `config.cue` sets `skewPolicy: "strict"`
- **THEN** config validation fails naming the allowed values `warn` and `refuse`

### Requirement: The `skewPolicy` value governs how a render responds to a module that requires a newer build than the platform pins

The `skewPolicy` value SHALL govern how a render responds when a module's `cue.mod` requires a newer build of an OPM-namespace path (core or a catalog) than the platform pins: `warn` renders against the platform's build and reports the skew; `refuse` fails the render before evaluation.

#### Scenario: Refuse configured

- **WHEN** `config.cue` sets `skewPolicy: "refuse"` and a render against the local platform hits a newer module requirement
- **THEN** the render is refused before evaluation naming the path and both versions

### Requirement: The `skewPolicy` key applies to the local default or a `--platform` directory and the cluster Platform CR takes precedence

The `skewPolicy` key applies when the platform is the local default or a `--platform` directory; when the cluster Platform CR is the source, the CR's `spec.skewPolicy` SHALL take precedence so CLI and operator judge the same platform the same way.

#### Scenario: Cluster CR overrides the key

- **WHEN** `config.cue` sets `skewPolicy: "refuse"` and the render resolves the cluster Platform CR whose `spec.skewPolicy` is `Warn`
- **THEN** the render uses warn and reports the CR as the policy's source
