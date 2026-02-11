## ADDED Requirements

### Requirement: Overlay generation uses typed AST construction

The ReleaseBuilder SHALL generate the CUE overlay file (`opm_release_overlay.cue`) using CUE AST construction (`cuelang.org/go/cue/ast`) instead of string formatting.

The generated overlay SHALL produce byte-identical CUE output compared to the previous `fmt.Sprintf` approach when both are formatted by `format.Node`.

#### Scenario: AST overlay produces valid CUE

- **WHEN** the ReleaseBuilder generates an overlay for a module with package name `testmodule`, release name `my-release`, and namespace `production`
- **THEN** the output SHALL be valid CUE that parses without errors via `parser.ParseFile`

#### Scenario: AST overlay contains required definitions

- **WHEN** the ReleaseBuilder generates an overlay
- **THEN** the output SHALL contain a `#opmReleaseMeta` definition with fields: `name`, `namespace`, `fqn`, `version`, `identity`, and `labels`
- **AND** `identity` SHALL use `uuid.SHA1` with the OPM namespace UUID and an interpolation of `fqn`, `name`, and `namespace`
- **AND** `labels` SHALL unify `metadata.labels` with the standard release labels (`module-release.opmodel.dev/name`, `module-release.opmodel.dev/version`, `module-release.opmodel.dev/uuid`)

#### Scenario: AST overlay uses correct label types for scope resolution

- **WHEN** the ReleaseBuilder constructs the overlay AST
- **THEN** field labels that are referenced from nested scopes (`name`, `namespace`, `fqn`, `version`, `identity`) SHALL use unquoted identifier labels (`ast.NewIdent`)
- **AND** field labels containing special characters (`module-release.opmodel.dev/*`) SHALL use quoted string labels (`ast.NewString`)
- **AND** `astutil.Resolve` SHALL be called on the constructed `*ast.File` to wire up scope references

#### Scenario: AST overlay matches previous string template output

- **WHEN** the ReleaseBuilder generates an overlay with any valid inputs
- **THEN** loading the AST overlay with a module and evaluating `#opmReleaseMeta.identity` SHALL produce the same UUID as the previous `fmt.Sprintf`-based overlay

### Requirement: ReleaseBuilder provides module inspection without CUE evaluation

The ReleaseBuilder SHALL expose a method to extract module metadata (`name`, `defaultNamespace`, `pkgName`) from a module directory using only CUE loader AST inspection — without calling `BuildInstance` or performing CUE evaluation.

#### Scenario: Metadata extracted from static string literals

- **WHEN** a module defines `metadata: name: "my-module"` and `metadata: defaultNamespace: "default"` as string literals
- **THEN** the inspection method SHALL return `name: "my-module"` and `defaultNamespace: "default"` without performing CUE evaluation

#### Scenario: Package name extracted from loader instance

- **WHEN** a module directory is loaded via `load.Instances`
- **THEN** the inspection method SHALL return the package name from `inst.PkgName`

#### Scenario: Graceful fallback for computed metadata

- **WHEN** a module defines `metadata.name` as an expression (not a string literal)
- **THEN** the inspection method SHALL return an empty string for `name`
- **AND** the pipeline SHALL fall back to extracting metadata via `BuildInstance` + `LookupPath`

## MODIFIED Requirements

### Requirement: ModuleLoader extracts metadata from module

| ID | Requirement |
|----|-------------|
| FR-B-032 | ModuleLoader MUST extract metadata (name, namespace, version) from module. |

The pipeline SHALL extract module metadata (`name`, `defaultNamespace`) using AST inspection of `inst.Files` from a single `load.Instances` call as the primary method. CUE evaluation via `BuildInstance` + `LookupPath` SHALL be used only as a fallback when AST inspection returns empty values.

The pipeline SHALL NOT perform a separate `load.Instances` + `BuildInstance` call solely for metadata extraction when AST inspection succeeds.

#### Scenario: Metadata extracted without CUE evaluation

- **WHEN** the pipeline renders a module with static string literals for `metadata.name` and `metadata.defaultNamespace`
- **THEN** metadata SHALL be extracted from the AST without calling `BuildInstance`
- **AND** the pipeline SHALL use at most two `load.Instances` calls total (inspection + overlay build), not three

#### Scenario: Metadata extraction falls back to evaluation

- **WHEN** the pipeline renders a module where `metadata.name` is a computed expression
- **THEN** the pipeline SHALL fall back to `BuildInstance` + `LookupPath` for metadata extraction
- **AND** the rendered output SHALL be identical to the output before this change
