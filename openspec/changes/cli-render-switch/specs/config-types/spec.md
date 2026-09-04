## ADDED Requirements

### Requirement: Skew policy configuration key

`config.cue` SHALL accept an optional `skewPolicy` key with the values `"warn"` and `"refuse"`, defaulting to `"warn"` when absent (0019 D18). The value SHALL govern how a render responds when a module's `cue.mod` requires a newer build of an OPM-namespace path (core or a catalog) than the platform pins: `warn` renders against the platform's build and reports the skew; `refuse` fails the render before evaluation. The key applies when the platform is the local default or a `--platform` directory; when the cluster Platform CR is the source, the CR's `spec.skewPolicy` SHALL take precedence so CLI and operator judge the same platform the same way. Any other value SHALL fail config validation naming the allowed values. No flag SHALL exist for it.

#### Scenario: Default is warn

- **WHEN** `config.cue` carries no `skewPolicy`
- **THEN** renders against a local or flag platform use the warn policy

#### Scenario: Refuse configured

- **WHEN** `config.cue` sets `skewPolicy: "refuse"` and a render against the local platform hits a newer module requirement
- **THEN** the render is refused before evaluation naming the path and both versions

#### Scenario: Cluster CR overrides the key

- **WHEN** `config.cue` sets `skewPolicy: "refuse"` and the render resolves the cluster Platform CR whose `spec.skewPolicy` is `Warn`
- **THEN** the render uses warn and reports the CR as the policy's source

#### Scenario: Invalid value rejected

- **WHEN** `config.cue` sets `skewPolicy: "strict"`
- **THEN** config validation fails naming the allowed values `warn` and `refuse`
