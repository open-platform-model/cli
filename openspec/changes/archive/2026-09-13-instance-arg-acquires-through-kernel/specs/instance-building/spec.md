## MODIFIED Requirements

### Requirement: Value selection falls back to module defaults when no files are given

When no `--values` files are provided, the builder SHALL discover values using the following priority:

1. When `instance.cue` is present: auto-discover `values.cue` from the module directory (existing behavior)
2. When `instance.cue` is absent and `debugValues` is defined in the module: use `debugValues` as the values source
3. When neither `instance.cue` nor `values.cue` nor `debugValues` is available: return a descriptive error

The builder SHALL NOT read values from `Module.Values`. If `--values` files are provided, `values.cue` and `debugValues` SHALL both be ignored.

When an instance package is acquired from its directory (`AcquireInstanceFromDir`, the instance-file path), the `values` field is inline in the instance CUE file itself. There is no `values.cue` fallback — the instance package is self-contained.

#### Scenario: No values file, `values.cue` exists in module directory

- **WHEN** `AcquireInstanceFromDir` acquires an instance package with no trailing values source
- **AND** `values.cue` exists in the module directory
- **THEN** `values.cue` is loaded alongside `instance.cue` as part of the CUE instance

#### Scenario: Instance file is self-contained

- **WHEN** `AcquireInstanceFromDir` acquires the package holding an instance file
- **THEN** the `values` field is read from the instance CUE file's inline definition
- **AND** no `values.cue` file is searched for or loaded

#### Scenario: No values files, no values.cue, debugValues defined

- **WHEN** no `--values` files are provided
- **AND** no `values.cue` file exists in the module directory
- **AND** the module defines a concrete `debugValues` field
- **THEN** the builder SHALL use `debugValues` as the values source

#### Scenario: No values files, no values.cue, no debugValues

- **WHEN** no `--values` files are provided
- **AND** no `values.cue` file exists in the module directory
- **AND** the module has no `debugValues` field
- **THEN** the builder SHALL return an error indicating the user must provide values via `values.cue`, `debugValues`, or `--values`

#### Scenario: Multiple `--values` files are unified

- **WHEN** more than one values file is provided via `--values`
- **THEN** the builder SHALL unify all files together before injection
