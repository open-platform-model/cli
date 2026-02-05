## ADDED Requirements

### Requirement: Module init with template

The CLI SHALL support `opm mod init <name> --template <ref>` to initialize modules from templates.

#### Scenario: Init from shorthand

- **WHEN** user runs `opm mod init my-app --template standard`
- **THEN** the CLI SHALL fetch the template and render to `./my-app/`

#### Scenario: Init from OCI reference

- **WHEN** user runs `opm mod init my-app --template oci://ghcr.io/org/tpl:v1`
- **THEN** the CLI SHALL fetch from the specified registry and render

#### Scenario: Init from local template

- **WHEN** user runs `opm mod init my-app --template file://./my-tpl`
- **THEN** the CLI SHALL use the local template without network access

---

### Requirement: Placeholder substitution

Templates SHALL support three standard placeholders using Go text/template syntax.

#### Scenario: ModuleName substitution

- **WHEN** template contains `{{.ModuleName}}`
- **THEN** the rendered output SHALL contain the module name from the command

#### Scenario: ModulePath substitution

- **WHEN** template contains `{{.ModulePath}}`
- **THEN** the rendered output SHALL contain the derived module path

#### Scenario: Version substitution

- **WHEN** template contains `{{.Version}}`
- **THEN** the rendered output SHALL contain `0.1.0` (default initial version)

---

### Requirement: ModulePath derivation

The CLI SHALL derive ModulePath from the module name when not explicitly provided.

#### Scenario: Default path derivation

- **WHEN** user runs `opm mod init my-app` without `--module` flag
- **THEN** ModulePath SHALL be `example.com/my_app` (hyphens converted to underscores)

#### Scenario: Explicit module path

- **WHEN** user runs `opm mod init my-app --module acme.com/my-app`
- **THEN** ModulePath SHALL use the provided value

---

### Requirement: Template file rendering

The CLI SHALL render `.tmpl` files and remove the `.tmpl` suffix.

#### Scenario: File suffix removal

- **WHEN** template contains `module.cue.tmpl`
- **THEN** the rendered file SHALL be `module.cue`

#### Scenario: Directory preservation

- **WHEN** template contains `cue.mod/module.cue.tmpl`
- **THEN** the rendered file SHALL be at `cue.mod/module.cue`

---

### Requirement: Generated module validation

Modules generated from official templates SHALL pass validation.

#### Scenario: Valid generated module

- **WHEN** `opm mod init my-app --template standard` completes
- **THEN** running `opm mod vet` in `./my-app/` SHALL pass without errors

---

### Requirement: Template caching

The CLI SHALL cache fetched templates to avoid redundant downloads.

#### Scenario: Cache hit

- **WHEN** user inits from the same template twice
- **THEN** the second init SHALL use the cached template (no network request)

#### Scenario: Cache location

- **WHEN** a template is fetched
- **THEN** it SHALL be cached at `~/.cache/cue/mod/extract/<registry>/<path>/<version>/`

---

### Requirement: Force overwrite

The CLI SHALL support `--force` to overwrite existing module directories.

#### Scenario: Directory exists without force

- **WHEN** user runs `opm mod init my-app` and `./my-app/` exists
- **THEN** the CLI SHALL exit with error

#### Scenario: Directory exists with force

- **WHEN** user runs `opm mod init my-app --force` and `./my-app/` exists
- **THEN** the CLI SHALL overwrite the existing directory
