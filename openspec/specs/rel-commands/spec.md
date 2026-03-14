## Purpose

Defines the `opm release` command group — a top-level CLI surface for release lifecycle management. The group handles both `#ModuleRelease` and `#BundleRelease` release files (this change implements `ModuleRelease` only). It provides two classes of commands: render commands (operate on a `.cue` release file) and cluster-query commands (operate on a deployed release by name or UUID).

## Requirements

### Requirement: release command group exists as a top-level command

The CLI SHALL expose `opm release` (alias: `rel`) as a top-level command group registered in `internal/cmd/root.go`. The group SHALL contain subcommands for release lifecycle management. Running `opm release` without a subcommand SHALL display help.

#### Scenario: release group registered at root level

- **WHEN** `opm release` is run without a subcommand
- **THEN** the CLI SHALL display help listing all `release` subcommands
- **AND** the exit code SHALL be 0

#### Scenario: rel alias resolves to release group

- **WHEN** `opm rel` is run without a subcommand
- **THEN** the CLI SHALL display the same help as `opm release`
- **AND** the exit code SHALL be 0

#### Scenario: release group implemented in dedicated package

- **WHEN** the `internal/cmd/release/` directory is inspected
- **THEN** it SHALL contain `release.go` with `NewReleaseCmd(*config.GlobalConfig)` constructor
- **AND** the cobra command SHALL use `Use: "release"` and `Aliases: []string{"rel"}`
- **AND** it SHALL contain one file per subcommand

### Requirement: release command group is polymorphic

The `opm release` command group is designed to handle both `#ModuleRelease` and `#BundleRelease` release files. This change implements `ModuleRelease` only. When a `#BundleRelease` file is detected, the CLI SHALL return a clear error rather than silently misbehaving.

#### Scenario: ModuleRelease file is accepted

- **WHEN** a render command is run with a `.cue` file containing `kind: "ModuleRelease"`
- **THEN** the CLI SHALL proceed with the standard `ModuleRelease` render pipeline

#### Scenario: BundleRelease file is rejected with clear error

- **WHEN** a render command is run with a `.cue` file containing `kind: "BundleRelease"`
- **THEN** the CLI SHALL exit with code 1
- **AND** display the error: `"bundle releases are not yet supported — use a #ModuleRelease file"`

#### Scenario: unknown release type is rejected

- **WHEN** a render command is run with a `.cue` file that does not contain a recognised `kind` field
- **THEN** the CLI SHALL exit with code 1 and display an error describing the unrecognised type

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

### Requirement: release cluster-query commands accept a release identifier as positional argument

The cluster-query commands (`status`, `tree`, `events`, `delete`) under `opm release` SHALL accept a release identifier as a required positional argument. The CLI SHALL auto-detect whether the identifier is a UUID or a release name.

#### Scenario: status by release name

- **WHEN** `opm release status jellyfin` is run
- **THEN** the CLI SHALL look up the release by name via label scan on inventory Secrets
- **AND** display the release health status

#### Scenario: status by UUID

- **WHEN** `opm release status 550e8400-e29b-41d4-a716-446655440000` is run
- **THEN** the CLI SHALL look up the release by UUID via direct Secret GET
- **AND** display the release health status

#### Scenario: delete by release name

- **WHEN** `opm release delete jellyfin` is run
- **THEN** the CLI SHALL look up the release by name and delete its resources from the cluster

#### Scenario: identifier auto-detection

- **WHEN** the positional argument matches the pattern `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`
- **THEN** the CLI SHALL treat it as a UUID
- **WHEN** the positional argument does not match the UUID pattern
- **THEN** the CLI SHALL treat it as a release name

### Requirement: release list command lists deployed releases

The `opm release list` command SHALL list all deployed releases in the target namespace. It SHALL accept no positional argument.

#### Scenario: list releases in namespace

- **WHEN** `opm release list -n production` is run
- **THEN** the CLI SHALL list all inventory Secrets in the `production` namespace
- **AND** display release names, UUIDs, and status

#### Scenario: list with no releases

- **WHEN** `opm release list` is run in a namespace with no deployed releases
- **THEN** the CLI SHALL display an empty list message
- **AND** exit with code 0

### Requirement: release cluster-query commands accept -n/--namespace flag

All cluster-query commands (`status`, `tree`, `events`, `delete`, `list`) SHALL accept `-n`/`--namespace` flag to specify the target namespace. This replaces the previous `ReleaseSelectorFlags` pattern.

#### Scenario: namespace flag with release name

- **WHEN** `opm release status jellyfin -n media` is run
- **THEN** the CLI SHALL look up the release named `jellyfin` in the `media` namespace

### Requirement: release render commands accept --provider flag

Render commands (`vet`, `build`, `apply`, `diff`) SHALL accept `--provider` flag for provider selection, consistent with the existing render pipeline. Provider SHALL NOT be specified in the release file.

#### Scenario: provider flag overrides config default

- **WHEN** `opm release build release.cue --provider kubernetes` is run
- **THEN** the render pipeline SHALL use the `kubernetes` provider regardless of config defaults

### Requirement: release cluster-connectivity commands accept K8s flags

The commands that connect to the cluster (`apply`, `diff`, `delete`, `status`, `tree`, `events`, `list`) SHALL accept `--kubeconfig` and `--context` flags.

#### Scenario: kubeconfig flag

- **WHEN** `opm release apply release.cue --kubeconfig ~/.kube/prod-config` is run
- **THEN** the CLI SHALL use the specified kubeconfig file for cluster connectivity
