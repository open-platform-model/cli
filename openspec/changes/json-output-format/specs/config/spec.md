## MODIFIED Requirements

### Requirement: Configuration precedence chain

The CLI SHALL resolve ALL configuration values using precedence: Flag > Environment > Config > Default. Resolution SHALL occur once during `PersistentPreRunE` and the results SHALL be stored for access by all subcommands.

| Value | Flag | Env Var | Config Field | Default |
|-------|------|---------|--------------|---------|
| Registry | `--registry` | `OPM_REGISTRY` | `config.registry` | (none) |
| Config Path | `--config` | `OPM_CONFIG` | (n/a) | `~/.opm/config.cue` |
| Format | `--format` | `OPM_FORMAT` | `config.format` | `text` |
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

#### Scenario: Format flag overrides environment

- **WHEN** `OPM_FORMAT=text` is set
- **WHEN** user runs `opm version --format json`
- **THEN** format resolves to `json`

#### Scenario: Format environment overrides config

- **WHEN** config.cue contains `format: "text"`
- **WHEN** `OPM_FORMAT=json` is set
- **THEN** format resolves to `json`

#### Scenario: Format config overrides default

- **WHEN** config.cue contains `format: "json"`
- **WHEN** no `--format` flag or `OPM_FORMAT` env var is set
- **THEN** format resolves to `json` (not `text`)

#### Scenario: Configuration loaded once per CLI invocation

- **WHEN** any subcommand is executed (apply, build, diff, delete, status)
- **THEN** `LoadOPMConfig()` SHALL be called exactly once in `PersistentPreRunE`
- **THEN** subcommands SHALL access the pre-loaded config via `GetOPMConfig()`
