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

### Requirement: RenderModule executes the common render pipeline and returns a result or ExitError

The `RenderModule` function SHALL accept a context, a `RenderModuleOpts` struct (containing args, RenderFlags, optional K8sFlags, OPMConfig, registry string, and verbose flag), and SHALL execute the full render pipeline preamble: resolve module path from args, validate OPM config is loaded, resolve Kubernetes config (namespace/provider) via `config.ResolveKubernetes`, build `build.RenderOptions`, validate options, create `build.NewPipeline`, and call `pipeline.Render()`.

On success, it SHALL return the `*build.RenderResult`. On failure at any step, it SHALL return an `*ExitError` with the appropriate exit code.

#### Scenario: RenderModule succeeds with valid module

- **WHEN** `RenderModule` is called with a valid module path, loaded OPMConfig, and valid RenderFlags
- **THEN** it SHALL return a non-nil `*build.RenderResult`
- **AND** the error SHALL be `nil`

#### Scenario: RenderModule fails when OPMConfig is nil

- **WHEN** `RenderModule` is called with a nil OPMConfig
- **THEN** it SHALL return an `*ExitError` with `Code` equal to `ExitGeneralError`
- **AND** the error message SHALL contain `"configuration not loaded"`

#### Scenario: RenderModule fails on render validation error

- **WHEN** `pipeline.Render()` returns a `*build.ReleaseValidationError`
- **THEN** `RenderModule` SHALL call the validation error formatter before returning
- **AND** it SHALL return an `*ExitError` with `Code` equal to `ExitValidationError` and `Printed` set to `true`

#### Scenario: RenderModule defaults module path to current directory

- **WHEN** `RenderModule` is called with an empty args slice
- **THEN** the module path used for `build.RenderOptions.ModulePath` SHALL be `"."`

#### Scenario: RenderModule uses first arg as module path

- **WHEN** `RenderModule` is called with args `["./my-module"]`
- **THEN** the module path used for `build.RenderOptions.ModulePath` SHALL be `"./my-module"`

### Requirement: ShowRenderOutput checks for errors, shows transformer matches, and logs warnings

The `ShowRenderOutput` function SHALL accept a `*build.RenderResult` and output options (verbose flag, verboseJSON flag). It SHALL:

1. Check `result.HasErrors()` — if true, format and print render errors via `PrintRenderErrors`, then return an `*ExitError` with `ExitValidationError`.
2. Show transformer match output — verbose mode shows module metadata and match reasons; default mode shows compact match lines; verboseJSON mode shows structured JSON.
3. Log any warnings from the render result via the module-scoped logger.

#### Scenario: ShowRenderOutput returns error when render has errors

- **WHEN** `ShowRenderOutput` is called with a `RenderResult` where `HasErrors()` returns `true`
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

- **WHEN** `ShowRenderOutput` is called with a `RenderResult` where `HasWarnings()` returns `true`
- **THEN** each warning SHALL be logged via the module-scoped logger at warn level

### Requirement: NewK8sClient creates a Kubernetes client or returns a connectivity ExitError

The `NewK8sClient` function SHALL accept kubeconfig path, context name, and API warnings setting. It SHALL create a `kubernetes.Client` using `kubernetes.NewClient`. On failure, it SHALL return an `*ExitError` with `Code` equal to `ExitConnectivityError`.

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

The `PrintValidationError` function SHALL accept a message string and an error. When the error is a `*build.ReleaseValidationError` with non-empty `Details`, it SHALL print a summary line followed by the CUE error details. For other errors, it SHALL use the standard key-value log format.

#### Scenario: ReleaseValidationError with CUE details

- **WHEN** `PrintValidationError` is called with a `*build.ReleaseValidationError` that has `Details` set
- **THEN** the output SHALL include the summary message
- **AND** the output SHALL include the CUE details as plain text on stderr

#### Scenario: Generic error without CUE details

- **WHEN** `PrintValidationError` is called with a plain `error`
- **THEN** the output SHALL use the standard error log format with the message and error key-value pair

### Requirement: PrintRenderErrors formats render errors with diagnostic detail

The `PrintRenderErrors` function SHALL accept a slice of errors and print each one with appropriate diagnostic information. For `*build.UnmatchedComponentError`, it SHALL print the component name and list available transformers with their requirements. For `*build.TransformError`, it SHALL print the component name, transformer FQN, and cause.

#### Scenario: Unmatched component error with available transformers

- **WHEN** `PrintRenderErrors` is called with an `UnmatchedComponentError` that has available transformers
- **THEN** the output SHALL include the component name
- **AND** the output SHALL list each available transformer with its FQN, requiredLabels, requiredResources, and requiredTraits

#### Scenario: Transform error

- **WHEN** `PrintRenderErrors` is called with a `TransformError`
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
