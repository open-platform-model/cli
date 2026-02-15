# Capability: mod-vet

## Purpose

The `opm mod vet` command provides standalone module validation without generating manifests. It validates modules via the render pipeline and reports per-resource validation results, enabling fast feedback for module authors.

## Requirements

### Requirement: mod vet command validates module without generating manifests

The `opm mod vet` command SHALL build the module via the render pipeline and report per-resource validation results. It SHALL NOT output manifests (YAML/JSON). Its purpose is pass/fail validation with clear per-resource feedback.

The command SHALL accept a module path argument (default: current directory) and a subset of `mod build` flags for values, namespace, name, and provider resolution.

#### Scenario: Valid module passes validation with per-resource output

- **WHEN** `opm mod vet .` is run on a valid module
- **THEN** the command SHALL call `pipeline.Render()` to build the module
- **THEN** each generated resource SHALL be printed as a `FormatResourceLine` with `"valid"` status
- **THEN** a final summary line SHALL be printed: `FormatCheckmark("Module valid (<N> resources)")`
- **THEN** the command SHALL exit with code 0

#### Scenario: Module with CUE validation errors fails with details

- **WHEN** `opm mod vet .` is run on a module with values that do not satisfy `#config`
- **THEN** the command SHALL print the validation error using `printValidationError`
- **THEN** the error output SHALL include CUE error details formatted by `formatCUEDetails`
- **THEN** error paths SHALL use `values.` prefix (e.g., `values.media."test-key"`) instead of `#config.` prefix
- **THEN** every "field not allowed" error SHALL include at least one `file:line:col` position pointing to the values file that introduced the disallowed field
- **THEN** type mismatch errors SHALL include positions from both the schema file and the values file
- **THEN** the command SHALL exit with code 2

#### Scenario: Module with render errors fails with component details

- **WHEN** `opm mod vet .` is run on a module with unmatched components
- **THEN** the command SHALL print render errors using `printRenderErrors`
- **THEN** the command SHALL exit with code 2

#### Scenario: Module with zero resources after successful render

- **WHEN** `opm mod vet .` is run on a module that renders zero resources
- **THEN** the command SHALL print: `FormatCheckmark("Module valid (0 resources)")`
- **THEN** the command SHALL exit with code 0

#### Scenario: Multiple values files with disallowed fields show per-file attribution

- **WHEN** `opm mod vet . -f base.cue -f overrides.cue` is run
- **AND** `base.cue` contains a disallowed field `"extra-base"` at line 10
- **AND** `overrides.cue` contains a disallowed field `"extra-override"` at line 5
- **THEN** the error for `values."extra-base"` SHALL include `→ ./base.cue:10:...`
- **THEN** the error for `values."extra-override"` SHALL include `→ ./overrides.cue:5:...`

#### Scenario: Nested validation errors show full path with positions

- **WHEN** `opm mod vet .` is run on a module where values contain a type mismatch 3 levels deep
- **THEN** the error path SHALL show the full nested path (e.g., `values.media.movies.mountPath`)
- **THEN** the error SHALL include file:line:col positions for both the schema constraint and the data value

### Requirement: mod vet reuses the render pipeline

The `opm mod vet` command SHALL use `pipeline.Render()` — the same render pipeline used by `mod build` and `mod apply`. It SHALL NOT implement its own CUE loading, building, or validation logic.

The per-resource validation output logic SHALL be implemented in the `internal/output` package (not in the command package) so that it is reusable by other commands (`mod build --verbose`, future commands).

#### Scenario: mod vet uses the same pipeline as mod build

- **WHEN** `opm mod vet .` is run
- **THEN** the command SHALL create a `build.Pipeline` and call `Render(ctx, opts)` with `RenderOptions`
- **THEN** the `RenderResult` SHALL be consumed for validation output only (no manifest rendering)

#### Scenario: Validation output logic is in the output package

- **WHEN** `mod vet` needs to print per-resource validation lines
- **THEN** it SHALL call output package functions (`FormatResourceLine` with `StatusValid`)
- **THEN** the same functions SHALL be usable by `mod build --verbose` for identical formatting

### Requirement: mod vet accepts values files for validation

The `opm mod vet` command SHALL support `--values` / `-f` flags for providing external values files (CUE format), matching the behavior of `mod build`.

#### Scenario: Validate with external values

- **WHEN** `opm mod vet . -f prod-values.cue` is run
- **THEN** the render pipeline SHALL unify the external values with the module
- **THEN** validation SHALL use the merged values

### Requirement: mod vet supports verbose output

The `opm mod vet` command SHALL support `--verbose` / `-v` flag to show transformer matching details in addition to per-resource validation results.

#### Scenario: Verbose output shows matching decisions

- **WHEN** `opm mod vet . --verbose` is run on a valid module
- **THEN** the output SHALL include module metadata (name, namespace, version, components)
- **THEN** the output SHALL include transformer matching details per component
- **THEN** the output SHALL include per-resource validation lines

### Requirement: mod vet supports strict trait handling

The `opm mod vet` command SHALL support the `--strict` flag for strict trait handling, matching the behavior of `mod build --strict`. When enabled, unhandled traits cause errors instead of warnings.

#### Scenario: Strict mode errors on unhandled traits

- **WHEN** `opm mod vet . --strict` is run on a module with unhandled traits
- **THEN** the command SHALL fail with an error listing the unhandled traits
- **THEN** the command SHALL exit with code 2

#### Scenario: Default mode warns on unhandled traits

- **WHEN** `opm mod vet .` is run on a module with unhandled traits
- **THEN** warnings SHALL be printed to stderr via the module logger
- **THEN** per-resource validation output SHALL still be printed
- **THEN** the command SHALL exit with code 0

### Requirement: mod vet command flags and syntax

```text
opm mod vet [path] [flags]

Arguments:
  path    Path to module directory (default: .)

Flags:
  -f, --values strings      Additional values files (can be repeated)
  -n, --namespace string    Target namespace
      --name string         Release name (default: module name)
      --provider string     Provider to use (default: from config)
      --strict              Error on unhandled traits
  -v, --verbose             Show matching decisions and module metadata
  -h, --help                Help for vet
```

#### Scenario: Default flags match expected behavior

- **WHEN** `opm mod vet` is run without any flags
- **THEN** path SHALL default to `"."`
- **THEN** strict SHALL default to `false`
- **THEN** verbose SHALL default to `false`

### Requirement: mod vet exit codes

| Code | Meaning |
|------|---------|
| 0 | Validation passed |
| 1 | Usage error (invalid flags, missing arguments) |
| 2 | Validation error (CUE errors, unmatched components, render failures) |

#### Scenario: Exit code 0 on success

- **WHEN** `opm mod vet .` succeeds
- **THEN** the exit code SHALL be 0

#### Scenario: Exit code 2 on validation failure

- **WHEN** `opm mod vet .` fails due to CUE errors
- **THEN** the exit code SHALL be 2
