## MODIFIED Requirements

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
