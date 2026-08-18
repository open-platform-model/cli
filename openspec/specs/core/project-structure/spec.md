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

- **FR-PS-005**: The official template set MUST be `minimal`, `standard`, and `advanced` (renaming `simple` to `minimal`), sourced as real published modules under `cli/templates/` rather than embedded text templates. `internal/templates/` and the embed machinery MUST NOT exist — the cli binary embeds no template file trees; templates reach users only as published modules, resolvable in the reserved segment at their declared versions once a cli release completes. The variants' authoritative definition is the `template-modules` capability; this requirement retains only the set's names and their cli-repo home. (Former FR-PS-006 and FR-PS-007 are folded into this requirement.)

### Validation

- **FR-PS-008**: The CLI MUST identify project root by searching for `cue.mod/` directory.
- **FR-PS-009**: If `module.cue` or `values.cue` is missing, the CLI MUST exit with code `2` (Validation Error).
- **FR-PS-010**: If a protected filename is used for an incompatible purpose, validation MUST fail.

## Design Rationale

### Why strict file naming

Strict conventions enable tooling to locate definitions without configuration. `module.cue` is always the entry point, `values.cue` always provides defaults.

### Why three templates

Different use cases need different complexity levels:

- `minimal`: Learning, prototypes (single file)
- `standard`: Team projects (separated concerns)
- `advanced`: Showcase (multiple components, trait attachments, values plumbing)

## Related

- **CLI Core**: `cli/openspec/specs/core/spec.md`
- **Template Modules**: `cli/openspec/specs/template-modules/spec.md` (source trees: `cli/templates/`)
