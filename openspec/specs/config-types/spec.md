# Capability: config-types

## Purpose

Defines the consolidated Go types used to hold CLI configuration at runtime. A single `GlobalConfig` struct in `internal/config` replaces the former three-layer nesting (`Config` → `OPMConfig` → `cmdtypes.GlobalConfig`), eliminating redundant wrappers and simplifying access paths throughout the codebase.

## Requirements

### Requirement: Single GlobalConfig type in config package

The `internal/config` package SHALL export a single `GlobalConfig` struct that contains all CLI configuration: config file values, resolved runtime state, and CLI flags. The types `Config` and `OPMConfig` SHALL NOT exist.

`GlobalConfig` SHALL contain the following fields:

- `Kubernetes` (`KubernetesConfig`): Kubernetes settings from config file
- `Log` (`LogConfig`): Logging settings from config file
- `Registry` (`string`): Resolved registry URL (after flag > env > config precedence)
- `ConfigPath` (`string`): Resolved config file path
- `Providers` (`map[string]cue.Value`): Loaded CUE provider definitions
- `CueContext` (`*cue.Context`): Shared CUE evaluation context
- `Flags` (`GlobalFlags`): Raw CLI flag values

#### Scenario: GlobalConfig contains all fields from former Config and OPMConfig

- **WHEN** `GlobalConfig` is fully populated by the loader
- **THEN** `cfg.Kubernetes.Namespace` SHALL be accessible directly (not `cfg.OPMConfig.Config.Kubernetes.Namespace`)
- **THEN** `cfg.Log.Kubernetes.APIWarnings` SHALL be accessible directly (not `cfg.OPMConfig.Config.Log.Kubernetes.APIWarnings`)
- **THEN** `cfg.Providers` SHALL be accessible directly (not `cfg.OPMConfig.Providers`)
- **THEN** `cfg.CueContext` SHALL be accessible directly (not `cfg.OPMConfig.CueContext`)

#### Scenario: Config and OPMConfig types do not exist

- **WHEN** the project is compiled
- **THEN** there SHALL be no exported type named `Config` in the `config` package
- **THEN** there SHALL be no exported type named `OPMConfig` in the `config` package

### Requirement: GlobalFlags substruct for raw CLI flag values

`GlobalConfig` SHALL contain a `Flags` field of type `GlobalFlags` that holds the raw values of all persistent CLI flags.

`GlobalFlags` SHALL contain:

- `Config` (`string`): Raw `--config` flag value
- `Registry` (`string`): Raw `--registry` flag value
- `Verbose` (`bool`): `--verbose` flag value
- `Timestamps` (`bool`): `--timestamps` flag value

#### Scenario: GlobalFlags populated from cobra flags

- **WHEN** the root command's `PersistentPreRunE` executes
- **THEN** `cfg.Flags.Config` SHALL contain the raw `--config` flag value (empty string if not set)
- **THEN** `cfg.Flags.Registry` SHALL contain the raw `--registry` flag value (empty string if not set)
- **THEN** `cfg.Flags.Verbose` SHALL contain the `--verbose` flag value
- **THEN** `cfg.Flags.Timestamps` SHALL contain the `--timestamps` flag value

#### Scenario: Flag values are distinct from resolved values

- **WHEN** `--registry` flag is empty but `OPM_REGISTRY` env var is set
- **THEN** `cfg.Flags.Registry` SHALL be empty
- **THEN** `cfg.Registry` SHALL contain the value from `OPM_REGISTRY`

### Requirement: Loader populates GlobalConfig directly

The `config.Load` function SHALL accept a `*GlobalConfig` and a `LoaderOptions` struct, and SHALL populate the config file fields, resolved runtime state, and config path directly on the provided struct. It SHALL return only an error.

The loader SHALL NOT return a separate config struct.

#### Scenario: Load populates all fields on GlobalConfig

- **WHEN** `config.Load(&cfg, opts)` succeeds
- **THEN** `cfg.Kubernetes`, `cfg.Log`, `cfg.Registry`, `cfg.ConfigPath`, `cfg.Providers`, and `cfg.CueContext` SHALL be populated
- **THEN** `cfg.Flags` SHALL NOT be modified by the loader

#### Scenario: Load with no config file uses defaults

- **WHEN** no config file exists at the resolved path
- **THEN** `cfg.Kubernetes.Kubeconfig` SHALL be `"~/.kube/config"`
- **THEN** `cfg.Kubernetes.Namespace` SHALL be `"default"`
- **THEN** `cfg.Log.Kubernetes.APIWarnings` SHALL be `"warn"`
- **THEN** `cfg.Providers` SHALL be `nil`

#### Scenario: Load resolves registry and config path

- **WHEN** `config.Load(&cfg, opts)` succeeds
- **THEN** `cfg.Registry` SHALL contain the resolved registry (flag > env > config precedence)
- **THEN** `cfg.ConfigPath` SHALL contain the resolved config file path (flag > env > default precedence)

### Requirement: No RegistrySource field

`GlobalConfig` SHALL NOT contain a `RegistrySource` field. The source of the resolved registry value SHALL NOT be tracked on the config struct.

#### Scenario: RegistrySource does not exist

- **WHEN** the project is compiled
- **THEN** `GlobalConfig` SHALL NOT have a field named `RegistrySource`

### Requirement: Resolver accepts GlobalConfig directly

`ResolveKubernetesOptions` SHALL accept a `Config *GlobalConfig` field instead of `Config *Config` and `ProviderNames []string`.

The resolver SHALL extract Kubernetes config values from `opts.Config.Kubernetes.*` and provider names from the keys of `opts.Config.Providers` internally.

#### Scenario: ResolveKubernetes reads from GlobalConfig

- **WHEN** `ResolveKubernetes` is called with `Config` set to a `*GlobalConfig` that has `Kubernetes.Namespace: "staging"` and `Providers: {"kubernetes": ...}`
- **WHEN** no flag or env override is set for namespace
- **THEN** namespace SHALL resolve to `"staging"` with source `config`
- **THEN** provider SHALL auto-resolve to `"kubernetes"` with source `config-auto`

#### Scenario: ResolveKubernetes handles nil Config

- **WHEN** `ResolveKubernetes` is called with `Config` set to `nil`
- **THEN** all fields SHALL resolve to their defaults
- **THEN** no panic SHALL occur

### Requirement: No cmdutil.ResolveKubernetes wrapper

The `cmdutil` package SHALL NOT export a `ResolveKubernetes` function. All callers SHALL use `config.ResolveKubernetes` directly.

#### Scenario: cmdutil has no ResolveKubernetes

- **WHEN** the project is compiled
- **THEN** the `cmdutil` package SHALL NOT contain an exported function named `ResolveKubernetes`

### Requirement: ResolveBase and ResolveAll are removed

The `config` package SHALL NOT export `ResolveBase`, `ResolveAll`, `ResolveBaseOptions`, `ResolveAllOptions`, `ResolvedBaseConfig`, or `ResolvedConfig`. The underlying helpers (`ResolveConfigPath`, `ResolveRegistry`, `resolveStringField`, `resolveProvider`) SHALL remain.

#### Scenario: ResolveBase does not exist

- **WHEN** the project is compiled
- **THEN** the `config` package SHALL NOT contain an exported function named `ResolveBase`
- **THEN** the `config` package SHALL NOT contain an exported type named `ResolveBaseOptions`
- **THEN** the `config` package SHALL NOT contain an exported type named `ResolvedBaseConfig`

#### Scenario: ResolveAll does not exist

- **WHEN** the project is compiled
- **THEN** the `config` package SHALL NOT contain an exported function named `ResolveAll`
- **THEN** the `config` package SHALL NOT contain an exported type named `ResolveAllOptions`
- **THEN** the `config` package SHALL NOT contain an exported type named `ResolvedConfig`

### Requirement: Dependency graph has no cycles

The dependency graph after consolidation SHALL have no import cycles. The `config` package SHALL NOT import `cmdtypes`, `cmdutil`, `build`, or any `cmd` package. The `cmdtypes` package SHALL NOT import `config`.

#### Scenario: config package imports

- **WHEN** the project is compiled
- **THEN** `internal/config` SHALL NOT have import paths containing `cmdtypes`, `cmdutil`, `build`, or `cmd`

#### Scenario: cmdtypes drops config import

- **WHEN** the project is compiled
- **THEN** `internal/cmdtypes` SHALL NOT have import paths containing `internal/config`

### Requirement: Behavioral equivalence

The consolidation SHALL NOT change any user-observable behavior. All CLI flags, output formats, exit codes, error messages, and resolution precedence SHALL remain identical.

#### Scenario: Config resolution produces identical results

- **WHEN** identical flags, environment variables, and config file are provided before and after the consolidation
- **THEN** all resolved values (registry, kubeconfig, context, namespace, provider, timestamps) SHALL be identical

#### Scenario: Build output is identical

- **WHEN** `opm mod build` is run with identical inputs before and after consolidation
- **THEN** stdout and stderr output SHALL be identical
- **THEN** exit code SHALL be identical
