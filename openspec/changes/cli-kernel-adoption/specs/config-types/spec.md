# Delta: config-types (cli-kernel-adoption)

`GlobalConfig` drops `Providers` and `CueContext`; the resolver drops provider auto-resolution (enhancement 0006 D39). The kernel owns the CUE context (see `kernel-render`).

## MODIFIED Requirements

### Requirement: Single GlobalConfig type in config package

The `internal/config` package SHALL export a single `GlobalConfig` struct that contains all CLI configuration: config file values, resolved runtime state, and CLI flags. The types `Config` and `OPMConfig` SHALL NOT exist.

`GlobalConfig` SHALL contain the following fields:

- `Kubernetes` (`KubernetesConfig`): Kubernetes settings from config file
- `Log` (`LogConfig`): Logging settings from config file
- `Registry` (`string`): Resolved registry URL (after flag > env > config precedence)
- `ConfigPath` (`string`): Resolved config file path
- `Flags` (`GlobalFlags`): Raw CLI flag values

`GlobalConfig` SHALL NOT contain a `Providers` field or a `CueContext` field. CUE evaluation contexts are owned by the per-invocation kernel, not by configuration.

#### Scenario: GlobalConfig contains only surviving fields

- **WHEN** `GlobalConfig` is fully populated by the loader
- **THEN** `cfg.Kubernetes.Namespace` SHALL be accessible directly
- **THEN** `cfg.Log.Kubernetes.APIWarnings` SHALL be accessible directly
- **THEN** `GlobalConfig` SHALL NOT have a field named `Providers`
- **THEN** `GlobalConfig` SHALL NOT have a field named `CueContext`

#### Scenario: Config and OPMConfig types do not exist

- **WHEN** the project is compiled
- **THEN** there SHALL be no exported type named `Config` in the `config` package
- **THEN** there SHALL be no exported type named `OPMConfig` in the `config` package

### Requirement: Loader populates GlobalConfig directly

The `config.Load` function SHALL accept a `*GlobalConfig` and a `LoaderOptions` struct, and SHALL populate the config file fields, resolved runtime state, and config path directly on the provided struct. It SHALL return only an error.

The loader SHALL NOT return a separate config struct, SHALL NOT construct a CUE context for the caller, and SHALL NOT extract providers.

#### Scenario: Load populates all fields on GlobalConfig

- **WHEN** `config.Load(&cfg, opts)` succeeds
- **THEN** `cfg.Kubernetes`, `cfg.Log`, `cfg.Registry`, and `cfg.ConfigPath` SHALL be populated
- **THEN** `cfg.Flags` SHALL NOT be modified by the loader

#### Scenario: Load with no config file uses defaults

- **WHEN** no config file exists at the resolved path
- **THEN** `cfg.Kubernetes.Kubeconfig` SHALL be `"~/.kube/config"`
- **THEN** `cfg.Kubernetes.Namespace` SHALL be `"default"`
- **THEN** `cfg.Log.Kubernetes.APIWarnings` SHALL be `"warn"`

#### Scenario: Load resolves registry and config path

- **WHEN** `config.Load(&cfg, opts)` succeeds
- **THEN** `cfg.Registry` SHALL contain the resolved registry (flag > env > config precedence)
- **THEN** `cfg.ConfigPath` SHALL contain the resolved config file path (flag > env > default precedence)

### Requirement: Resolver accepts GlobalConfig directly

`ResolveKubernetesOptions` SHALL accept a `Config *GlobalConfig` field. The resolver SHALL extract Kubernetes config values from `opts.Config.Kubernetes.*` internally. The resolver SHALL NOT perform provider resolution of any kind (`resolveProvider` does not exist).

#### Scenario: ResolveKubernetes reads from GlobalConfig

- **WHEN** `ResolveKubernetes` is called with `Config` set to a `*GlobalConfig` that has `Kubernetes.Namespace: "staging"`
- **WHEN** no flag or env override is set for namespace
- **THEN** namespace SHALL resolve to `"staging"` with source `config`

#### Scenario: ResolveKubernetes handles nil Config

- **WHEN** `ResolveKubernetes` is called with `Config` set to `nil`
- **THEN** all fields SHALL resolve to their defaults
- **THEN** no panic SHALL occur

#### Scenario: No provider resolution

- **WHEN** the project is compiled
- **THEN** the `config` package SHALL NOT contain a function named `resolveProvider`
- **THEN** the resolved-Kubernetes result SHALL NOT contain a `Provider` field
