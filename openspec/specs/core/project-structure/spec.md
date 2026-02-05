# Project Structure

## Purpose

Defines the mandatory project structure, directory layout, and protected filenames for OPM Module projects.

## User Stories

### Module author understanding structure

As a Module Author, I need to understand the required file structure so that I can organize my module correctly and avoid validation errors.

### CLI validating projects

As the OPM CLI, I need to enforce consistent project structure so that modules are portable and predictable across environments.

## Requirements

### Mandatory Files

- **FR-PS-001**: Every Module project MUST contain `module.cue` at its root (main `#Module` definition).
- **FR-PS-002**: Every Module project MUST contain `values.cue` at its root (concrete default values).
- **FR-PS-003**: Every Module project MUST contain `cue.mod/module.cue` (CUE module configuration).

### Protected Filenames

- **FR-PS-004**: The following filenames are reserved and SHOULD only be used for their designated purpose:
  - `components.cue` - Component definitions
  - `scopes.cue` - Scope definitions  
  - `policies.cue` - Policy definitions
  - `debug_values.cue` - Extended values for validation and debugging

### Template Layouts

- **FR-PS-005**: `mod init --template simple` MUST create minimal single-file structure.
- **FR-PS-006**: `mod init --template standard` (default) MUST create separated concerns structure.
- **FR-PS-007**: `mod init --template advanced` MUST create multi-package architecture with subpackages.

### Validation

- **FR-PS-008**: The CLI MUST identify project root by searching for `cue.mod/` directory.
- **FR-PS-009**: If `module.cue` or `values.cue` is missing, the CLI MUST exit with code `2` (Validation Error).
- **FR-PS-010**: If a protected filename is used for an incompatible purpose, validation MUST fail.

## Design Rationale

### Why strict file naming

Strict conventions enable tooling to locate definitions without configuration. `module.cue` is always the entry point, `values.cue` always provides defaults.

### Why three templates

Different use cases need different complexity levels:

- `simple`: Learning, prototypes (single file)
- `standard`: Team projects (separated concerns)
- `advanced`: Enterprise (multi-package with subpackages)

## Related

- **CLI Core**: `cli/openspec/specs/core/spec.md`
- **Templates Implementation**: `cli/internal/templates/`
