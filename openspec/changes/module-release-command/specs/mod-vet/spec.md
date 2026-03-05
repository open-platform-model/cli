## MODIFIED Requirements

### Requirement: mod vet command validates module without generating manifests

The `opm mod vet` command SHALL build the module via the render pipeline and report per-resource validation results. It SHALL NOT output manifests (YAML/JSON). Its purpose is pass/fail validation with clear per-resource feedback.

The command SHALL accept a module path argument (default: current directory) and a subset of `mod build` flags for values, namespace, name, and provider resolution.

When no `-f`/`--values` flag is provided, `opm mod vet` SHALL use the module's `debugValues` field as the values source. If `-f` is provided, it SHALL override `debugValues` and use the specified values files instead. If neither `debugValues` nor `-f` exists, the command SHALL return an error indicating that values must be provided via `debugValues` in the module or via `-f`.

#### Scenario: Valid module passes validation using debugValues

- **WHEN** `opm mod vet .` is run on a module that defines `debugValues`
- **AND** no `-f` flag is provided
- **THEN** the command SHALL use `debugValues` as the values source for the render pipeline
- **AND** each generated resource SHALL be printed as a `FormatResourceLine` with `"valid"` status
- **AND** a final summary line SHALL be printed: `FormatCheckmark("Module valid (<N> resources)")`
- **AND** the command SHALL exit with code 0

#### Scenario: -f flag overrides debugValues

- **WHEN** `opm mod vet . -f prod-values.cue` is run on a module that defines `debugValues`
- **THEN** the command SHALL use `prod-values.cue` as the values source
- **AND** `debugValues` SHALL be ignored

#### Scenario: No debugValues and no -f flag

- **WHEN** `opm mod vet .` is run on a module that does not define `debugValues` (or `debugValues` is `_`)
- **AND** no `-f` flag is provided
- **THEN** the command SHALL return an error indicating values must be provided
- **AND** the exit code SHALL be 2

#### Scenario: Module with CUE validation errors fails with details

- **WHEN** `opm mod vet .` is run on a module with `debugValues` that do not satisfy `#config`
- **THEN** the command SHALL print the validation error using `printValidationError`
- **AND** the command SHALL exit with code 2

#### Scenario: Module with render errors fails with component details

- **WHEN** `opm mod vet .` is run on a module with unmatched components
- **THEN** the command SHALL print render errors using `printRenderErrors`
- **AND** the command SHALL exit with code 2

#### Scenario: Module with zero resources after successful render

- **WHEN** `opm mod vet .` is run on a module that renders zero resources
- **THEN** the command SHALL print: `FormatCheckmark("Module valid (0 resources)")`
- **AND** the command SHALL exit with code 0
