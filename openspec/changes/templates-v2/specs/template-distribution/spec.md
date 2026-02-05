## ADDED Requirements

### Requirement: Template manifest format

Templates SHALL include a `template.cue` manifest at the root defining metadata.

#### Scenario: Valid manifest

- **WHEN** a template directory contains `template.cue` with name, version, and description fields
- **THEN** the CLI SHALL accept it as a valid template

#### Scenario: Missing manifest

- **WHEN** a template directory has no `template.cue` file
- **THEN** the CLI SHALL exit with error code 2 and message "template.cue manifest not found"

#### Scenario: Invalid manifest schema

- **WHEN** `template.cue` is missing required fields (name, version, description)
- **THEN** the CLI SHALL exit with error code 2 and describe the missing fields

---

### Requirement: Manifest field validation

The manifest SHALL validate field formats per the schema.

#### Scenario: Valid name format

- **WHEN** manifest has `name: "my-template"` (lowercase alphanumeric with hyphens)
- **THEN** validation SHALL pass

#### Scenario: Invalid name format

- **WHEN** manifest has `name: "My_Template"` (uppercase or underscores)
- **THEN** the CLI SHALL exit with error code 2 and message about name format

#### Scenario: Description minimum length

- **WHEN** manifest has `description: "Short"` (less than 10 characters)
- **THEN** the CLI SHALL exit with error code 2 and message "description must be at least 10 characters"

---

### Requirement: Template file structure

Templates SHALL contain at least one `.tmpl` file for rendering.

#### Scenario: Template with tmpl files

- **WHEN** a template directory contains `module.cue.tmpl`
- **THEN** the CLI SHALL recognize it as a template file

#### Scenario: No tmpl files

- **WHEN** a template directory has `template.cue` but no `.tmpl` files
- **THEN** the CLI SHALL exit with error code 2 and message "template must contain at least one .tmpl file"

---

### Requirement: OCI artifact packaging

Templates SHALL be packaged as OCI artifacts with defined media types.

#### Scenario: Template push creates OCI artifact

- **WHEN** `opm template publish` is executed
- **THEN** the CLI SHALL create an OCI artifact with config media type `application/vnd.opmodel.template.config.v1+json`

#### Scenario: Template content layer

- **WHEN** a template is published
- **THEN** the content layer SHALL use media type `application/vnd.opmodel.template.content.v1.tar+gzip`

---

### Requirement: Reference resolution

The CLI SHALL resolve template references in multiple formats.

#### Scenario: Shorthand reference

- **WHEN** user specifies `--template standard`
- **THEN** the CLI SHALL resolve to `oci://${REGISTRY}/templates/standard:latest`

#### Scenario: Explicit OCI reference

- **WHEN** user specifies `--template oci://ghcr.io/org/tpl:v1`
- **THEN** the CLI SHALL use the reference as-is

#### Scenario: Implicit OCI reference

- **WHEN** user specifies `--template ghcr.io/org/tpl:v1` (contains `/`, no scheme)
- **THEN** the CLI SHALL prepend `oci://` and resolve

#### Scenario: Local file reference

- **WHEN** user specifies `--template file://./my-template`
- **THEN** the CLI SHALL read from local filesystem path `./my-template`

---

### Requirement: Registry precedence

Shorthand references SHALL resolve using registry precedence chain.

#### Scenario: Registry flag takes priority

- **WHEN** user specifies `--registry custom.io` and `--template standard`
- **THEN** the CLI SHALL resolve to `oci://custom.io/templates/standard:latest`

#### Scenario: Environment variable fallback

- **WHEN** `OPM_REGISTRY=env.io` is set and no `--registry` flag
- **THEN** shorthand references SHALL resolve using `env.io`

#### Scenario: Default registry

- **WHEN** no registry is configured
- **THEN** shorthand references SHALL resolve using `registry.opmodel.dev`
