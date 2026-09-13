# Capability: validation-output

## Purpose

Validation failures from `opm mod vet` and `opm instance vet` reach the user as one grouped block per error class, with `file:line:col` positions attached, instead of one repeated `ERRO` line per underlying CUE error. The e2e suite in `tests/e2e/vet_output_test.go` holds that grouping in place.

## Requirements

### Requirement: E2E Tests for Validation Grouping

The CLI test suite SHALL verify that validation output during module/instance vetting preserves grouped formatting rather than emitting flattened error lines, and SHALL pin the vet verdicts the kernel's values validation produces.

#### Scenario: Vetting an invalid instance

- **WHEN** a user runs `opm instance vet` with conflicting or not-allowed values
- **THEN** the CLI output groups errors by error type and prints associated file line paths cleanly without duplicating `ERRO render failed` for every line.

#### Scenario: Vetting an invalid module directly

- **WHEN** a user runs `opm mod vet` with conflicting or not-allowed values
- **THEN** the CLI output groups errors cleanly without duplicating `ERRO values do not satisfy #config` for every line.

#### Scenario: Open debugValues refused at the schema position

- **WHEN** a user runs `opm mod vet` on a module whose `debugValues` is `_` and whose `#config` has a field without a default
- **THEN** the output SHALL be one grouped block under `values do not satisfy #config` naming the incomplete field at its `#config` position
- **AND** no `not concrete` line SHALL appear
- **AND** the exit code SHALL be 2

#### Scenario: Values files against a module without #config

- **WHEN** a user runs `opm mod vet -f values.cue` on a module that declares no `#config`
- **THEN** the command SHALL refuse with `module does not define #config; values files cannot be validated` and exit 2
- **AND** the output SHALL NOT contain `Values satisfy #config`
