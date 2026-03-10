# Capability: mod-vet

## Purpose

The `opm mod vet` command provides standalone module-config validation without generating manifests. It validates a module's `debugValues` or explicit values files against `#config`, enabling fast feedback for module authors.

## Requirements

### Requirement: mod vet command validates module without generating manifests

The `opm mod vet` command SHALL load the module directly and validate values against `#config`. It SHALL NOT render resources or output manifests (YAML/JSON). Its purpose is pass/fail config validation with clear diagnostics for module authors.

The command SHALL accept a module path argument (default: current directory) and values flags for supplying one or more external values files.

When no `-f`/`--values` flag is provided, `opm mod vet` SHALL use the module's `debugValues` field as the values source. If `-f` is provided, it SHALL override `debugValues` and use the specified values files instead. If neither `debugValues` nor `-f` exists, the command SHALL return an error indicating that values must be provided via `debugValues` in the module or via `-f`.

#### Scenario: Valid module passes validation using debugValues

- **WHEN** `opm mod vet .` is run on a module that defines `debugValues`
- **AND** no `-f` flag is provided
- **THEN** the command SHALL use `debugValues` as the values source
- **AND** it SHALL print `FormatVetCheck("Values satisfy #config", "debugValues")`
- **AND** a final summary line SHALL be printed: `FormatCheckmark("Module config valid")`
- **AND** the command SHALL exit with code 0

#### Scenario: -f flag overrides debugValues

- **WHEN** `opm mod vet . -f prod-values.cue` is run on a module that defines `debugValues`
- **THEN** the command SHALL use `prod-values.cue` as the values source
- **AND** `debugValues` SHALL be ignored

#### Scenario: No debugValues and no -f flag

- **WHEN** `opm mod vet .` is run on a module that does not define `debugValues` (or `debugValues` is `_`)
- **AND** no `-f` flag is provided
- **THEN** the command SHALL return an error directing the user to add `debugValues` or provide values with `-f`
- **AND** the exit code SHALL be 2

#### Scenario: Module with CUE validation errors fails with details

- **WHEN** `opm mod vet .` is run on a module with values that do not satisfy `#config`
- **THEN** the command SHALL print the validation error using `PrintValidationError`
- **THEN** error paths SHALL use `values.` prefix (e.g., `values.media."test-key"`) instead of `#config.` prefix
- **THEN** every "field not allowed" error SHALL include at least one `file:line:col` position pointing to the values file that introduced the disallowed field
- **THEN** type mismatch errors SHALL include positions from both the schema file and the values file
- **THEN** grouped error headers SHALL count visible issues rather than raw source locations
- **THEN** the command SHALL exit with code 2

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

#### Scenario: Multiple values files report schema errors and merge conflicts together

- **WHEN** `opm mod vet . -f base.cue -f overrides.cue` is run
- **AND** `base.cue` contains schema violations
- **AND** `base.cue` and `overrides.cue` also contain a conflicting assignment
- **THEN** the command SHALL report both the schema violations and the merge conflict in one run

### Requirement: mod vet does not use the render pipeline

The `opm mod vet` command SHALL NOT call the release render pipeline used by `mod build`, `mod apply`, or `opm rel vet`.

It SHALL:

1. Load the module package directly
2. Resolve values from `debugValues` or the supplied `-f` files
3. Ensure each supplied values input is concrete
4. Call `ValidateConfig` against the module's `#config`
5. Print validation output and exit without rendering resources

#### Scenario: mod vet loads module directly

- **WHEN** `opm mod vet .` is run
- **THEN** the command SHALL load the module package directly
- **AND** it SHALL NOT resolve a provider
- **AND** it SHALL NOT compute transformer matches
- **AND** it SHALL NOT render resources

### Requirement: mod vet accepts values files for validation

The `opm mod vet` command SHALL support `--values` / `-f` flags for providing external values files (CUE format), matching the behavior of `mod build`.

#### Scenario: Validate with external values

- **WHEN** `opm mod vet . -f prod-values.cue` is run
- **THEN** the command SHALL validate `prod-values.cue` against the module's `#config`
- **THEN** validation SHALL use the merged values

### Requirement: mod vet command flags and syntax

```text
opm mod vet [path] [flags]

Arguments:
  path    Path to module directory (default: .)

Flags:
  -f, --values strings      Additional values files (can be repeated)
  -h, --help                Help for vet
```

#### Scenario: Default flags match expected behavior

- **WHEN** `opm mod vet` is run without any flags
- **THEN** path SHALL default to `"."`

### Requirement: mod vet exit codes

| Code | Meaning |
|------|---------|
| 0 | Validation passed |
| 1 | Usage error (invalid flags, missing arguments) |
| 2 | Validation error (CUE errors, invalid values, missing `debugValues`) |

#### Scenario: Exit code 0 on success

- **WHEN** `opm mod vet .` succeeds
- **THEN** the exit code SHALL be 0

#### Scenario: Exit code 2 on validation failure

- **WHEN** `opm mod vet .` fails due to CUE errors
- **THEN** the exit code SHALL be 2
