# Capability: cmdutil

## Purpose

The `internal/cmdutil` package provides shared command utilities for `mod` subcommands. It centralizes flag group management, render pipeline orchestration, Kubernetes client creation, and output formatting helpers so that each command file contains only its unique logic.

## Requirements

### Requirement: RenderFlags struct registers a consistent set of render-related flags

The `RenderFlags` struct SHALL provide an `AddTo(*cobra.Command)` method that registers the flags `--values`/`-f` (string array, repeatable), `--namespace`/`-n` (string), `--release-name` (string), and `--provider` (string) on the given cobra command. Flag names, short aliases, and default values SHALL be identical to the current per-command registrations.

#### Scenario: RenderFlags registers all four flags on a cobra command

- **WHEN** `RenderFlags.AddTo(cmd)` is called on a new cobra command
- **THEN** the command SHALL have a `--values` flag (short: `-f`) of type `StringArray` with default `nil`
- **AND** the command SHALL have a `--namespace` flag (short: `-n`) of type `String` with default `""`
- **AND** the command SHALL have a `--release-name` flag of type `String` with default `""`
- **AND** the command SHALL have a `--provider` flag of type `String` with default `""`

#### Scenario: RenderFlags values are accessible after flag parsing

- **WHEN** a command using `RenderFlags` is invoked with `--values a.cue -f b.cue -n production --release-name my-app --provider kubernetes`
- **THEN** `RenderFlags.Values` SHALL equal `["a.cue", "b.cue"]`
- **AND** `RenderFlags.Namespace` SHALL equal `"production"`
- **AND** `RenderFlags.ReleaseName` SHALL equal `"my-app"`
- **AND** `RenderFlags.Provider` SHALL equal `"kubernetes"`

### Requirement: K8sFlags struct registers Kubernetes connection flags

The `K8sFlags` struct SHALL provide an `AddTo(*cobra.Command)` method that registers `--kubeconfig` (string) and `--context` (string) on the given cobra command.

#### Scenario: K8sFlags registers both flags on a cobra command

- **WHEN** `K8sFlags.AddTo(cmd)` is called on a new cobra command
- **THEN** the command SHALL have a `--kubeconfig` flag of type `String` with default `""`
- **AND** the command SHALL have a `--context` flag of type `String` with default `""`

#### Scenario: K8sFlags values are accessible after flag parsing

- **WHEN** a command using `K8sFlags` is invoked with `--kubeconfig /path/to/config --context staging`
- **THEN** `K8sFlags.Kubeconfig` SHALL equal `"/path/to/config"`
- **AND** `K8sFlags.Context` SHALL equal `"staging"`

### Requirement: InstanceSelectorFlags struct registers and validates instance identification flags

The `InstanceSelectorFlags` struct SHALL provide an `AddTo(*cobra.Command)` method that registers `--instance-name` (string), `--instance-id` (string), and `--namespace`/`-n` (string). It SHALL also provide a `Validate()` method that enforces mutual exclusivity between `--instance-name` and `--instance-id`, and requires exactly one to be present. <!-- Was: ReleaseSelectorFlags, --release-name/--release-id (0002 D10/D-X4.2) -->

#### Scenario: Both selectors set is rejected

- **WHEN** `InstanceSelectorFlags.Validate()` is called with both `InstanceName` and `InstanceID` set to non-empty strings
- **THEN** it SHALL return an error with message containing `"--instance-name and --instance-id are mutually exclusive"`

#### Scenario: Neither selector set is rejected

- **WHEN** `InstanceSelectorFlags.Validate()` is called with both `InstanceName` and `InstanceID` set to empty strings
- **THEN** it SHALL return an error with message containing `"either --instance-name or --instance-id is required"`

#### Scenario: Exactly one selector set is accepted

- **WHEN** `InstanceSelectorFlags.Validate()` is called with `InstanceName` set to `"my-app"` and `InstanceID` set to `""`
- **THEN** it SHALL return no error

### Requirement: InstanceSelectorFlags provides a LogName helper

The `InstanceSelectorFlags` struct SHALL provide a `LogName()` method that returns the instance name if set, or a truncated instance ID prefix (first 8 characters) formatted as `"instance:<prefix>"` otherwise. This is used for scoped logger creation. <!-- Was: ReleaseSelectorFlags.LogName, "release:<prefix>" (0002 D10) -->

#### Scenario: LogName prefers instance name

- **WHEN** `LogName()` is called with `InstanceName` set to `"my-app"`
- **THEN** it SHALL return `"my-app"`

#### Scenario: LogName falls back to truncated instance ID

- **WHEN** `LogName()` is called with `InstanceName` set to `""` and `InstanceID` set to `"a1b2c3d4-e5f6-7890-abcd"`
- **THEN** it SHALL return `"instance:a1b2c3d4"`

### Requirement: RenderRelease orchestration

`cmdutil.RenderRelease()` SHALL use the release-file loading path exclusively. There is no synthesis branch.

**When `release.cue` is present** (unchanged):
- Call `loader.LoadReleasePackage()`, `loader.DetectReleaseKind()`, `loader.LoadModuleReleaseFromValue()` (or bundle equivalent), then `engine.ModuleRenderer.Render()`.

In all cases, resources are converted to `[]*unstructured.Unstructured` before passing to downstream packages. No CUE types cross this boundary.

#### Scenario: CUE boundary enforcement (unchanged)
- **WHEN** `RenderRelease()` passes resources to `internal/kubernetes/` or `internal/inventory/`
- **THEN** resources are `[]*unstructured.Unstructured` — no CUE types cross this boundary

### Requirement: Values file resolution stays in cmdutil

Values file resolution SHALL remain in `internal/cmdutil/` as a CLI-layer concern. With the synthesis path removed, the resolution simplifies:

- When `--values` files are provided: pass them to `LoadReleasePackage`.
- When no `--values` files are provided: pass empty string to `LoadReleasePackage()`, which defaults to `values.cue` in the release directory (existing behavior).

#### Scenario: Values flag resolution (unchanged)
- **WHEN** the user provides `--values custom-values.cue`
- **THEN** cmdutil resolves the path and passes it to `LoadReleasePackage`

#### Scenario: Default values fallback with release.cue present
- **WHEN** no `--values` flag is provided
- **AND** `release.cue` is present
- **THEN** cmdutil passes empty string to `LoadReleasePackage()`, which defaults to `values.cue` in the release directory

### Requirement: ShowRenderOutput checks for errors, shows transformer matches, and logs warnings

The `ShowRenderOutput` function SHALL accept a render result and output options (verbose flag). It SHALL:

1. Check for render errors — if present, format and print render errors via `PrintRenderErrors`, then return an `*ExitError` with `ExitValidationError`.
2. Show transformer match output — verbose mode shows module metadata and match reasons; default mode shows compact match lines.
3. Log any warnings from the render result via the module-scoped logger.

#### Scenario: ShowRenderOutput returns error when render has errors

- **WHEN** `ShowRenderOutput` is called with a result containing errors
- **THEN** it SHALL call `PrintRenderErrors` with the errors
- **AND** it SHALL return an `*ExitError` with `Code` equal to `ExitValidationError`

#### Scenario: ShowRenderOutput shows compact matches in default mode

- **WHEN** `ShowRenderOutput` is called with verbose set to `false` and a result containing transformer matches
- **THEN** it SHALL print one line per match using `FormatTransformerMatch`
- **AND** it SHALL print a warning for each unmatched component using `FormatTransformerUnmatched`

#### Scenario: ShowRenderOutput shows detailed matches in verbose mode

- **WHEN** `ShowRenderOutput` is called with verbose set to `true`
- **THEN** it SHALL print module metadata (name, namespace, version, components)
- **AND** it SHALL print match details including the match reason
- **AND** it SHALL print per-resource validation lines

#### Scenario: ShowRenderOutput logs warnings

- **WHEN** `ShowRenderOutput` is called with a result that has warnings
- **THEN** each warning SHALL be logged via the module-scoped logger at warn level

### Requirement: NewK8sClient creates a Kubernetes client or returns a connectivity ExitError

The `NewK8sClient` function SHALL accept a `*config.ResolvedKubernetesConfig` and an API warnings string. It SHALL create a `kubernetes.Client` using `kubernetes.NewClient`. On failure, it SHALL return an `*ExitError` with `Code` equal to `ExitConnectivityError`.

#### Scenario: NewK8sClient succeeds with valid kubeconfig

- **WHEN** `NewK8sClient` is called with a valid kubeconfig and context
- **THEN** it SHALL return a non-nil `*kubernetes.Client`
- **AND** the error SHALL be `nil`

#### Scenario: NewK8sClient returns connectivity error on failure

- **WHEN** `NewK8sClient` is called and `kubernetes.NewClient` returns an error
- **THEN** it SHALL return an `*ExitError` with `Code` equal to `ExitConnectivityError`

### Requirement: cmdutil package does not import the cmd package

The `internal/cmdutil` package SHALL NOT import `internal/cmd`. All dependencies SHALL flow in one direction: `cmd` imports `cmdutil`. The `cmdutil` functions SHALL accept their dependencies (OPMConfig, registry, verbose flag) as explicit parameters rather than accessing package-level state.

#### Scenario: Dependency direction is enforced

- **WHEN** the Go toolchain compiles the project
- **THEN** `internal/cmdutil` SHALL have no import path that includes `internal/cmd`
- **AND** `internal/cmd` MAY import `internal/cmdutil`

### Requirement: Flag groups compose without conflict on a single cobra command

Multiple flag group structs SHALL be usable on the same cobra command without flag name collisions. When a command uses both `RenderFlags` and `K8sFlags`, all flags from both groups SHALL be registered and independently accessible.

#### Scenario: RenderFlags and K8sFlags compose on one command

- **WHEN** both `RenderFlags.AddTo(cmd)` and `K8sFlags.AddTo(cmd)` are called on the same command
- **THEN** the command SHALL have all 6 flags registered (values, namespace, release-name, provider, kubeconfig, context)
- **AND** each flag SHALL be independently settable without conflict

### Requirement: PrintValidationError formats render validation errors consistently

The `PrintValidationError` function SHALL accept a message string and an error. When the error is a `*errors.ConfigError` with field errors, it SHALL print a summary line followed by the CUE error details. For other errors, it SHALL use the standard key-value log format.

#### Scenario: ConfigError with CUE details

- **WHEN** `PrintValidationError` is called with a `*errors.ConfigError` that has field errors
- **THEN** the output SHALL include the summary message
- **AND** the output SHALL include the CUE details as plain text on stderr

#### Scenario: Generic error without CUE details

- **WHEN** `PrintValidationError` is called with a plain `error`
- **THEN** the output SHALL use the standard error log format with the message and error key-value pair

### Requirement: PrintRenderErrors formats render errors with diagnostic detail

The `PrintRenderErrors` function SHALL accept a slice of errors and print each one with appropriate diagnostic information. For `*engine.UnmatchedComponentsError`, it SHALL print the component names and per-transformer diagnostics (missing labels, resources, traits). For transform errors, it SHALL print the component name, transformer FQN, and cause.

#### Scenario: Unmatched components error with diagnostics

- **WHEN** `PrintRenderErrors` is called with an `UnmatchedComponentsError`
- **THEN** the output SHALL include the unmatched component names
- **AND** the output SHALL list per-transformer diagnostics (missing labels, resources, traits)

#### Scenario: Transform error

- **WHEN** `PrintRenderErrors` is called with a transform error
- **THEN** the output SHALL include the component name, transformer FQN, and the cause error

### Requirement: Refactored mod commands preserve exact behavioral equivalence

The `mod` subcommands that consume `InstanceSelectorFlags` SHALL preserve the same observable behavior after the rename: identical resolution, output, and exit codes for equivalent inputs, with the flag names updated to `--instance-name`/`--instance-id`. <!-- Was: --release-name/--release-id (0002 D-X4.2) -->

#### Scenario: mod delete behavior preserved under renamed flags

- **WHEN** `opm mod delete --instance-name my-app -n production` is run
- **THEN** it SHALL produce the same resolution and deletion behavior the pre-rename `--release-name` form produced

#### Scenario: mod status behavior preserved under renamed flags

- **WHEN** `opm mod status --instance-name my-app -n production` is run
- **THEN** it SHALL produce the same status output the pre-rename `--release-name` form produced

### Requirement: Shared inventory resolution helper in cmdutil

`cmdutil.ResolveInventory` SHALL resolve an inventory record from an `*InstanceSelectorFlags` (carrying instance name and/or instance ID). If `flags.InstanceID` is non-empty, it SHALL resolve via `inventory.GetInventory` using the instance ID; if `flags.InstanceName` is also set, that name SHALL be used as the display name. If only `flags.InstanceName` is non-empty, it SHALL resolve via `inventory.FindInventoryByInstanceName`. When no inventory Secret is found, it SHALL return an `InstanceNotFoundError`. <!-- Was: *ReleaseSelectorFlags, ReleaseID/ReleaseName, ReleaseNotFoundError (0002 D10) -->

#### Scenario: Resolve by instance name

- **WHEN** `InstanceSelectorFlags.InstanceName` is set and the inventory Secret exists
- **THEN** `ResolveInventory` SHALL return the matching inventory record

#### Scenario: Resolve by instance ID

- **WHEN** `InstanceSelectorFlags.InstanceID` is set and the inventory Secret exists
- **THEN** `ResolveInventory` SHALL return the matching inventory record

#### Scenario: Instance not found

- **WHEN** the underlying inventory lookup returns no Secret
- **THEN** `ResolveInventory` SHALL return an `InstanceNotFoundError`
