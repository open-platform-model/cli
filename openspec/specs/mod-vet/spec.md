# Capability: mod-vet

## Purpose

The `opm mod vet` command provides standalone module-config validation without generating manifests. It validates a module's `debugValues` or explicit values files against `#config`, enabling fast feedback for module authors.

## Requirements

### Requirement: mod vet command validates module without generating manifests

The `opm mod vet` command SHALL load the module directly, validate values against `#config`, and then render the synthesized instance against the module-deps platform, or against `--platform <dir>` when given (see the `platform-resolution` capability). It SHALL NOT output manifests (YAML/JSON). Its purpose is a pass/fail verdict with clear diagnostics for module authors: identity, values, and whether the module's components render with the catalogs it declares.

The command SHALL accept a module path argument (default: current directory) and values flags for supplying one or more external values files.

When no `-f`/`--values` flag is provided, `opm mod vet` SHALL use the module's `debugValues` field as the values source. If `-f` is provided, it SHALL override `debugValues` and use the specified values files instead. If neither `debugValues` nor `-f` exists, the command SHALL return an error indicating that values must be provided via `debugValues` in the module or via `-f`.

#### Scenario: Valid module passes validation using debugValues

- **WHEN** `opm mod vet .` is run on a module that defines `debugValues`
- **AND** no `-f` flag is provided
- **THEN** the command SHALL use `debugValues` as the values source
- **AND** it SHALL print `FormatVetCheck("Values satisfy #config", "debugValues")`
- **AND** it SHALL print `FormatCheckmark("Module config valid")`
- **AND** after the render it SHALL print one validation line per rendered object and a final summary line `FormatCheckmark("Module valid (<n> resources)")`
- **AND** the command SHALL exit with code 0

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

#### Scenario: Module with CUE validation errors fails with details

- **WHEN** `opm mod vet .` is run on a module with values that do not satisfy `#config`
- **THEN** the command SHALL print the validation error using `PrintValidationError`
- **THEN** error paths SHALL use `values.` prefix (e.g., `values.media."test-key"`) instead of `#config.` prefix
- **THEN** every "field not allowed" error SHALL include at least one `file:line:col` position pointing to the values file that introduced the disallowed field
- **THEN** type mismatch errors SHALL include positions from both the schema file and the values file
- **THEN** grouped error headers SHALL count visible issues rather than raw source locations
- **THEN** the command SHALL exit with code 2

#### Scenario: Multiple values files with disallowed fields show per-file attribution

- **WHEN** `opm mod vet . -f base.cue -f overrides.cue` is run
- **AND** `base.cue` contains a disallowed field `"extra-base"` at line 10
- **AND** `overrides.cue` contains a disallowed field `"extra-override"` at line 5
- **THEN** the error for `values."extra-base"` SHALL include `→ ./base.cue:10:...`
- **THEN** the error for `values."extra-override"` SHALL include `→ ./overrides.cue:5:...`

#### Scenario: Nested validation errors show full path with positions

- **WHEN** `opm mod vet .` is run on a module where values contain a type mismatch 3 levels deep
- **THEN** the error path SHALL show the full nested path (e.g., `values.media.movies.mountPath`)
- **THEN** the error SHALL include file:line:col positions for both the schema constraint and the data value

#### Scenario: Multiple values files report schema errors and merge conflicts together

- **WHEN** `opm mod vet . -f base.cue -f overrides.cue` is run
- **AND** `base.cue` contains schema violations
- **AND** `base.cue` and `overrides.cue` also contain a conflicting assignment
- **THEN** the command SHALL report both the schema violations and the merge conflict in one run

### Requirement: mod vet renders against the module's deps

After the identity and coordinate checks and the `#config` validation pass, `opm mod vet` SHALL synthesize the module's instance from the same values and render it through the kernel against the resolved platform, exactly as `opm mod build` does, and SHALL report the rendered objects without printing them. A failure in the identity checks or the `#config` validation SHALL stop the command before any platform is generated or acquired, so a cheap failure never reaches the registry.

#### Scenario: Config failure stops before the render

- **WHEN** `opm mod vet .` runs and the resolved values leave a required `#config` field incomplete
- **THEN** the command SHALL print the standard grouped validation block under "values do not satisfy #config", naming the incomplete field and the source position
- **AND** no platform SHALL be generated or acquired
- **AND** the exit code SHALL be 2

#### Scenario: Build and vet reach one verdict

- **WHEN** `opm mod vet . -f values.cue` and `opm mod build . -f values.cue` are run on the same module, values file and platform source
- **THEN** both SHALL succeed, or both SHALL refuse with the same `#config` violations or the same render refusal

#### Scenario: The platform flag switches vet to a platform

- **WHEN** `opm mod vet . --platform ./pulled/` runs
- **THEN** the render SHALL use the platform module at `./pulled/`
- **AND** the output SHALL report that source

### Requirement: mod vet accepts values files for validation

The `opm mod vet` command SHALL support `--values` / `-f` flags for providing external values files (CUE format), matching the behavior of `mod build`.

#### Scenario: Validate with external values

- **WHEN** `opm mod vet . -f prod-values.cue` is run
- **THEN** the command SHALL validate `prod-values.cue` against the module's `#config`
- **THEN** validation SHALL use the merged values

### Requirement: mod vet command flags and syntax

The `opm mod vet` command SHALL accept an optional module path argument that defaults to the current directory, and SHALL expose `-f`/`--values` (repeatable), `-n`/`--namespace`, `--name` and `--platform`. All four affect the verdict: `-f` selects the values, `-n` and `--name` set the synthesized instance's namespace and name for the render, and `--platform` renders against that platform module instead of the module-deps platform.

`--instance-name` SHALL stay accepted as a deprecated alias of `--name` with the same effect. It SHALL NOT appear in the command's help, and its use SHALL print one line on standard error that names `--name`. Passing both `--name` and `--instance-name` SHALL be a usage error, and nothing SHALL be validated or rendered.

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

#### Scenario: Default flags match expected behavior

- **WHEN** `opm mod vet` is run without any flags
- **THEN** path SHALL default to `"."`
- **AND** the render SHALL use the module-deps platform

#### Scenario: Name flag sets the synthesized instance name

- **WHEN** `opm module vet ./my-module --name web` is run
- **THEN** the synthesized instance SHALL be named `web`
- **AND** nothing about a deprecated flag SHALL be printed

#### Scenario: Deprecated alias keeps working

- **WHEN** `opm module vet ./my-module --instance-name web` is run
- **THEN** the synthesized instance SHALL be named `web`
- **AND** standard error SHALL carry one line saying that `--instance-name` is deprecated and naming `--name`

#### Scenario: Both spellings together are refused

- **WHEN** `opm module vet ./my-module --name a --instance-name b` is run
- **THEN** the command SHALL fail with a usage error before the module is loaded

### Requirement: mod vet exit codes

The `opm mod vet` command SHALL signal its verdict through the process exit code: 0 when validation and the render pass, 1 on a usage error or when the platform cannot be generated or acquired, 2 on any validation failure or render refusal, and 3 when the registry cannot be reached for the core schema, as the table below details.

| Code | Meaning |
|------|---------|
| 0 | Validation and render passed |
| 1 | Usage error (invalid flags, missing arguments), or the platform could not be generated or acquired (an unpublished pin, a bad `--platform` directory) |
| 2 | Validation error (CUE errors, invalid values, missing `debugValues`, identity/coordinate check failures) or render refusal (unmatched components, unresolved demands, a failed transformer) |
| 3 | Registry unreachable (core-schema fetch) |

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

### Requirement: Identity and coordinate checks before values validation

`opm module vet` SHALL, after loading the module and before values resolution, verify: the identity package conforms to core's `#IdentityPackage` (by unification, surfacing CUE's diagnostic); `metadata.modulePath` and `metadata.version` derive from the identity package's values; and `cue.mod`'s `module:` line agrees with the declared module path. Failures SHALL report through the standard validation-error rendering with exit code 2, in the same aligned two-value form the publish refusals use, and SHALL be reported even for a module that declares no `debugValues`.

#### Scenario: Coordinate drift caught at vet

- **WHEN** `cue.mod` and the identity package state different module paths
- **THEN** vet fails with both values and both files named, before any values validation

#### Scenario: Replaced derivation caught at vet

- **WHEN** `metadata.version` states a literal that disagrees with the identity package's `Version`
- **THEN** vet fails naming both values and the derivation fix

### Requirement: Registry-aware loading

Vet's module load SHALL resolve dependencies using the resolved registry (flag > env > config precedence), not the ambient process environment alone. Failing to reach the registry for the core-schema fetch SHALL exit 3 (connectivity), distinct from check failures (2).

#### Scenario: Registry flag respected

- **WHEN** vet runs with `--registry` pointing at a reachable mapping
- **THEN** the module's dependencies resolve through that mapping
