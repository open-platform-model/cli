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

The pipeline SHALL extract module metadata (`name`, `defaultNamespace`) using AST inspection of `inst.Files` from a single `load.Instances` call as the primary method. CUE evaluation via `BuildInstance` + `LookupPath` SHALL be used only as a fallback when AST inspection returns empty values.

The pipeline SHALL NOT perform a separate `load.Instances` + `BuildInstance` call solely for metadata extraction when AST inspection succeeds.

##### Scenario: Metadata extracted without CUE evaluation

- **WHEN** the pipeline renders a module with static string literals for `metadata.name` and `metadata.defaultNamespace`
- **THEN** metadata SHALL be extracted from the AST without calling `BuildInstance`
- **AND** the pipeline SHALL use at most two `load.Instances` calls total (inspection + overlay build), not three

##### Scenario: Metadata extraction falls back to evaluation

- **WHEN** the pipeline renders a module where `metadata.name` is a computed expression
- **THEN** the pipeline SHALL fall back to `BuildInstance` + `LookupPath` for metadata extraction
- **AND** the rendered output SHALL be identical to the output before this change

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

#### Requirement: ReleaseBuilder provides module inspection without CUE evaluation

The ReleaseBuilder SHALL expose a method to extract module metadata (`name`, `defaultNamespace`, `pkgName`) from a module directory using only CUE loader AST inspection — without calling `BuildInstance` or performing CUE evaluation.

##### Scenario: Metadata extracted from static string literals

- **WHEN** a module defines `metadata: name: "my-module"` and `metadata: defaultNamespace: "default"` as string literals
- **THEN** the inspection method SHALL return `name: "my-module"` and `defaultNamespace: "default"` without performing CUE evaluation

##### Scenario: Package name extracted from loader instance

- **WHEN** a module directory is loaded via `load.Instances`
- **THEN** the inspection method SHALL return the package name from `inst.PkgName`

##### Scenario: Graceful fallback for computed metadata

- **WHEN** a module defines `metadata.name` as an expression (not a string literal)
- **THEN** the inspection method SHALL return an empty string for `name`
- **AND** the pipeline SHALL fall back to extracting metadata via `BuildInstance` + `LookupPath`

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
      --verbose-json        Structured JSON verbose output
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
