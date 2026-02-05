# CLI Validation

## Purpose

Defines the validation commands for OPM CLI: `mod vet` (module validation) and `config vet` (configuration validation). Uses Go CUE SDK directly for custom error formatting and programmatic control.

## User Stories

### Module author validating work

As a Module Author, I need immediate feedback on syntax errors and schema violations before attempting to build or apply, so I can catch errors early in development.

### Module author validating with concrete values

As a Module Author, I need to ensure my module has complete, concrete values suitable for rendering before publishing or sharing.

### Platform operator validating configuration

As a Platform Operator, I need to validate my configuration before running module commands, so config errors don't cause cryptic failures during module operations.

## Requirements

### Module Validation (`mod vet`)

- **FR-001**: The CLI MUST provide `opm mod vet` using the Go CUE SDK.
- **FR-002**: The command MUST accept an optional path argument defaulting to current directory.
- **FR-003**: The command MUST support `--package` flag (short: `-p`) to specify CUE package (default: `main`).
- **FR-004**: The command MUST support `--debug` flag to prefer `debug_values.cue` over `values.cue`.
- **FR-005**: The command MUST support `--concrete` flag requiring all values to be concrete.
- **FR-006**: The command MUST support `--values` flag (short: `-f`) accepting multiple values files (CUE, YAML, JSON).

### Module Validation Phases

- **FR-007**: Phase 1 - Load & Build: Parse CUE files, resolve imports via registry, build unified value.
- **FR-008**: Phase 2 - Schema Validation: Validate against `core.#Module`, `#Component`, `#Scope` schemas.
- **FR-009**: Phase 3 - Concrete Validation (optional): With `--concrete`, validate all values are concrete.

### Config Validation (`config vet`)

- **FR-020**: The CLI MUST provide `opm config vet` using the Go CUE SDK.
- **FR-021**: The command MUST accept an optional path argument defaulting to `~/.opm/`.
- **FR-022**: The command MUST support `--registry` flag to override registry for CUE module resolution.
- **FR-023**: Config validation MUST NOT support `--concrete` (config has optional fields by design).

### Config Validation Phases

- **FR-024**: Phase 1 - Load & Build: Parse config.cue, resolve imports via registry, build unified value.
- **FR-025**: Phase 2 - Schema Validation: Validate against `core.#Config` schema.

### Output & Error Formatting

- **FR-030**: On success, display validated entities summary (module name, component count, scope count).
- **FR-031**: On failure, output all errors with file location (file:line:col), description, and suggested fix.
- **FR-032**: Use Charm ecosystem logging format with color-coded severity.
- **FR-033**: Exit codes: `0` success, `2` validation errors, `1` other errors.
- **FR-034**: Aggregate and display all errors before exiting (fail-on-end pattern).

### Schema Dependencies

- **FR-040**: Module validation MUST validate against `core.#Module` from `opmodel.dev/core@v0`.
- **FR-041**: Config validation MUST validate against `core.#Config` from `opmodel.dev/core@v0`.

### Config Initialization

- **FR-050**: The `opm config init` command MUST generate configs that import and unify with `core.#Config`.
- **FR-051**: Generated configs MUST pass `opm config vet` without errors.
- **FR-052**: The config template MUST import `opmodel.dev/core@v0` and use `core.#Config &` for self-validation.

### Performance

- **NFR-001**: Module validation MUST complete in under 5 seconds for 20 components (warm cache).
- **NFR-002**: Error messages MUST be actionable and beginner-friendly.
- **NFR-003**: Config validation MUST complete in under 2 seconds.

## Design Rationale

### Why Go CUE SDK over shelling out

Direct SDK use provides better error context, performance, and integration with the CLI's config system. Custom error formatting via Charm ecosystem wouldn't be possible when shelling out to `cue vet`.

### Why fail-on-end pattern

Aggregating all errors before exiting helps developers fix multiple issues in a single iteration rather than playing whack-a-mole with sequential error discovery.

### Why no --concrete for config

Config files have optional fields by design (e.g., `context?: string`). Requiring concreteness would force users to specify every optional field.

### Why no provider connectivity validation

Provider connectivity is a runtime concern, not a schema validation concern. Validating that providers can be fetched from the registry introduces network dependencies, makes validation non-deterministic, and prevents offline development. Schema validation should be fast and offline-capable.

### Why self-validating configs

By making generated configs unify with `core.#Config`, schema violations are caught at CUE evaluation time - not just when running `opm config vet`. This provides earlier feedback and ensures consistency.

## Related

- **Implementation**: `cli/internal/validator/`, `cli/internal/config/templates.go`
- **Schema**: `catalog/v0/core/config.cue`, `catalog/v0/core/module.cue`
- **CLI Core**: `cli/openspec/specs/core/spec.md`
