## MODIFIED Requirements

### Requirement: mod build and mod apply work without an instance.cue file

`opm mod build` and `opm mod apply` SHALL produce correct output when run against a module directory that contains no `instance.cue` file, provided that either `debugValues` is defined in the module or an explicit `-f` values file is given.

#### Scenario: build succeeds on bare module with debugValues
- **WHEN** `opm mod build .` is run in a module directory with no `instance.cue`
- **AND** the module defines a concrete `debugValues` field
- **THEN** the command SHALL render and output Kubernetes manifests
- **AND** the exit code SHALL be 0

#### Scenario: apply succeeds on bare module with debugValues
- **WHEN** `opm mod apply .` is run in a module directory with no `instance.cue`
- **AND** the module defines a concrete `debugValues` field
- **THEN** the command SHALL render resources and attempt to apply them to the cluster
- **AND** the exit code SHALL be 0 on successful apply

#### Scenario: build with -f on bare module
- **WHEN** `opm mod build . -f my-values.cue` is run in a module directory with no `instance.cue`
- **THEN** the command SHALL use the provided values file instead of `debugValues`
- **AND** output manifests SHALL reflect the values from the `-f` file

## ADDED Requirements

### Requirement: The commands synthesize a ModuleInstance in memory when no instance.cue exists

`opm mod build` and `opm mod apply` SHALL synthesize a `ModuleInstance` in memory by filling `#config` with the provided values and extracting `#components` from the resulting module value. The full render pipeline (matching, transformer execution, resource generation) SHALL execute identically to the `instance.cue`-backed path.
(Requirement: mod build and mod apply work without an instance.cue file)

#### Scenario: Bare module renders through a synthesized ModuleInstance

- **WHEN** `opm mod build` runs in a module directory with no `instance.cue` and values are provided
- **THEN** the command SHALL synthesize a `ModuleInstance` in memory by filling `#config` with the provided values and extracting `#components` from the resulting module value
- **AND** the full render pipeline (matching, transformer execution, resource generation) SHALL execute identically to the `instance.cue`-backed path
