## MODIFIED Requirements

### Requirement: Configuration precedence chain

The CLI SHALL resolve ALL configuration values using precedence: Flag > Environment > Config > Default. Resolution SHALL occur once during `PersistentPreRunE` and the results SHALL be stored for access by all subcommands.

| Value | Flag | Env Var | Config Field | Default |
|-------|------|---------|--------------|---------|
| Registry | `--registry` | `OPM_REGISTRY` | `config.registry` | (none) |
| Config Path | `--config` | `OPM_CONFIG` | (n/a) | `~/.opm/config.cue` |
| Kubeconfig | `--kubeconfig` | `OPM_KUBECONFIG` | `kubernetes.kubeconfig` | `~/.kube/config` |
| Context | `--context` | `OPM_CONTEXT` | `kubernetes.context` | current-context |
| Namespace | `--namespace` | `OPM_NAMESPACE` | `kubernetes.namespace` | `default` |
| Provider | `--provider` | (n/a) | auto-resolved from providers | (none) |
| Timestamps | `--timestamps` | (n/a) | `log.timestamps` | `true` |

Each resolved value SHALL record the final value, the source it came from, and any shadowed values from lower-precedence sources.

Subcommands SHALL NOT call `LoadOPMConfig()` independently. They SHALL access the pre-resolved configuration via accessor functions.

Subcommand-local flags (e.g., `opm mod apply --namespace production`) SHALL override the globally-resolved value for that invocation.

#### Scenario: Flag overrides environment variable

- **WHEN** `--registry localhost:5001` flag is provided
- **THEN** registry resolves to `localhost:5001`
- **THEN** `OPM_REGISTRY` environment value is shadowed

#### Scenario: Environment overrides config file

- **WHEN** `OPM_NAMESPACE=production` is set
- **THEN** namespace resolves to `production`
- **THEN** `kubernetes.namespace` config value is shadowed

#### Scenario: Config file overrides default for kubeconfig

- **WHEN** config.cue contains `kubernetes: { kubeconfig: "/custom/kubeconfig" }`
- **WHEN** no `--kubeconfig` flag or `OPM_KUBECONFIG` env var is set
- **THEN** kubeconfig resolves to `/custom/kubeconfig` (not `~/.kube/config`)

#### Scenario: Config file overrides default for namespace

- **WHEN** config.cue contains `kubernetes: { namespace: "staging" }`
- **WHEN** no `--namespace` flag or `OPM_NAMESPACE` env var is set
- **THEN** namespace resolves to `staging` (not `default`)

#### Scenario: Default used when no override present

- **WHEN** no flag, env, or config specifies namespace
- **THEN** namespace resolves to `default`

#### Scenario: Subcommand local flag overrides global resolution

- **WHEN** config.cue contains `kubernetes: { namespace: "staging" }`
- **WHEN** user runs `opm mod apply --namespace production`
- **THEN** namespace resolves to `production` for that command invocation

#### Scenario: Timestamps flag overrides config

- **WHEN** `--timestamps=false` flag is provided
- **WHEN** config.cue contains `log: { timestamps: true }`
- **THEN** timestamps resolve to `false` (flag takes precedence)

#### Scenario: Timestamps config overrides default

- **WHEN** no `--timestamps` flag is provided
- **WHEN** config.cue contains `log: { timestamps: false }`
- **THEN** timestamps resolve to `false` (config overrides default of `true`)

#### Scenario: Configuration loaded once per CLI invocation

- **WHEN** any subcommand is executed (apply, build, diff, delete, status)
- **THEN** `LoadOPMConfig()` SHALL be called exactly once in `PersistentPreRunE`
- **THEN** subcommands SHALL access the pre-loaded config via `GetOPMConfig()`

### Requirement: Resolution tracking for debugging

The CLI SHALL track the source of each resolved configuration value for verbose logging.

Each resolved value records:

- The final value used
- The source (flag, env, config, config-auto, or default)
- Any shadowed values from lower-precedence sources

The CLI SHALL emit a single "initializing CLI" debug log line showing all resolved values when `--verbose` is specified. Redundant per-field debug log lines emitted during the config loading process SHALL be removed.

The following debug log lines SHALL be removed as redundant:

- `"bootstrap: extracted registry from config"` — subsumed by "resolved registry" and "initializing CLI"
- `"setting CUE_REGISTRY for config load"` — internal implementation detail
- `"extracted provider from config"` (per-provider iteration) — subsumed by provider loader output
- `"extracted providers from config"` (count) — subsumed by provider loader output

The following debug log lines SHALL be retained:

- `"resolved config path"` — shows config file location and source
- `"resolved registry"` — shows registry URL and source

#### Scenario: Verbose mode shows all resolved values

- **WHEN** `--verbose` flag is specified
- **THEN** the "initializing CLI" debug log SHALL include all resolved values: kubeconfig, context, namespace, config path, registry, and provider
- **THEN** each value SHALL show the final resolved value (not raw flag defaults)

#### Scenario: Verbose mode shows resolution source

- **WHEN** `--verbose` flag is specified
- **WHEN** `--namespace production` overrides `OPM_NAMESPACE=staging`
- **THEN** debug log shows: key=namespace, value=production, source=flag

#### Scenario: No redundant log lines during config loading

- **WHEN** `--verbose` flag is specified
- **THEN** the debug output SHALL NOT contain "bootstrap: extracted registry from config"
- **THEN** the debug output SHALL NOT contain "setting CUE_REGISTRY for config load"
- **THEN** the debug output SHALL NOT contain "extracted provider from config"
- **THEN** the debug output SHALL NOT contain "extracted providers from config"

## ADDED Requirements

### Requirement: Consolidated verbose debug output for build pipeline

The build pipeline SHALL emit a single "release built" debug log per release build. Duplicate log lines from internal build stages SHALL be removed.

The following debug log lines SHALL be removed as redundant:

- `"release built successfully"` in `ReleaseBuilder.Build()` and `ReleaseBuilder.BuildFromValue()` — duplicate of pipeline-level "release built"
- `"loading provider"` in `ProviderLoader.LoadProvider()` — redundant with the `"loaded provider"` summary that follows

The `"extracted transformer"` debug log SHALL use the FQN as the `name` field value and SHALL NOT include a separate `fqn` field.

#### Scenario: Single release-built log per build

- **WHEN** `--verbose` flag is specified
- **WHEN** a module is built via the render pipeline
- **THEN** exactly one "release built" debug log SHALL appear (from the pipeline level)
- **THEN** no "release built successfully" log SHALL appear from the release builder

#### Scenario: Transformer log uses FQN as name

- **WHEN** `--verbose` flag is specified
- **WHEN** transformers are extracted from a provider
- **THEN** the "extracted transformer" log SHALL show `name=kubernetes#opmodel.dev/providers/kubernetes/transformers@v0#DeploymentTransformer`
- **THEN** the log SHALL NOT include a separate `fqn=` field
