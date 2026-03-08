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

### Requirement: ReleaseSelectorFlags struct registers and validates release identification flags

The `ReleaseSelectorFlags` struct SHALL provide an `AddTo(*cobra.Command)` method that registers `--release-name` (string), `--release-id` (string), and `--namespace`/`-n` (string). It SHALL also provide a `Validate()` method that enforces mutual exclusivity between `--release-name` and `--release-id`, and requires exactly one to be present.

#### Scenario: Validate rejects both release-name and release-id

- **WHEN** `ReleaseSelectorFlags.Validate()` is called with both `ReleaseName` and `ReleaseID` set to non-empty strings
- **THEN** it SHALL return an error with message containing `"mutually exclusive"`

#### Scenario: Validate rejects neither release-name nor release-id

- **WHEN** `ReleaseSelectorFlags.Validate()` is called with both `ReleaseName` and `ReleaseID` set to empty strings
- **THEN** it SHALL return an error with message containing `"either --release-name or --release-id is required"`

#### Scenario: Validate accepts exactly one of release-name or release-id

- **WHEN** `ReleaseSelectorFlags.Validate()` is called with `ReleaseName` set to `"my-app"` and `ReleaseID` set to `""`
- **THEN** it SHALL return `nil`

### Requirement: ReleaseSelectorFlags provides a LogName helper

The `ReleaseSelectorFlags` struct SHALL provide a `LogName()` method that returns the release name if set, or a truncated release ID prefix (first 8 characters) formatted as `"release:<prefix>"` otherwise. This is used for scoped logger creation.

#### Scenario: LogName returns release name when set

- **WHEN** `LogName()` is called with `ReleaseName` set to `"my-app"`
- **THEN** it SHALL return `"my-app"`

#### Scenario: LogName returns truncated release ID when name is empty

- **WHEN** `LogName()` is called with `ReleaseName` set to `""` and `ReleaseID` set to `"a1b2c3d4-e5f6-7890-abcd"`
- **THEN** it SHALL return `"release:a1b2c3d4"`

### Requirement: RenderRelease orchestration
`cmdutil.RenderRelease()` SHALL orchestrate the new rendering pipeline: call `loader.LoadReleasePackage()`, `loader.DetectReleaseKind()`, `loader.LoadModuleReleaseFromValue()` (or bundle equivalent), then `engine.ModuleRenderer.Render()` (or `BundleRenderer.Render()`). It SHALL convert `[]*core.Resource` to `[]*unstructured.Unstructured` at this layer before passing to downstream packages.

#### Scenario: Module release rendering
- **WHEN** `RenderRelease()` is called for a ModuleRelease
- **THEN** it loads via `pkg/loader`, renders via `pkg/engine`, converts resources to Unstructured, and returns results compatible with `internal/kubernetes/` and `internal/inventory/`

#### Scenario: CUE boundary enforcement
- **WHEN** `RenderRelease()` passes resources to `internal/kubernetes/` or `internal/inventory/`
- **THEN** resources are `[]*unstructured.Unstructured` — no CUE types cross this boundary

### Requirement: Values file resolution stays in cmdutil
Values file resolution (--values CLI flags with fallback to values.cue) SHALL remain in `internal/cmdutil/` as a CLI-layer concern. The resolved file path is passed to `loader.LoadReleasePackage()`.

#### Scenario: Values flag resolution
- **WHEN** the user provides `--values custom-values.cue`
- **THEN** cmdutil resolves the path and passes it to `LoadReleasePackage(cueCtx, releaseFile, resolvedValuesFile)`

#### Scenario: Default values fallback
- **WHEN** no --values flag is provided
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

After refactoring to use `cmdutil`, each mod command SHALL produce identical output, exit codes, error messages, and flag behavior compared to the pre-refactoring implementation. No user-observable change SHALL occur.

#### Scenario: mod build output is identical after refactoring

- **WHEN** `opm mod build` is run with identical inputs before and after the refactoring
- **THEN** the stdout output (rendered manifests) SHALL be byte-identical
- **AND** the stderr output (logs, warnings, verbose) SHALL be semantically identical
- **AND** the exit code SHALL be identical

#### Scenario: mod apply error handling is identical after refactoring

- **WHEN** `opm mod apply` encounters a render error, connectivity error, or apply error before and after the refactoring
- **THEN** the error message format SHALL be identical
- **AND** the exit code SHALL be identical

#### Scenario: mod vet validation output is identical after refactoring

- **WHEN** `opm mod vet` is run on a valid module before and after the refactoring
- **THEN** the per-resource validation lines and summary SHALL be identical
- **AND** the exit code SHALL be 0 in both cases

#### Scenario: mod delete confirmation prompt is identical after refactoring

- **WHEN** `opm mod delete --release-name my-app -n production` is run before and after the refactoring
- **THEN** the confirmation prompt text SHALL be identical

#### Scenario: mod diff output is identical after refactoring

- **WHEN** `opm mod diff` is run with identical inputs and cluster state before and after the refactoring
- **THEN** the diff output (summary line, per-resource diffs) SHALL be identical

#### Scenario: mod status table output is identical after refactoring

- **WHEN** `opm mod status --release-name my-app -n production` is run before and after the refactoring
- **THEN** the table output format SHALL be identical

### Requirement: Shared inventory resolution helper in cmdutil

The `cmdutil` package SHALL provide a `ResolveInventory` function that encapsulates
the full inventory lookup-and-discover flow used by `mod delete` and `mod status`.

The function SHALL accept:
- A context
- A Kubernetes client
- A `*ReleaseSelectorFlags` (carrying release name and/or release ID)
- A namespace string
- A structured logger scoped to the release

The function SHALL return the resolved `*inventory.InventorySecret`, the discovered
live `[]*core.Resource`, and an error.

The function MUST implement the following resolution logic:
- If `rsf.ReleaseID` is non-empty: resolve via `inventory.GetInventory` using the
  release ID. If `rsf.ReleaseName` is also set, use it as the display name; otherwise
  use the release ID as the display name.
- If `rsf.ReleaseName` is non-empty (and no ReleaseID): resolve via
  `inventory.FindInventoryByReleaseName`.
- If inventory lookup fails: log the error and return an `*ExitError` with code
  `ExitGeneralError`.
- If the inventory Secret is not found: log the error and return an `*ExitError`
  with code `ExitNotFound`.
- After resolving the Secret: call `inventory.DiscoverResourcesFromInventory` to fetch
  live resources. If this fails: log the error and return an `*ExitError` with code
  `ExitGeneralError`.

#### Scenario: Resolution by release name succeeds

- **WHEN** `ReleaseSelectorFlags.ReleaseName` is set and the inventory Secret exists
- **THEN** `ResolveInventory` returns the Secret and its discovered live resources with no error

#### Scenario: Resolution by release ID succeeds

- **WHEN** `ReleaseSelectorFlags.ReleaseID` is set and the inventory Secret exists
- **THEN** `ResolveInventory` returns the Secret and its discovered live resources with no error

#### Scenario: Release not found

- **WHEN** the inventory Secret does not exist
- **THEN** `ResolveInventory` returns an `*ExitError` with code `ExitNotFound`

#### Scenario: Kubernetes error during inventory lookup

- **WHEN** `inventory.GetInventory` or `inventory.FindInventoryByReleaseName` returns
  a non-nil error
- **THEN** `ResolveInventory` logs the error and returns an `*ExitError` with code
  `ExitGeneralError`

#### Scenario: Resource discovery fails

- **WHEN** the inventory Secret is found but `DiscoverResourcesFromInventory` returns
  an error
- **THEN** `ResolveInventory` logs the error and returns an `*ExitError` with code
  `ExitGeneralError`
