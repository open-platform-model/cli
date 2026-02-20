# CLI Build Command

## Purpose

The `opm mod build` command renders OPM modules into platform-specific manifests by implementing the Pipeline interface from render-pipeline-v1. The command is implemented in `internal/build/`.

## Design Rationale

1. **Implements Pipeline interface**: Core logic in `internal/build/` satisfies render-pipeline-v1 contract.
2. **Separate output formatting**: CLI-specific formatting in `internal/output/`, not in Pipeline.
3. **#config pattern**: Modules use `#config` for schema, `values` for defaults, enabling type-safe configuration.
4. **Release building phase**: ReleaseBuilder injects values into #config before component extraction.
5. **Parallel execution**: FillPath injection for #component and #context.

## Dependencies

- **render-pipeline-v1**: Implements Pipeline, RenderResult, RenderOptions interfaces
- **config-v1**: Uses OPMConfig for provider resolution
- **platform-adapter-spec**: References #Transformer, #Provider CUE definitions

---

## User Stories

### User Story 1 - Render Module to Kubernetes Manifests (Priority: P1)

A developer wants to convert their OPM module into Kubernetes manifests.

**Independent Test**: Given a valid OPM module, running `opm mod build` produces valid Kubernetes YAML.

**Acceptance Scenarios**:

1. **Given** a module with a Container resource and stateless workload-type label, **When** `opm mod build` runs, **Then** a Kubernetes Deployment is generated.
2. **Given** a module with a Container resource and Expose trait, **When** `opm mod build` runs, **Then** both Deployment and Service are generated.
3. **Given** a module with multiple components, **When** `opm mod build` runs, **Then** all components are transformed and output in a single manifest.

### User Story 2 - Understand Transformer Matching (Priority: P2)

A developer wants to understand which transformers matched their components.

**Acceptance Scenarios**:

1. **Given** a component, **When** running `opm mod build --verbose`, **Then** it shows which transformers matched and why.
2. **Given** a component missing a required label, **When** running `opm mod build --verbose`, **Then** it explains why transformers didn't match.

### User Story 3 - Handle Unmatched Components (Priority: P2)

A developer has a component that doesn't match any transformer.

**Acceptance Scenarios**:

1. **Given** a component with no matching transformers, **When** `opm mod build` runs, **Then** it errors with list of available transformers and their requirements.
2. **Given** `--strict` mode enabled, **When** a component has unhandled traits, **Then** the CLI errors with the list of unhandled traits.

### User Story 4 - Output Format Control (Priority: P3)

A developer wants to control how rendered manifests are output.

**Acceptance Scenarios**:

1. **Given** a module, **When** running `opm mod build -o yaml`, **Then** output is YAML.
2. **Given** a module, **When** running `opm mod build -o json`, **Then** output is JSON.
3. **Given** a module, **When** running `opm mod build --split --out-dir ./manifests`, **Then** each resource is written to a separate file.

---

## Functional Requirements

### CLI Command

| ID | Requirement |
|----|-------------|
| FR-B-001 | `mod build` MUST accept a path argument (default: current directory). |
| FR-B-002 | `mod build` MUST support `--values` / `-f` flag for additional values files (repeatable). |
| FR-B-003 | `mod build` MUST support `--namespace` / `-n` flag to override namespace. |
| FR-B-004 | `mod build` MUST support `--name` flag to override release name. |
| FR-B-005 | `mod build` MUST support `--provider` flag to select provider. |
| FR-B-006 | `mod build` MUST support `--output` / `-o` flag (yaml, json). |
| FR-B-007 | `mod build` MUST support `--split` flag for separate files. |
| FR-B-008 | `mod build` MUST support `--out-dir` flag for split output directory. |
| FR-B-009 | `mod build` MUST support `--strict` flag for strict trait handling. |
| FR-B-010 | `mod build` MUST support `--verbose` flag for matching decisions. |

### Pipeline Implementation

| ID | Requirement |
|----|-------------|
| FR-B-020 | `internal/build.Pipeline` MUST implement render-pipeline-v1 `Pipeline` interface. |
| FR-B-021 | Pipeline MUST execute transformers in parallel. |
| FR-B-022 | Pipeline MUST use FillPath for #component and #context injection. |
| FR-B-023 | Pipeline MUST aggregate errors (fail-on-end pattern). |
| FR-B-024 | Pipeline MUST order resources by weight for sequential apply. |

### Module Loading

| ID | Requirement |
|----|-------------|
| FR-B-030 | ModuleLoader MUST require `values.cue` at module root. |
| FR-B-031 | ModuleLoader MUST unify `values.cue` with `--values` files in order. |
| FR-B-032 | ModuleLoader MUST extract metadata (name, namespace, version) from module. |
| FR-B-033 | `--namespace` MUST take precedence over `module.metadata.defaultNamespace`. |
| FR-B-034 | `--name` MUST take precedence over `module.metadata.name`. |

#### Requirement: ModuleLoader extracts metadata from module

The pipeline SHALL extract all module metadata (`name`, `defaultNamespace`, `fqn`, `version`, `identity`, `labels`) from the fully evaluated `cue.Value` produced by `BuildInstance()`. All metadata fields SHALL be populated via `LookupPath` + `.String()` on the evaluated value. No AST inspection of `inst.Files` SHALL be used for metadata extraction.

`inst.PkgName` SHALL still be read from the `*build.Instance` returned by `load.Instances()`, as it is not available from the evaluated value.

##### Scenario: All metadata extracted from CUE evaluation

- **WHEN** the pipeline loads a module with static string literals for all metadata fields
- **THEN** `name`, `defaultNamespace`, `fqn`, `version`, and `identity` SHALL each be populated from `LookupPath` on the evaluated `cue.Value`
- **AND** no AST walk of `inst.Files` SHALL occur

##### Scenario: Computed metadata name resolves correctly

- **WHEN** a module defines `metadata.name` as a computed CUE expression that evaluates to a concrete string
- **THEN** `mod.Metadata.Name` SHALL be populated with the evaluated concrete string
- **AND** the pipeline SHALL not treat computed names differently from literal names

##### Scenario: Package name extracted from build instance

- **WHEN** a module directory is loaded via `load.Instances()`
- **THEN** `mod.PkgName()` SHALL be populated from `inst.PkgName`

### ReleaseBuilder

#### Requirement: Overlay generation uses typed AST construction

The ReleaseBuilder SHALL generate the CUE overlay file (`opm_release_overlay.cue`) using CUE AST construction (`cuelang.org/go/cue/ast`) instead of string formatting.

The generated overlay SHALL produce byte-identical CUE output compared to the previous `fmt.Sprintf` approach when both are formatted by `format.Node`.

##### Scenario: AST overlay produces valid CUE

- **WHEN** the ReleaseBuilder generates an overlay for a module with package name `testmodule`, release name `my-release`, and namespace `production`
- **THEN** the output SHALL be valid CUE that parses without errors via `parser.ParseFile`

##### Scenario: AST overlay contains required definitions

- **WHEN** the ReleaseBuilder generates an overlay
- **THEN** the output SHALL contain a `#opmReleaseMeta` definition with fields: `name`, `namespace`, `fqn`, `version`, `identity`, and `labels`
- **AND** `identity` SHALL use `uuid.SHA1` with the OPM namespace UUID and an interpolation of `fqn`, `name`, and `namespace`
- **AND** `labels` SHALL unify `metadata.labels` with the standard release labels (`module-release.opmodel.dev/name`, `module-release.opmodel.dev/version`, `module-release.opmodel.dev/uuid`)

##### Scenario: AST overlay uses correct label types for scope resolution

- **WHEN** the ReleaseBuilder constructs the overlay AST
- **THEN** field labels that are referenced from nested scopes (`name`, `namespace`, `fqn`, `version`, `identity`) SHALL use unquoted identifier labels (`ast.NewIdent`)
- **AND** field labels containing special characters (`module-release.opmodel.dev/*`) SHALL use quoted string labels (`ast.NewString`)
- **AND** `astutil.Resolve` SHALL be called on the constructed `*ast.File` to wire up scope references

##### Scenario: AST overlay matches previous string template output

- **WHEN** the ReleaseBuilder generates an overlay with any valid inputs
- **THEN** loading the AST overlay with a module and evaluating `#opmReleaseMeta.identity` SHALL produce the same UUID as the previous `fmt.Sprintf`-based overlay

#### Requirement: Build() gates on IsConcrete() per component

`Build()` SHALL return a non-nil error if any component extracted from `#components` after `FillPath("#config", values)` is not concrete. The check SHALL be performed immediately after `core.ExtractComponents()` returns, before constructing the `ModuleRelease`. The error SHALL identify the component name.

##### Scenario: All components concrete — Build() succeeds

- **WHEN** `Build()` is called with values that satisfy all `#config` constraints
- **AND** all components in `#components` are concrete after `FillPath`
- **THEN** `Build()` SHALL return a non-nil `*core.ModuleRelease` and a nil error

##### Scenario: Non-concrete component — Build() returns error

- **WHEN** after `FillPath("#config", values)` a component's `Value` is not concrete
- **THEN** `Build()` SHALL return a non-nil error containing the component name
- **AND** the error message SHALL indicate the component is not concrete after value injection
- **AND** no `*core.ModuleRelease` SHALL be returned

### Module Configuration Pattern

| ID | Requirement |
|----|-------------|
| FR-B-035 | Modules MUST define `#config` for user-facing configuration schema. |
| FR-B-036 | Modules MUST define `values: #config` to declare values satisfy the schema. |
| FR-B-037 | Components in `#components` MUST reference `#config` for configuration values. |
| FR-B-038 | ReleaseBuilder MUST inject `values` into `#config` via `FillPath` before component extraction. |
| FR-B-039 | ReleaseBuilder MUST validate all extracted components are fully concrete. |

### Transformer Matching

| ID | Requirement |
|----|-------------|
| FR-B-040 | Matcher MUST check requiredLabels, requiredResources, requiredTraits. |
| FR-B-041 | Matcher MUST allow multiple transformers to match one component. |
| FR-B-042 | Matcher MUST report unmatched components with available transformers. |
| FR-B-043 | Matcher MUST track which traits were handled by matched transformers. |

### Output Formatting

| ID | Requirement |
|----|-------------|
| FR-B-050 | Output MUST support YAML format (default). |
| FR-B-051 | Output MUST support JSON format. |
| FR-B-052 | Split output MUST use pattern `<lowercase-kind>-<resource-name>.yaml`. |
| FR-B-053 | Output MUST be deterministic (same input = same output). |

### Error Handling

| ID | Requirement |
|----|-------------|
| FR-B-060 | Unmatched components MUST include list of available transformers. |
| FR-B-061 | Unhandled traits in `--strict` mode MUST cause error. |
| FR-B-062 | Unhandled traits in normal mode MUST cause warning. |
| FR-B-063 | Values file conflicts MUST return CUE's native unification error. |
| FR-B-064 | Non-concrete component after release building MUST fail with ReleaseValidationError. |
| FR-B-065 | Module missing `values` field MUST fail with descriptive error. |

### Requirement: Values validation uses recursive field walking with custom closedness checking

The ReleaseBuilder's values-against-config validation (Step 4b) SHALL validate user-provided values against the `#config` definition by recursively walking every field in the merged values struct and checking each field against the corresponding schema node.

At each level of recursion, the validator SHALL:

1. Check if the field is allowed by the schema using `cue.Value.Allows()`. If not allowed, the validator SHALL emit a "field not allowed" error and SHALL NOT recurse into the field's children.
2. Resolve the schema field — first via literal field lookup (`LookupPath(MakePath(sel))`), then via pattern constraint resolution (`LookupPath(MakePath(Str(key).Optional()))`) for fields matched by `[Name=string]: { ... }` patterns.
3. Unify the data field with the resolved schema field and check for type/constraint errors. If errors exist, the validator SHALL NOT recurse into the field's children.
4. If the field is a struct and passes validation, recurse into its children with the resolved schema field as the new schema context.

The validator SHALL handle arbitrary nesting depth.

#### Scenario: Top-level disallowed field is caught

- **WHEN** values contain a field `"extra-field"` that does not exist in `#config`
- **AND** `#config` is a closed definition with no pattern constraint at the top level
- **THEN** the validator SHALL emit a "field not allowed" error for `values."extra-field"`
- **AND** the error SHALL include the file:line:col position from the values file where the field is defined

#### Scenario: Nested disallowed field inside allowed struct is caught

- **WHEN** values contain `media: { tvshows: { badField: "oops" } }`
- **AND** `#config.media` uses a pattern constraint `[Name=string]: { mountPath: string, size: string }`
- **AND** the pattern's inner struct does not allow `badField`
- **THEN** the validator SHALL emit a "field not allowed" error for `values.media.tvshows.badField`
- **AND** the error SHALL include the file:line:col position of `badField` in the values file

#### Scenario: Pattern constraint fields are not flagged as disallowed

- **WHEN** values contain `media: { tvshows: { mountPath: "/data/tv", size: "100Gi" } }`
- **AND** `#config.media` uses a pattern constraint `[Name=string]: { mountPath: string, size: string }`
- **THEN** the validator SHALL NOT emit any errors for `values.media.tvshows`
- **AND** the validator SHALL recurse into `tvshows` to validate its children against the pattern's constraint struct

#### Scenario: Type mismatch at nested level is caught with correct path

- **WHEN** values contain `media: { movies: "not-a-struct" }`
- **AND** `#config.media` uses a pattern constraint `[Name=string]: { mountPath: string, size: string }`
- **THEN** the validator SHALL emit a type mismatch error for `values.media.movies`
- **AND** the error SHALL NOT recurse into the errored field

#### Scenario: Optional fields in schema are accepted

- **WHEN** `#config` defines `publishedServerUrl?: string`
- **AND** values contain `publishedServerUrl: "https://example.com"`
- **THEN** the validator SHALL NOT emit any errors for `values.publishedServerUrl`

#### Scenario: Empty values struct passes validation

- **WHEN** values contain no fields (empty struct)
- **THEN** the validator SHALL return no errors

### Requirement: Validation error paths use values-rooted paths

All validation errors produced by the values-against-config validation SHALL use paths rooted at `values` (e.g., `values.media."test-key"`) instead of paths rooted at `#config` (e.g., `#config.media."test-key"`).

For "field not allowed" errors generated by the custom closedness checker, the path SHALL be constructed directly with the `values.` prefix.

For type/constraint errors from CUE unification, the error path SHALL be rewritten by prepending the `values`-rooted field path to the CUE error's relative path.

#### Scenario: Closedness error path is values-rooted

- **WHEN** a "field not allowed" error is emitted for a field at `#config.someField`
- **THEN** the error path SHALL be `values.someField`, not `#config.someField`

#### Scenario: Type mismatch error path is values-rooted

- **WHEN** a type mismatch occurs at `#config.media."test-key"`
- **THEN** the error path in the formatted output SHALL be `values.media."test-key"`

#### Scenario: Deeply nested error path is fully qualified

- **WHEN** a validation error occurs 4 levels deep in the values struct
- **THEN** the error path SHALL include all intermediate field names (e.g., `values.a.b.c.d`)

### Requirement: Every validation error includes source file position

All validation errors produced by the values-against-config validation SHALL include at least one `file:line:col` position pointing to the source location in the user's values file.

For "field not allowed" errors, the position SHALL be determined using `cue.Value.Pos()` on the data field value. For unified values (multiple sources), `cue.Value.Expr()` SHALL be used to decompose the value into its constituent conjuncts and extract per-source positions.

For type/constraint errors from CUE unification, the positions from CUE's native error reporting SHALL be preserved (these already include positions from both the schema and data sides).

#### Scenario: Single values file error has position

- **WHEN** a "field not allowed" error is emitted for a field defined in `val.cue` at line 20, column 2
- **THEN** the error output SHALL include `→ ./val.cue:20:2`

#### Scenario: Multi-file error attributes to correct source

- **WHEN** values are provided via `-f base.cue -f overrides.cue`
- **AND** `overrides.cue` introduces a disallowed field at line 5, column 3
- **THEN** the error output SHALL include `→ ./overrides.cue:5:3`

#### Scenario: Type mismatch shows both schema and data positions

- **WHEN** a type mismatch occurs between a schema constraint in `module.cue:46:24` and a value in `val.cue:18:25`
- **THEN** the error output SHALL include both `→ ./module.cue:46:24` and `→ ./val.cue:18:25`

#### Scenario: Graceful fallback when position unavailable

- **WHEN** a validation error occurs on a value that has no source position info (e.g., computed by CUE)
- **THEN** the error SHALL still be reported with its path and message
- **AND** the position line SHALL be omitted rather than showing an invalid position

### Requirement: Multiple values files are unified before validation

When multiple `--values` / `-f` files are provided, the ReleaseBuilder SHALL unify all values files into a single merged value BEFORE running validation against `#config`.

Values files SHALL NOT be validated individually against `#config`, as individual files may be intentionally incomplete (e.g., base configuration in one file, environment-specific overrides in another).

Source attribution for errors in the merged value SHALL use `cue.Value.Expr()` to decompose unified values back to their originating conjuncts with preserved source positions.

#### Scenario: Split values across files validates correctly

- **WHEN** `base.cue` provides `values: { image: "nginx", port: 80 }`
- **AND** `env.cue` provides `values: { timezone: "UTC", configStorageSize: "10Gi" }`
- **AND** both files together satisfy all `#config` requirements
- **THEN** validation SHALL pass without errors
- **AND** neither file SHALL be rejected for being individually incomplete

#### Scenario: Conflicting values between files uses CUE native error

- **WHEN** `a.cue` provides `values: { port: 8080 }`
- **AND** `b.cue` provides `values: { port: 9090 }`
- **THEN** CUE's native unification error SHALL be returned
- **AND** the error SHALL include positions from both files

### Requirement: mod build --verbose shows per-resource validation lines

The `--verbose` output of `opm mod build` SHALL include per-resource validation lines in the "Generated Resources" section. Each resource SHALL be rendered using `FormatResourceLine` with `"valid"` status, matching the `r:<Kind/namespace/name>  <status>` format used by `mod apply`.

This replaces the current plain-text resource listing in verbose output.

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

---

## Non-Functional Requirements

| ID | Requirement |
|----|-------------|
| NFR-B-001 | Module with 5 components MUST render in under 2 seconds (excluding network). |
| NFR-B-002 | No predefined limits on components or transformers. |

---

## Success Criteria

| ID | Criteria |
|----|----------|
| SC-B-001 | Module with 5 components renders in under 2 seconds. |
| SC-B-002 | Same input produces identical output (deterministic). |
| SC-B-003 | 100% of matched components produce valid Kubernetes resources. |
| SC-B-004 | Error messages include actionable guidance. |
| SC-B-005 | Verbose output shows matching decisions. |

---

## Edge Cases

| Case | Handling |
|------|----------|
| Two transformers with identical requirements | Both execute, both produce resources |
| Transformer produces zero resources | Empty result is valid |
| Invalid values file | Fail with CUE validation error |
| Values file conflict | Return CUE's native unification error |
| No namespace provided (and no default) | Fail with "namespace required" error |
| Empty module (no components) | Success with empty resources |
| Non-concrete component after release building | Fail with ReleaseValidationError including component name |
| Module missing `values` field | Fail with "module missing 'values' field" error |
| Module missing `#components` field | Fail with "module missing '#components' field" error |

---

## Command Syntax

```text
opm mod build [path] [flags]

Arguments:
  path    Path to module directory (default: .)

Flags:
  -f, --values strings      Additional values files (can be repeated)
  -n, --namespace string    Target namespace
      --name string         Release name (default: module name)
      --provider string     Provider to use (default: from config)
  -o, --output string       Output format: yaml, json (default: yaml)
      --split               Write separate files per resource
      --out-dir string      Directory for split output (default: ./manifests)
      --strict              Error on unhandled traits
  -v, --verbose             Show matching decisions
  -h, --help                Help for build
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Usage error (invalid flags, missing arguments) |
| 2 | Render error (unmatched components, transform failures) |

---

## Example Usage

```bash
# Basic build
opm mod build ./my-module

# Build with values
opm mod build ./my-module -f prod-values.cue -n production

# Build with split output
opm mod build ./my-module --split --out-dir ./manifests

# Build with verbose output
opm mod build ./my-module --verbose

# Build as JSON
opm mod build ./my-module -o json
```
