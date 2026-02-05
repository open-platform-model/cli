## ADDED Requirements

### Requirement: Two-phase configuration loading

The CLI SHALL load configuration using a two-phase process to resolve circular dependencies between config file and registry.

Phase 1 (Bootstrap): Extract the `registry` field from `~/.opm/config.cue` using regex pattern matching without full CUE parsing.

Phase 2 (Full Load): Set `CUE_REGISTRY` environment variable to the resolved registry, then evaluate the config file with full CUE import resolution.

#### Scenario: Config file imports provider definitions from registry

- **WHEN** config.cue contains `import prov "opmodel.dev/providers@v0"`
- **THEN** bootstrap extraction reads registry value via regex
- **THEN** CUE_REGISTRY is set before full CUE evaluation
- **THEN** provider imports resolve successfully

#### Scenario: Config file has no registry field

- **WHEN** config.cue does not contain a registry field
- **THEN** bootstrap extraction returns empty string
- **THEN** full load proceeds without CUE_REGISTRY set (unless env var provides it)

#### Scenario: Config file does not exist

- **WHEN** `~/.opm/config.cue` does not exist
- **THEN** bootstrap extraction returns empty string without error
- **THEN** full load returns default configuration values

### Requirement: Configuration precedence chain

The CLI SHALL resolve configuration values using precedence: Flag > Environment > Config > Default.

| Value | Flag | Env Var | Config Field | Default |
|-------|------|---------|--------------|---------|
| Registry | `--registry` | `OPM_REGISTRY` | `config.registry` | (none) |
| Config Path | `--config` | `OPM_CONFIG` | (n/a) | `~/.opm/config.cue` |
| Kubeconfig | `--kubeconfig` | `OPM_KUBECONFIG` | `kubernetes.kubeconfig` | `~/.kube/config` |
| Context | `--context` | `OPM_CONTEXT` | `kubernetes.context` | current-context |
| Namespace | `--namespace` | `OPM_NAMESPACE` | `kubernetes.namespace` | `default` |

#### Scenario: Flag overrides environment variable

- **WHEN** `--registry localhost:5001` flag is provided
- **THEN** registry resolves to `localhost:5001`
- **THEN** `OPM_REGISTRY` environment value is shadowed

#### Scenario: Environment overrides config file

- **WHEN** `OPM_NAMESPACE=production` is set
- **THEN** namespace resolves to `production`
- **THEN** `kubernetes.namespace` config value is shadowed

#### Scenario: Config file overrides default

- **WHEN** config.cue contains `kubernetes: { namespace: "staging" }`
- **THEN** namespace resolves to `staging` (not `default`)

#### Scenario: Default used when no override present

- **WHEN** no flag, env, or config specifies namespace
- **THEN** namespace resolves to `default`

### Requirement: CUE-native configuration format

The CLI SHALL use CUE format for configuration (`~/.opm/config.cue`) with schema validation at load time.

#### Scenario: Valid CUE configuration loads successfully

- **WHEN** config.cue contains valid CUE with proper schema
- **THEN** configuration loads without error
- **THEN** CUE constraints are enforced (e.g., `namespace: *"default" | string`)

#### Scenario: Invalid CUE configuration fails with actionable error

- **WHEN** config.cue contains syntax errors
- **THEN** load fails with error message
- **THEN** error includes hint: "Run 'opm config vet' to check for configuration errors"

### Requirement: Standard filesystem paths

The CLI SHALL use `~/.opm/` as the home directory for all configuration files.

| Path | Purpose |
|------|---------|
| `~/.opm/` | OPM home directory |
| `~/.opm/config.cue` | Main configuration file |
| `~/.opm/cue.mod/module.cue` | CUE module metadata |

#### Scenario: Paths expand tilde to user home

- **WHEN** path contains `~`
- **THEN** `~` expands to user's home directory via `os.UserHomeDir()`

#### Scenario: Config path overridden by environment

- **WHEN** `OPM_CONFIG=/custom/path/config.cue` is set
- **THEN** config loads from `/custom/path/config.cue` instead of default

### Requirement: Fail-fast on missing registry with providers

The CLI SHALL fail immediately with actionable error if providers are configured but no registry is resolvable.

#### Scenario: Providers configured without registry

- **WHEN** config.cue contains `providers: { kubernetes: ... }`
- **WHEN** no registry is resolvable from flag, env, or config
- **THEN** load fails with validation error
- **THEN** error message: "providers configured but no registry resolvable"
- **THEN** hint: "Set OPM_REGISTRY environment variable, use --registry flag, or add registry field to config.cue"

#### Scenario: No providers configured without registry

- **WHEN** config.cue does not reference providers
- **WHEN** no registry is resolvable
- **THEN** load succeeds (registry not required)

### Requirement: Resolution tracking for debugging

The CLI SHALL track the source of each resolved configuration value for verbose logging.

Each resolved value records:

- The final value used
- The source (flag, env, config, or default)
- Any shadowed values from lower-precedence sources

#### Scenario: Verbose mode shows resolution chain

- **WHEN** `--verbose` flag is specified
- **WHEN** `--namespace production` overrides `OPM_NAMESPACE=staging`
- **THEN** debug log shows: key=namespace, value=production, source=flag
- **THEN** debug log shows: shadowed_source=env, shadowed_value=staging
