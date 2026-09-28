## MODIFIED Requirements

### Requirement: Synthetic instance metadata defaults

The synthesis SHALL produce a `metadata.name` and `metadata.namespace` for the synthetic `#ModuleInstance` even when the user has not supplied them. The default name SHALL be a valid instance name for every valid module name: a module name is a CUE package name, which may carry `_` but never `-`, while an instance name is a DNS label, which may carry `-` but never `_`.

#### Scenario: Default name derived from module metadata

- **WHEN** the user does not pass `--name`
- **THEN** the synthetic `metadata.name` SHALL be `"<module.metadata.name>-debug"` with every `_` in the module name replaced by `-`

#### Scenario: Multi-word module name

- **WHEN** a module named `my_app` is synthesized without `--name`
- **THEN** the synthetic `metadata.name` SHALL be `"my-app-debug"`
- **AND** synthesis SHALL NOT refuse the name

#### Scenario: Default namespace

- **WHEN** the user does not pass `--namespace`
- **THEN** the synthetic `metadata.namespace` SHALL be `"default"`

#### Scenario: Flag overrides

- **WHEN** the user passes `--name <n>` or `--namespace <ns>` (or both)
- **THEN** the synthetic `metadata.name` and/or `metadata.namespace` SHALL take the flag values
