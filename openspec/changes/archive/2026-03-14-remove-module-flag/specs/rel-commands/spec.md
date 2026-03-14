## REMOVED Requirements

### Requirement: release render commands accept --module flag for local module injection

**Reason**: The `--module` flag created the most complex mutation path in the release pipeline (4 field mutations on a partially-constructed `*module.Release`). Removing it enforces a single module resolution path (CUE imports) and simplifies the render pipeline. The CLI is pre-1.0 so there is no backward-compatibility obligation.

**Migration**: Use CUE imports in the release file to fill `#module`. Set up a local CUE registry or use relative import paths in `cue.mod/module.cue` for local development.

## MODIFIED Requirements

### Requirement: release render commands accept a release file as positional argument

The render commands (`vet`, `build`, `apply`, `diff`) under `opm release` SHALL accept a `.cue` file path as a required positional argument. This file SHALL contain a `#ModuleRelease` definition.

#### Scenario: release build with release file

- **WHEN** `opm release build jellyfin_release.cue` is run
- **THEN** the CLI SHALL load the release file, render it through the pipeline, and output manifests

#### Scenario: release build without file argument

- **WHEN** `opm release build` is run without a positional argument
- **THEN** the CLI SHALL exit with code 1 and display a usage error

#### Scenario: release apply with release file

- **WHEN** `opm release apply jellyfin_release.cue` is run
- **THEN** the CLI SHALL load the release file, render it, and apply resources to the cluster via SSA

#### Scenario: release vet with release file

- **WHEN** `opm release vet jellyfin_release.cue` is run
- **THEN** the CLI SHALL load and validate the release file through the render pipeline
- **AND** output per-resource validation results
- **AND** exit with code 0 on success or code 2 on validation failure

#### Scenario: release diff with release file

- **WHEN** `opm release diff jellyfin_release.cue` is run
- **THEN** the CLI SHALL render the release and compare against live cluster state

#### Scenario: release build without #module imported

- **WHEN** `opm release build release.cue` is run
- **AND** the release file does not import or fill `#module`
- **THEN** the CLI SHALL exit with code 1
- **AND** display an error indicating `#module` is not filled and the user must import a module
