## ADDED Requirements

<!-- Renamed from `rel-commands` (enhancement 0002 D6/D10). Spec dir is git mv'd at archive. The `BundleRelease file is rejected with clear error` scenario from the former `release command group is polymorphic` requirement is dropped here — X2 removed bundle support and its message (handoff recorded in X2 D-X2.3). -->

### Requirement: instance command group exists as a top-level command

The CLI SHALL expose `opm instance` (alias: `inst`) as a top-level command group registered in `internal/cmd/root.go`. The group SHALL contain subcommands for instance lifecycle management. Running `opm instance` without a subcommand SHALL display help. The former `release`/`rel` verb and alias are removed (no back-compat alias — enhancement 0002 D8).

#### Scenario: instance group registered at root level

- **WHEN** `opm instance` is run without a subcommand
- **THEN** the CLI SHALL display help listing all `instance` subcommands
- **AND** the exit code SHALL be 0

#### Scenario: inst alias resolves to instance group

- **WHEN** `opm inst` is run without a subcommand
- **THEN** the CLI SHALL display the same help as `opm instance`
- **AND** the exit code SHALL be 0

#### Scenario: removed release verb is not recognised

- **WHEN** `opm release` or `opm rel` is run
- **THEN** the CLI SHALL exit non-zero with an unknown-command error
- **AND** SHALL NOT display the instance group help

#### Scenario: instance group implemented in dedicated package

- **WHEN** the `internal/cmd/instance/` directory is inspected
- **THEN** it SHALL contain `instance.go` with `NewInstanceCmd(*config.GlobalConfig)` constructor
- **AND** the cobra command SHALL use `Use: "instance"` and `Aliases: []string{"inst"}`
- **AND** it SHALL contain one file per subcommand

### Requirement: instance command group accepts ModuleInstance files

The `opm instance` command group operates on `#ModuleInstance` instance files. When an instance file declares an unrecognised `kind`, the CLI SHALL return a clear error rather than silently misbehaving.

#### Scenario: ModuleInstance file is accepted

- **WHEN** a render command is run with a `.cue` file containing `kind: "ModuleInstance"`
- **THEN** the CLI SHALL proceed with the standard `ModuleInstance` render pipeline

#### Scenario: unknown instance kind is rejected

- **WHEN** a render command is run with a `.cue` file whose `kind` field is not a recognised instance kind
- **THEN** the CLI SHALL exit with code 1 and display an error describing the unrecognised kind (e.g. `unknown instance kind: "<kind>"`)

### Requirement: instance render commands accept an instance file as positional argument

The render commands (`vet`, `build`, `apply`, `diff`) under `opm instance` SHALL accept a `.cue` file path as a required positional argument. This file SHALL contain a `#ModuleInstance` definition.

#### Scenario: instance build with instance file

- **WHEN** `opm instance build jellyfin_instance.cue` is run
- **THEN** the CLI SHALL load the instance file, render it through the pipeline, and output manifests

#### Scenario: instance build without file argument

- **WHEN** `opm instance build` is run without a positional argument
- **THEN** the CLI SHALL exit with code 1 and display a usage error

#### Scenario: instance apply with instance file

- **WHEN** `opm instance apply jellyfin_instance.cue` is run
- **THEN** the CLI SHALL load the instance file, render it, and apply resources to the cluster via SSA

#### Scenario: instance vet with instance file

- **WHEN** `opm instance vet jellyfin_instance.cue` is run
- **THEN** the CLI SHALL load and validate the instance file through the render pipeline
- **AND** output per-resource validation results
- **AND** exit with code 0 on success or code 2 on validation failure

#### Scenario: instance diff with instance file

- **WHEN** `opm instance diff jellyfin_instance.cue` is run
- **THEN** the CLI SHALL render the instance and compare against live cluster state

#### Scenario: instance build without #module imported

- **WHEN** `opm instance build instance.cue` is run
- **AND** the instance file does not import or fill `#module`
- **THEN** the CLI SHALL exit with code 1
- **AND** display an error indicating `#module` is not filled and the user must import a module

### Requirement: instance cluster-query commands accept an instance identifier as positional argument

The cluster-query commands (`status`, `tree`, `events`, `delete`) under `opm instance` SHALL accept an instance identifier as a required positional argument. The CLI SHALL auto-detect whether the identifier is a UUID or an instance name.

#### Scenario: status by instance name

- **WHEN** `opm instance status jellyfin` is run
- **THEN** the CLI SHALL look up the instance by name via label scan on inventory Secrets
- **AND** display the instance health status

#### Scenario: status by UUID

- **WHEN** `opm instance status 550e8400-e29b-41d4-a716-446655440000` is run
- **THEN** the CLI SHALL look up the instance by UUID via direct Secret GET
- **AND** display the instance health status

#### Scenario: delete by instance name

- **WHEN** `opm instance delete jellyfin` is run
- **THEN** the CLI SHALL look up the instance by name and delete its resources from the cluster

#### Scenario: identifier auto-detection

- **WHEN** the positional argument matches the pattern `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`
- **THEN** the CLI SHALL treat it as a UUID
- **WHEN** the positional argument does not match the UUID pattern
- **THEN** the CLI SHALL treat it as an instance name

### Requirement: instance list command lists deployed instances

The `opm instance list` command SHALL list all deployed instances in the target namespace. It SHALL accept no positional argument.

#### Scenario: list instances in namespace

- **WHEN** `opm instance list -n production` is run
- **THEN** the CLI SHALL list all inventory Secrets in the `production` namespace
- **AND** display instance names, UUIDs, and status

#### Scenario: list with no instances

- **WHEN** `opm instance list` is run in a namespace with no deployed instances
- **THEN** the CLI SHALL display an empty list message
- **AND** exit with code 0

### Requirement: instance cluster-query commands accept -n/--namespace flag

All cluster-query commands (`status`, `tree`, `events`, `delete`, `list`) SHALL accept `-n`/`--namespace` flag to specify the target namespace.

#### Scenario: namespace flag with instance name

- **WHEN** `opm instance status jellyfin -n media` is run
- **THEN** the CLI SHALL look up the instance named `jellyfin` in the `media` namespace

### Requirement: instance render commands accept --provider flag

Render commands (`vet`, `build`, `apply`, `diff`) SHALL accept `--provider` flag for provider selection, consistent with the existing render pipeline. Provider SHALL NOT be specified in the instance file.

#### Scenario: provider flag overrides config default

- **WHEN** `opm instance build instance.cue --provider kubernetes` is run
- **THEN** the render pipeline SHALL use the `kubernetes` provider regardless of config defaults

### Requirement: instance cluster-connectivity commands accept K8s flags

The commands that connect to the cluster (`apply`, `diff`, `delete`, `status`, `tree`, `events`, `list`) SHALL accept `--kubeconfig` and `--context` flags.

#### Scenario: kubeconfig flag

- **WHEN** `opm instance apply instance.cue --kubeconfig ~/.kube/prod-config` is run
- **THEN** the CLI SHALL use the specified kubeconfig file for cluster connectivity
