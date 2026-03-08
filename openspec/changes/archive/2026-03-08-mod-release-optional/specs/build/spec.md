## MODIFIED Requirements

### Requirement: Module Loading — values.cue is not required at module root

`FR-B-030` in the main spec states `ModuleLoader MUST require values.cue at module root`. This requirement is superseded.

When no `--values` files are provided, the loader SHALL first check for `values.cue` in the module directory. If `values.cue` is absent, the loader SHALL fall back to `debugValues` from the module. Only when neither `values.cue`, `debugValues`, nor `--values` files are present SHALL the loader return an error.

`FR-B-031` (`ModuleLoader MUST unify values.cue with --values files`) is also superseded. The fallback chain is: `--values` files (if provided) → `values.cue` (if present) → `debugValues` (if defined). These sources are not unified together — each is a mutually exclusive fallback tier.

#### Scenario: No --values files and no values.cue — debugValues used
- **WHEN** `opm mod build .` is run with no `--values` flag
- **AND** no `values.cue` exists in the module directory
- **AND** the module defines `debugValues`
- **THEN** the loader SHALL use `debugValues` as the values source without error

#### Scenario: No --values files, no values.cue, no debugValues — error
- **WHEN** `opm mod build .` is run with no `--values` flag
- **AND** no `values.cue` exists in the module directory
- **AND** the module has no `debugValues` field
- **THEN** the loader SHALL return an error indicating the user must provide values via `values.cue`, `debugValues`, or `--values`

### Requirement: ModuleLoader extracts metadata from module

The pipeline SHALL extract all module metadata (`name`, `defaultNamespace`, `fqn`, `version`, `identity`, `labels`) from the fully evaluated `cue.Value` produced by `BuildInstance()`. All metadata fields SHALL be populated via `LookupPath` + `.String()` on the evaluated value. No AST inspection of `inst.Files` SHALL be used for metadata extraction.

`inst.PkgName` SHALL still be read from the `*build.Instance` returned by `load.Instances()`, as it is not available from the evaluated value.

When no `release.cue` is present and synthesis mode is active, module metadata SHALL be decoded from the module value's `metadata` field directly (not via `#module.metadata` as in the release value path).

#### Scenario: All metadata extracted from CUE evaluation
- **WHEN** the pipeline loads a module with static string literals for all metadata fields
- **THEN** `name`, `defaultNamespace`, `fqn`, `version`, and `identity` SHALL each be populated from `LookupPath` on the evaluated `cue.Value`
- **AND** no AST walk of `inst.Files` SHALL occur

#### Scenario: Computed metadata name resolves correctly
- **WHEN** a module defines `metadata.name` as a computed CUE expression that evaluates to a concrete string
- **THEN** `mod.Metadata.Name` SHALL be populated with the evaluated concrete string
- **AND** the pipeline SHALL not treat computed names differently from literal names

#### Scenario: Package name extracted from build instance
- **WHEN** a module directory is loaded via `load.Instances()`
- **THEN** `mod.PkgName()` SHALL be populated from `inst.PkgName`

#### Scenario: Synthesis mode decodes module metadata from module value
- **WHEN** `opm mod build .` runs in synthesis mode (no `release.cue`)
- **AND** the module defines `metadata.name: "jellyfin"` and `metadata.defaultNamespace: "jellyfin"`
- **THEN** the synthesized `ModuleRelease.Module.Metadata.Name` SHALL be `"jellyfin"`
- **AND** the synthesized `ModuleRelease.Metadata.Namespace` SHALL be `"jellyfin"`

## MODIFIED Requirements

### Requirement: mod build --verbose shows per-resource validation lines

The `--verbose` output of `opm mod build` SHALL include per-resource validation lines in the "Generated Resources" section. Each resource SHALL be rendered using `FormatResourceLine` with `"valid"` status, matching the `r:<Kind/namespace/name>  <status>` format used by `mod apply`.

This applies in both the normal (release.cue-backed) and synthesis (no release.cue) modes.

#### Scenario: Verbose output renders resources with FormatResourceLine
- **WHEN** `opm mod build . --verbose` is run on a valid module that generates 3 resources
- **THEN** the "Generated Resources" section SHALL contain 3 lines
- **THEN** each line SHALL use `FormatResourceLine(kind, namespace, name, "valid")` format
- **THEN** the `r:` prefix SHALL be dim, resource path SHALL be cyan, and `"valid"` SHALL be green

#### Scenario: Verbose output aligns with mod apply resource output
- **WHEN** `opm mod build . --verbose` generates a `StatefulSet/default/jellyfin` resource
- **THEN** the verbose output SHALL render: `r:StatefulSet/default/jellyfin          valid`
- **THEN** the format SHALL be visually consistent with `mod apply`'s `r:StatefulSet/default/jellyfin  created`

#### Scenario: Build validation errors show values-rooted paths
- **WHEN** `opm mod build . -f values.cue` fails due to a disallowed field in values
- **THEN** the error details SHALL show paths rooted at `values.` (e.g., `values."extra-field"`)
- **AND** the error details SHALL include file:line:col positions from the values file

#### Scenario: Verbose output works in synthesis mode
- **WHEN** `opm mod build . --verbose` is run with no `release.cue` using `debugValues`
- **THEN** the verbose output SHALL include per-resource validation lines
- **AND** the output format SHALL be identical to release-backed builds

## ADDED Requirements

### Requirement: mod build defaults to debugValues when no -f flag is given
When `opm mod build` is invoked without a `-f` / `--values` flag, the command SHALL automatically use the module's `debugValues` field as the values source. This applies in both synthesis mode (no `release.cue`) and release mode (has `release.cue`).

#### Scenario: No -f flag, debugValues present — build succeeds
- **WHEN** `opm mod build .` is run with no `-f` flag
- **AND** the module defines `debugValues`
- **THEN** the build SHALL succeed using those values

#### Scenario: No -f flag, no debugValues — build fails with actionable error
- **WHEN** `opm mod build .` is run with no `-f` flag
- **AND** the module has no `debugValues` field
- **AND** no `release.cue` is present
- **THEN** the command SHALL fail with a message directing the user to add `debugValues` or use `-f`
