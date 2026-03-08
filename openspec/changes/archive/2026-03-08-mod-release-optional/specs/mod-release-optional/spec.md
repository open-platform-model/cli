## ADDED Requirements

### Requirement: mod build and mod apply work without a release.cue file
`opm mod build` and `opm mod apply` SHALL produce correct output when run against a module directory that contains no `release.cue` file, provided that either `debugValues` is defined in the module or an explicit `-f` values file is given.

The commands SHALL synthesize a `ModuleRelease` in memory by filling `#config` with the provided values and extracting `#components` from the resulting module value. The full render pipeline (matching, transformer execution, resource generation) SHALL execute identically to the `release.cue`-backed path.

#### Scenario: build succeeds on bare module with debugValues
- **WHEN** `opm mod build .` is run in a module directory with no `release.cue`
- **AND** the module defines a concrete `debugValues` field
- **THEN** the command SHALL render and output Kubernetes manifests
- **AND** the exit code SHALL be 0

#### Scenario: apply succeeds on bare module with debugValues
- **WHEN** `opm mod apply .` is run in a module directory with no `release.cue`
- **AND** the module defines a concrete `debugValues` field
- **THEN** the command SHALL render resources and attempt to apply them to the cluster
- **AND** the exit code SHALL be 0 on successful apply

#### Scenario: build with -f on bare module
- **WHEN** `opm mod build . -f my-values.cue` is run in a module directory with no `release.cue`
- **THEN** the command SHALL use the provided values file instead of `debugValues`
- **AND** output manifests SHALL reflect the values from the `-f` file

### Requirement: mod build and mod apply default to debugValues when no -f is given
When no `-f` / `--values` flag is provided, `opm mod build` and `opm mod apply` SHALL use the module's `debugValues` field as the values source. This applies regardless of whether `release.cue` exists.

When `release.cue` is present and no `-f` is given, the `debugValues` SHALL be filled into the release package (existing `DebugValues` path). When `release.cue` is absent, the `debugValues` SHALL be used for the synthesis path.

#### Scenario: build with release.cue present and no -f uses debugValues
- **WHEN** `opm mod build .` is run in a module directory that has a `release.cue`
- **AND** no `-f` flag is provided
- **THEN** the command SHALL extract `debugValues` from the module and use them as the values source
- **AND** the output manifests SHALL reflect the `debugValues` values

#### Scenario: build with release.cue present and -f uses provided values
- **WHEN** `opm mod build . -f prod-values.cue` is run
- **AND** a `release.cue` is present
- **THEN** the command SHALL use `prod-values.cue` and NOT `debugValues`

#### Scenario: apply with no release.cue and no -f uses debugValues
- **WHEN** `opm mod apply .` is run in a module directory with no `release.cue`
- **AND** no `-f` flag is provided
- **THEN** the command SHALL use `debugValues` from the module

### Requirement: Synthesis mode uses module defaultNamespace as the release namespace
When no `release.cue` exists, the synthesized `ModuleRelease` SHALL use `metadata.defaultNamespace` from the module as the release namespace, unless the namespace was explicitly set via `-n` / `--namespace` flag or `OPM_NAMESPACE` environment variable.

#### Scenario: Synthesized release uses module defaultNamespace
- **WHEN** `opm mod build .` is run with no `release.cue` and no `-n` flag
- **AND** the module defines `metadata.defaultNamespace: "jellyfin"`
- **THEN** the rendered resources SHALL target the `jellyfin` namespace

#### Scenario: -n flag overrides module defaultNamespace in synthesis mode
- **WHEN** `opm mod build . -n staging` is run with no `release.cue`
- **THEN** the rendered resources SHALL target the `staging` namespace

### Requirement: Synthesis mode derives the release name from module metadata
When no `release.cue` exists, the synthesized `ModuleRelease` SHALL use `metadata.name` from the module as the release name, unless overridden by `--name` flag.

#### Scenario: Synthesized release uses module metadata name
- **WHEN** `opm mod build .` is run with no `release.cue`
- **AND** the module defines `metadata.name: "jellyfin"`
- **THEN** the release name in log output SHALL be `jellyfin`

### Requirement: Clear error when no release.cue, no debugValues, and no -f
When `release.cue` is absent AND no `-f` flag is given AND the module does not define `debugValues`, `opm mod build` and `opm mod apply` SHALL fail with a descriptive error message that explains what the user must do.

#### Scenario: Error when no values source available
- **WHEN** `opm mod build .` is run in a module directory with no `release.cue`
- **AND** the module has no `debugValues` field
- **AND** no `-f` flag is provided
- **THEN** the command SHALL fail with exit code non-zero
- **AND** the error message SHALL mention both options: adding `debugValues` to the module OR using `-f <values-file>`
