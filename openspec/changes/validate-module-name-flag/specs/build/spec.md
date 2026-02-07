## MODIFIED Requirements

### Requirement: FR-B-034 — Name override validation

`--name` MUST take precedence over `module.metadata.name`. The override value MUST be validated against RFC 1123 DNS label format (`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`, 1-63 characters). Invalid names MUST be rejected with a validation error.

#### Scenario: Valid kebab-case name accepted

- **WHEN** user runs `opm mod apply ./module --name my-blog`
- **THEN** `ModuleMetadata.Name` SHALL be `"my-blog"`
- **AND** the `module.opmodel.dev/name` label on applied resources SHALL be `"my-blog"`

#### Scenario: Single-word lowercase name accepted

- **WHEN** user runs `opm mod apply ./module --name blog`
- **THEN** `ModuleMetadata.Name` SHALL be `"blog"`

#### Scenario: PascalCase name rejected

- **WHEN** user runs `opm mod apply ./module --name MyBlog`
- **THEN** the CLI SHALL exit with a validation error
- **AND** the error message SHALL indicate the name must be RFC 1123 DNS label format
- **AND** the error message SHALL include an example of valid format (e.g., `"my-blog"`)

#### Scenario: Uppercase name rejected

- **WHEN** user runs `opm mod apply ./module --name MY-BLOG`
- **THEN** the CLI SHALL exit with a validation error

#### Scenario: Name starting with hyphen rejected

- **WHEN** user runs `opm mod apply ./module --name -my-blog`
- **THEN** the CLI SHALL exit with a validation error

#### Scenario: Name exceeding 63 characters rejected

- **WHEN** user runs `opm mod apply ./module --name` with a value longer than 63 characters
- **THEN** the CLI SHALL exit with a validation error

## ADDED Requirements

### Requirement: FR-B-071 — Delete fails when module not found

`kubernetes.Delete()` MUST return an `ErrNotFound` error when `DiscoverResources()` returns zero resources. The CLI MUST exit with code 5 (`ExitNotFound`).

#### Scenario: Delete of non-existent module returns error

- **WHEN** user runs `opm mod delete --name nonexistent -n default --force`
- **AND** no resources with `module.opmodel.dev/name=nonexistent` exist in namespace `default`
- **THEN** the CLI SHALL print an error message containing the module name and namespace
- **AND** the error SHALL include a hint that module names are case-sensitive and lowercase kebab-case
- **AND** the exit code SHALL be 5

#### Scenario: Delete of existing module succeeds

- **WHEN** user runs `opm mod delete --name blog -n default --force`
- **AND** resources with `module.opmodel.dev/name=blog` exist in namespace `default`
- **THEN** the CLI SHALL delete all matching resources
- **AND** the exit code SHALL be 0

### Requirement: FR-B-072 — Delete label matching is strict

`BuildModuleSelector()` MUST NOT normalize the `--name` input. Label matching is exact and case-sensitive against the stored label values.

#### Scenario: Case mismatch fails delete

- **WHEN** user runs `opm mod delete --name Blog -n default --force`
- **AND** resources exist with label `module.opmodel.dev/name=blog`
- **THEN** the selector `module.opmodel.dev/name=Blog` SHALL match zero resources
- **AND** the CLI SHALL exit with code 5 and an error message
