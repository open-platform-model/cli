## MODIFIED Requirements

### Requirement: FR-B-034 — Name override normalization

`--name` MUST take precedence over `module.metadata.name`. The override value MUST be lowercased before being assigned to `ModuleMetadata.Name`.

#### Scenario: Name override is lowercased

- **WHEN** user runs `opm mod apply ./module --name MyBlog`
- **THEN** `ModuleMetadata.Name` SHALL be `"myblog"`
- **AND** the `module.opmodel.dev/name` label on applied resources SHALL be `"myblog"`

#### Scenario: Lowercase override is unchanged

- **WHEN** user runs `opm mod apply ./module --name my-blog`
- **THEN** `ModuleMetadata.Name` SHALL be `"my-blog"`

## ADDED Requirements

### Requirement: FR-B-070 — Label name normalization

`injectLabels()` MUST lowercase `meta.Name` before writing to the `module.opmodel.dev/name` label. This is a defensive normalization independent of the `--name` override.

#### Scenario: CUE-authored PascalCase name is lowercased in labels

- **WHEN** a module defines `metadata.name: "Blog"` and is applied without `--name` override
- **THEN** the `module.opmodel.dev/name` label on all applied resources SHALL be `"blog"`
- **AND** log output MAY display the original name `"Blog"`

#### Scenario: Hyphenated name is preserved in labels

- **WHEN** a module defines `metadata.name: "my-blog"` and is applied
- **THEN** the `module.opmodel.dev/name` label SHALL be `"my-blog"`

### Requirement: FR-B-071 — Delete fails when module not found

`kubernetes.Delete()` MUST return an `ErrNotFound` error when `DiscoverResources()` returns zero resources. The CLI MUST exit with code 5 (`ExitNotFound`).

#### Scenario: Delete of non-existent module returns error

- **WHEN** user runs `opm mod delete --name nonexistent -n default --force`
- **AND** no resources with `module.opmodel.dev/name=nonexistent` exist in namespace `default`
- **THEN** the CLI SHALL print an error message containing the module name and namespace
- **AND** the error SHALL include a hint that module names are case-sensitive and lowercase
- **AND** the exit code SHALL be 5

#### Scenario: Delete of existing module succeeds

- **WHEN** user runs `opm mod delete --name blog -n default --force`
- **AND** resources with `module.opmodel.dev/name=blog` exist in namespace `default`
- **THEN** the CLI SHALL delete all matching resources
- **AND** the exit code SHALL be 0

### Requirement: FR-B-072 — Delete label matching is strict

`BuildModuleSelector()` MUST NOT normalize the `--name` input. Label matching is exact and case-sensitive against the stored (lowercase) label values.

#### Scenario: Case mismatch fails delete

- **WHEN** user runs `opm mod delete --name Blog -n default --force`
- **AND** resources exist with label `module.opmodel.dev/name=blog`
- **THEN** the selector `module.opmodel.dev/name=Blog` SHALL match zero resources
- **AND** the CLI SHALL exit with code 5 and an error message
