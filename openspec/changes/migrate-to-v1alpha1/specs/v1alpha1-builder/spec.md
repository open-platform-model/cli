## ADDED Requirements

### Requirement: Builder loads ModuleRelease schema from core v1

The builder SHALL load `#ModuleRelease` from `opmodel.dev/core@v1` (resolved from the module's pinned dependency cache). All error messages referencing the core package SHALL use `core@v1`.

#### Scenario: Core v1 schema loaded successfully

- **WHEN** the module's dependency cache contains `opmodel.dev@v1`
- **THEN** the builder SHALL load `opmodel.dev/core@v1` via `load.Instances`
- **THEN** `#ModuleRelease` SHALL be extracted from the built value

#### Scenario: Core v1 not found

- **WHEN** the module's dependency cache does not contain `opmodel.dev@v1`
- **THEN** the builder SHALL return an error containing `"opmodel.dev/core@v1"`

### Requirement: Config default module template uses v1 paths

The `DefaultModuleTemplate` in `config/templates.go` SHALL use `opmodel.dev/config@v1` as the module path. Provider import paths SHALL use `opmodel.dev/providers@v1`.

#### Scenario: Config init generates v1 module path

- **WHEN** `opm config init` generates a default config
- **THEN** the config CUE module declaration SHALL be `module: "opmodel.dev/config@v1"`

#### Scenario: Config provider import uses v1

- **WHEN** `opm config init` generates a default config with provider imports
- **THEN** the provider import SHALL reference `opmodel.dev/providers@v1`

### Requirement: No v0 references in production Go source

No production Go source file (excluding `experiments/`) SHALL contain hardcoded `@v0` CUE import paths. All references to `opmodel.dev/*@v0` in builder, loader, and config packages SHALL be updated to `@v1`.

#### Scenario: Builder has no v0 references

- **WHEN** `internal/builder/builder.go` is inspected
- **THEN** it SHALL contain `opmodel.dev/core@v1` and NOT contain `opmodel.dev/core@v0`

#### Scenario: Config has no v0 references

- **WHEN** `internal/config/templates.go` is inspected
- **THEN** it SHALL NOT contain any `@v0` catalog paths
