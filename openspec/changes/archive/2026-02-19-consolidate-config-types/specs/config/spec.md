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

Subcommands SHALL NOT call `config.Load()` independently. They SHALL access the pre-loaded configuration via the `*config.GlobalConfig` passed from the root command.

Subcommand-local flags (e.g., `opm mod apply --namespace production`) SHALL override the globally-resolved value for that invocation.

Registry and config path resolution SHALL be performed by `config.Load` during `PersistentPreRunE`. Kubernetes config resolution SHALL be performed by `config.ResolveKubernetes` at the point of use in each subcommand. There SHALL NOT be a separate `ResolveBase` or `ResolveAll` resolution step.

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
- **THEN** `config.Load()` SHALL be called exactly once in `PersistentPreRunE`
- **THEN** subcommands SHALL access the pre-loaded config via the `*config.GlobalConfig` pointer

#### Scenario: No separate base resolution step

- **WHEN** `PersistentPreRunE` runs
- **THEN** `config.Load` SHALL resolve registry and config path
- **THEN** there SHALL NOT be a separate `ResolveBase` call after loading
