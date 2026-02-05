## ADDED Requirements

### Requirement: Template list command

The CLI SHALL provide `opm template list` to discover available templates.

#### Scenario: List from default registry

- **WHEN** user runs `opm template list`
- **THEN** the CLI SHALL display templates from the configured registry in table format (NAME, VERSION, DESCRIPTION)

#### Scenario: List with JSON output

- **WHEN** user runs `opm template list -o json`
- **THEN** the CLI SHALL output JSON with template metadata array

#### Scenario: Registry unreachable

- **WHEN** the registry is unreachable
- **THEN** the CLI SHALL exit with error code 3 (Connectivity Error) and descriptive message

---

### Requirement: Template show command

The CLI SHALL provide `opm template show <ref>` to inspect template details.

#### Scenario: Show template metadata

- **WHEN** user runs `opm template show standard`
- **THEN** the CLI SHALL display name, version, description, placeholders, and file tree

#### Scenario: Show with file tree

- **WHEN** a template has nested directories
- **THEN** `opm template show` SHALL display the directory structure

#### Scenario: Template not found

- **WHEN** user runs `opm template show unknown`
- **THEN** the CLI SHALL exit with error code 5 (Not Found)

---

### Requirement: Template get command

The CLI SHALL provide `opm template get <ref>` to download templates for editing.

#### Scenario: Download to default directory

- **WHEN** user runs `opm template get standard`
- **THEN** the CLI SHALL download template files to `./standard/`

#### Scenario: Download to custom directory

- **WHEN** user runs `opm template get standard --dir ./my-tpl`
- **THEN** the CLI SHALL download template files to `./my-tpl/`

#### Scenario: Target directory exists

- **WHEN** target directory exists and is non-empty
- **THEN** the CLI SHALL exit with error unless `--force` is provided

#### Scenario: Force overwrite

- **WHEN** user runs `opm template get standard --dir ./exists --force`
- **THEN** the CLI SHALL overwrite the existing directory

---

### Requirement: Template validate command

The CLI SHALL provide `opm template validate` to validate local templates.

#### Scenario: Valid template

- **WHEN** user runs `opm template validate` in a valid template directory
- **THEN** the CLI SHALL output "Template is valid: <name> v<version>"

#### Scenario: Invalid template

- **WHEN** template has validation errors
- **THEN** the CLI SHALL exit with error code 2 and list all validation issues

---

### Requirement: Template publish command

The CLI SHALL provide `opm template publish <oci-ref>` to publish templates.

#### Scenario: Successful publish

- **WHEN** user runs `opm template publish ghcr.io/org/tpl:v1` in a valid template directory
- **THEN** the CLI SHALL validate, package, and push the template to the registry

#### Scenario: Pre-publish validation

- **WHEN** template has validation errors
- **THEN** `opm template publish` SHALL fail before pushing with error code 2

#### Scenario: Authentication

- **WHEN** registry requires authentication
- **THEN** the CLI SHALL use credentials from `~/.docker/config.json`

#### Scenario: Publish confirmation

- **WHEN** publish succeeds
- **THEN** the CLI SHALL display the published reference
