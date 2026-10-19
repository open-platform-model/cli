## REMOVED Requirements

### Requirement: mod vet command validates module without generating manifests
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "`opm mod vet` validates values against `#config` and renders the synthesized instance without outputting manifests", "`opm mod vet` accepts a module path argument and values flags for external values files", "`opm mod vet` takes its values from `debugValues` unless `-f` is provided".

### Requirement: mod vet command flags and syntax
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "`opm mod vet` accepts an optional module path argument and exposes `-f`/`--values`, `-n`/`--namespace`, `--name` and `--platform`", "`--instance-name` stays accepted as a deprecated alias of `--name` on `opm mod vet`".

### Requirement: mod vet exit codes
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "`opm mod vet` signals its verdict through the process exit code", "`opm mod vet` maps a failed core-schema fetch as `opm module publish` does and prints the login hint on a refusal".

### Requirement: Identity and coordinate checks before values validation
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "`opm module vet` verifies the identity package, the derived coordinates and the `cue.mod` module line before values resolution", "Failures of the identity and coordinate checks report as validation errors with exit code 2 even without `debugValues`".

## ADDED Requirements

### Requirement: `opm mod vet` validates values against `#config` and renders the synthesized instance without outputting manifests
The `opm mod vet` command SHALL load the module directly, validate values against `#config`, and then render the synthesized instance against the module-deps platform, or against `--platform <dir>` when given (see the `platform-resolution` capability). It SHALL NOT output manifests (YAML/JSON). Its purpose is a pass/fail verdict with clear diagnostics for module authors: identity, values, and whether the module's components render with the catalogs it declares.

#### Scenario: Valid module passes validation using debugValues

- **WHEN** `opm mod vet .` is run on a module that defines `debugValues`
- **AND** no `-f` flag is provided
- **THEN** the command SHALL use `debugValues` as the values source
- **AND** it SHALL print `FormatVetCheck("Values satisfy #config", "debugValues")`
- **AND** it SHALL print `FormatCheckmark("Module config valid")`
- **AND** after the render it SHALL print one validation line per rendered object and a final summary line `FormatCheckmark("Module valid (<n> resources)")`
- **AND** the command SHALL exit with code 0

#### Scenario: Module with CUE validation errors fails with details

- **WHEN** `opm mod vet .` is run on a module with values that do not satisfy `#config`
- **THEN** the command SHALL print the validation error using `PrintValidationError`
- **THEN** error paths SHALL use `values.` prefix (e.g., `values.media."test-key"`) instead of `#config.` prefix
- **THEN** every "field not allowed" error SHALL include at least one `file:line:col` position pointing to the values file that introduced the disallowed field
- **THEN** type mismatch errors SHALL include positions from both the schema file and the values file
- **THEN** grouped error headers SHALL count visible issues rather than raw source locations
- **THEN** the command SHALL exit with code 2

#### Scenario: Nested validation errors show full path with positions

- **WHEN** `opm mod vet .` is run on a module where values contain a type mismatch 3 levels deep
- **THEN** the error path SHALL show the full nested path (e.g., `values.media.movies.mountPath`)
- **THEN** the error SHALL include file:line:col positions for both the schema constraint and the data value

### Requirement: `opm mod vet` accepts a module path argument and values flags for external values files
The `opm mod vet` command SHALL accept a module path argument (default: current directory) and values flags for supplying one or more external values files.

#### Scenario: Multiple values files with disallowed fields show per-file attribution

- **WHEN** `opm mod vet . -f base.cue -f overrides.cue` is run
- **AND** `base.cue` contains a disallowed field `"extra-base"` at line 10
- **AND** `overrides.cue` contains a disallowed field `"extra-override"` at line 5
- **THEN** the error for `values."extra-base"` SHALL include `→ ./base.cue:10:...`
- **THEN** the error for `values."extra-override"` SHALL include `→ ./overrides.cue:5:...`

#### Scenario: Multiple values files report schema errors and merge conflicts together

- **WHEN** `opm mod vet . -f base.cue -f overrides.cue` is run
- **AND** `base.cue` contains schema violations
- **AND** `base.cue` and `overrides.cue` also contain a conflicting assignment
- **THEN** the command SHALL report both the schema violations and the merge conflict in one run

### Requirement: `opm mod vet` takes its values from `debugValues` unless `-f` is provided
When no `-f`/`--values` flag is provided, `opm mod vet` SHALL use the module's `debugValues` field as the values source. If `-f` is provided, it SHALL override `debugValues` and use the specified values files instead. If neither `debugValues` nor `-f` exists, the command SHALL return an error indicating that values must be provided via `debugValues` in the module or via `-f`.

#### Scenario: -f flag overrides debugValues

- **WHEN** `opm mod vet . -f prod-values.cue` is run on a module that defines `debugValues`
- **THEN** the command SHALL use `prod-values.cue` as the values source
- **AND** `debugValues` SHALL be ignored

#### Scenario: No debugValues and no -f flag

- **WHEN** `opm mod vet .` is run on a module that does not define a `debugValues` field
- **AND** no `-f` flag is provided
- **THEN** the command SHALL return an error directing the user to add `debugValues` or provide values with `-f`
- **AND** the exit code SHALL be 2

A `debugValues` field left open (`_`) is a values source, not a missing one: the `#config` check merges it with `#config` and passes when every field has a default. The render that follows synthesizes an instance from it, and synthesis requires concrete values, so the command then refuses it exactly as `opm mod build` does. A module that relies on `#config` defaults declares `debugValues: {}`.

#### Scenario: Open debugValues is refused at synthesis

- **WHEN** `opm mod vet .` is run on a module whose `debugValues` is left open (`_`) and whose `#config` gives every field a default
- **AND** no `-f` flag is provided
- **THEN** the command SHALL print `FormatVetCheck("Values satisfy #config", "debugValues")` and `FormatCheckmark("Module config valid")`
- **AND** it SHALL then refuse at synthesis, naming the incomplete `values`, the verdict `opm mod build` reaches for the same input
- **AND** the exit code SHALL be 2

#### Scenario: Values files against a module without #config

- **WHEN** `opm mod vet . -f values.cue` is run on a module that declares no `#config`
- **THEN** the command SHALL refuse with an error stating that the module does not define `#config` and values files cannot be validated, the verdict `opm mod build` reaches for the same input
- **AND** the exit code SHALL be 2

### Requirement: `opm mod vet` accepts an optional module path argument and exposes `-f`/`--values`, `-n`/`--namespace`, `--name` and `--platform`
The `opm mod vet` command SHALL accept an optional module path argument that defaults to the current directory, and SHALL expose `-f`/`--values` (repeatable), `-n`/`--namespace`, `--name` and `--platform`. All four affect the verdict: `-f` selects the values, `-n` and `--name` set the synthesized instance's namespace and name for the render, and `--platform` renders against that platform module instead of the module-deps platform. The syntax is shown in the scenario `Command syntax`.

#### Scenario: Default flags match expected behavior

- **WHEN** `opm mod vet` is run without any flags
- **THEN** path SHALL default to `"."`
- **AND** the render SHALL use the module-deps platform

#### Scenario: Name flag sets the synthesized instance name

- **WHEN** `opm module vet ./my-module --name web` is run
- **THEN** the synthesized instance SHALL be named `web`
- **AND** nothing about a deprecated flag SHALL be printed

#### Scenario: Command syntax

- **WHEN** `opm mod vet --help` shows the syntax of the command
- **THEN** the syntax SHALL be:

  ```text
  opm mod vet [path] [flags]

  Arguments:
    path    Path to module directory (default: .)

  Flags:
    -f, --values strings        Additional values files (can be repeated)
    -n, --namespace string      Namespace of the synthesized instance
        --name string           Name of the synthesized instance (default: <module name>-debug)
        --platform string       Render against this platform module instead of the module's deps
    -h, --help                  Help for vet
  ```

### Requirement: `--instance-name` stays accepted as a deprecated alias of `--name` on `opm mod vet`
`--instance-name` SHALL stay accepted as a deprecated alias of `--name` with the same effect. It SHALL NOT appear in the command's help, and its use SHALL print one line on standard error that names `--name`. Passing both `--name` and `--instance-name` SHALL be a usage error, and nothing SHALL be validated or rendered.

#### Scenario: Deprecated alias keeps working

- **WHEN** `opm module vet ./my-module --instance-name web` is run
- **THEN** the synthesized instance SHALL be named `web`
- **AND** standard error SHALL carry one line saying that `--instance-name` is deprecated and naming `--name`

#### Scenario: Both spellings together are refused

- **WHEN** `opm module vet ./my-module --name a --instance-name b` is run
- **THEN** the command SHALL fail with a usage error before the module is loaded

### Requirement: `opm mod vet` signals its verdict through the process exit code
The `opm mod vet` command SHALL signal its verdict through the process exit code: 0 when validation and the render pass, 1 on a usage error or when the platform cannot be generated or acquired, 2 on any validation failure or render refusal and when the core schema cannot be loaded because no registry is configured, 3 when the core-schema fetch fails against a configured registry, and 4 when that registry refuses the credentials, as the table in the scenario `Exit code table` details.

#### Scenario: Exit code 0 on success

- **WHEN** `opm mod vet .` succeeds
- **THEN** the exit code SHALL be 0

#### Scenario: Exit code 2 on validation failure

- **WHEN** `opm mod vet .` fails due to CUE errors
- **THEN** the exit code SHALL be 2

#### Scenario: Exit code 2 on render refusal

- **WHEN** `opm mod vet .` passes the `#config` check but a component matches no transformer in the module's catalogs
- **THEN** the command SHALL print the kernel's refusal with its diagnostics
- **AND** the exit code SHALL be 2

#### Scenario: Exit code 2 when no registry is configured

- **WHEN** `opm mod vet .` runs with no `--registry`, no `OPM_REGISTRY`, no `registry` in the config file and no `CUE_REGISTRY`, and the core schema cannot be loaded
- **THEN** the message SHALL say no registry is configured, keep the cause, and point to `opm config init`
- **AND** the exit code SHALL be 2

#### Scenario: Exit code 3 when the registry is unreachable or fails

- **WHEN** the configured registry refuses the connection, or answers the core-schema fetch with 503
- **THEN** the exit code SHALL be 3 and the message SHALL NOT name the login

#### Scenario: Exit code table

- **WHEN** `opm mod vet .` ends
- **THEN** the exit code SHALL be the code of the row that applies:

  | Code | Meaning |
  |------|---------|
  | 0 | Validation and render passed |
  | 1 | Usage error (invalid flags, missing arguments), or the platform could not be generated or acquired (an unpublished pin, a bad `--platform` directory) |
  | 2 | Validation error (CUE errors, invalid values, missing `debugValues`, identity/coordinate check failures), render refusal (unmatched components, unresolved demands, a failed transformer), or no registry configured (core-schema fetch) |
  | 3 | The core-schema fetch failed against a configured registry (unreachable, or a failure answer other than a refused credential) |
  | 4 | The registry refused the credentials on the core-schema fetch (a 401, or a 403 that reaches the cli as a refusal) |

### Requirement: `opm mod vet` maps a failed core-schema fetch as `opm module publish` does and prints the login hint on a refusal
The mapping of a failed core-schema fetch SHALL be the one `opm module publish` uses, and a refusal SHALL print the `opm registry login <host>` hint. (Requirement: `opm mod vet` signals its verdict through the process exit code)

#### Scenario: Exit code 4 when the registry refuses the credentials

- **WHEN** `opm mod vet .` runs against a configured registry that answers the core-schema fetch with 401
- **THEN** the message SHALL say the registry refused the credentials, keep the cause, and end with `opm registry login <host>`
- **AND** the exit code SHALL be 4

#### Scenario: A 403 on the tag lookup reads as a failed registry operation

- **WHEN** the configured registry answers the core-schema fetch with 403, which CUE's registry client reports as "not found"
- **THEN** the exit code SHALL be 3, as it is for `opm module publish`

### Requirement: `opm module vet` verifies the identity package, the derived coordinates and the `cue.mod` module line before values resolution
`opm module vet` SHALL, after loading the module and before values resolution, verify: the identity package conforms to core's `#IdentityPackage` (by unification, surfacing CUE's diagnostic); `metadata.modulePath` and `metadata.version` derive from the identity package's values; and `cue.mod`'s `module:` line agrees with the declared module path.

#### Scenario: Replaced derivation caught at vet

- **WHEN** `metadata.version` states a literal that disagrees with the identity package's `Version`
- **THEN** vet fails naming both values and the derivation fix

### Requirement: Failures of the identity and coordinate checks report as validation errors with exit code 2 even without `debugValues`
Failures SHALL report through the standard validation-error rendering with exit code 2, in the same aligned two-value form the publish refusals use, and SHALL be reported even for a module that declares no `debugValues`. (Requirement: `opm module vet` verifies the identity package, the derived coordinates and the `cue.mod` module line before values resolution)

#### Scenario: Coordinate drift caught at vet

- **WHEN** `cue.mod` and the identity package state different module paths
- **THEN** vet fails with both values and both files named, before any values validation
