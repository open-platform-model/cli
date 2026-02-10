## MODIFIED Requirements

### Requirement: Configuration precedence chain

The CLI SHALL resolve configuration values using precedence: Flag > Environment > Config > Default.

| Value | Flag | Env Var | Config Field | Default |
|-------|------|---------|--------------|---------|
| Registry | `--registry` | `OPM_REGISTRY` | `config.registry` | (none) |
| Config Path | `--config` | `OPM_CONFIG` | (n/a) | `~/.opm/config.cue` |
| Kubeconfig | `--kubeconfig` | `OPM_KUBECONFIG` | `kubernetes.kubeconfig` | `~/.kube/config` |
| Context | `--context` | `OPM_CONTEXT` | `kubernetes.context` | current-context |
| Namespace | `--namespace` | `OPM_NAMESPACE` | `kubernetes.namespace` | `default` |
| Timestamps | `--timestamps` | (n/a) | `log.timestamps` | `true` |

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

#### Scenario: Timestamps flag overrides config

- **WHEN** `--timestamps=false` flag is provided
- **WHEN** config.cue contains `log: { timestamps: true }`
- **THEN** timestamps resolve to `false` (flag takes precedence)

#### Scenario: Timestamps config overrides default

- **WHEN** no `--timestamps` flag is provided
- **WHEN** config.cue contains `log: { timestamps: false }`
- **THEN** timestamps resolve to `false` (config overrides default of `true`)

## ADDED Requirements

### Requirement: Log configuration section in CUE config

The CUE configuration schema SHALL support an optional `log` section for controlling log output behavior.

The schema SHALL be:

```cue
log?: {
    timestamps: bool | *true
}
```

The `log` section SHALL be optional. When omitted, all log settings SHALL use their default values.

#### Scenario: Config with log section loads successfully

- **WHEN** config.cue contains `log: { timestamps: false }`
- **THEN** configuration loads without error
- **THEN** `config.log.timestamps` resolves to `false`

#### Scenario: Config without log section uses defaults

- **WHEN** config.cue does not contain a `log` section
- **THEN** configuration loads without error
- **THEN** log timestamps default to `true`

#### Scenario: Invalid log config value rejected at load time

- **WHEN** config.cue contains `log: { timestamps: "yes" }`
- **THEN** configuration loading SHALL fail with a CUE validation error
- **THEN** the error SHALL indicate that `timestamps` must be a boolean
