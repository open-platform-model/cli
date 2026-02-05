# Delta Spec: CLI Build Command

## Overview

This delta adds requirements for the `opm mod build` command and the render pipeline that transforms OPM modules into platform-specific manifests.

The rendering pipeline consists of three key concepts:

- **Provider**: Platform adapter containing transformer registry
- **Transformer**: Converts OPM components to platform resources
- **Render Pipeline**: CLI orchestration from Module to output manifests

## Design Decisions

1. **CLI-only rendering**: Rendering logic resides in the CLI, not in CUE schemas.
2. **Kubernetes-first**: Focus on Kubernetes as primary target, with hooks for future extensibility.
3. **Parallel Execution**: Component rendering occurs in parallel goroutines.
4. **CUE-First Unification**: CLI relies on CUE's unification engine for validation and conflict resolution.

## Clarifications

- **Provider Resolution**: Providers are defined in `~/.opm/config.cue` by importing provider modules. The `--provider` flag selects which configured provider to use. Modules MUST NOT declare or reference providers.
- **Error Handling**: Fail on End - render all components, collect errors, exit with non-zero status after all are processed.
- **Verbose Output**: Both human-readable (default `--verbose`) and structured JSON (`--verbose=json`).
- **TransformerContext Fields**: `name`, `namespace`, `version`, `provider`, `timestamp` (RFC3339), `strict` (bool), `labels`.
- **File Naming**: With `--split`, use pattern `<lowercase-kind>-<resource-name>.yaml`.
- **Sensitive Data**: Redact secrets in logs.
- **Scalability**: No defined limits on components/transformers.

---

## User Stories

### User Story 1 - Render Module to Kubernetes Manifests (Priority: P1)

A developer wants to convert their OPM module into Kubernetes manifests.

**Independent Test**: Given a valid OPM module, running `opm mod build` produces valid Kubernetes YAML.

**Acceptance Scenarios**:

1. **Given** a module with a Container resource and stateless workload-type label, **When** the CLI renders, **Then** a Kubernetes Deployment is generated.
2. **Given** a module with a Container resource and Expose trait, **When** the CLI renders, **Then** both a Deployment and Service are generated.
3. **Given** a module with multiple components, **When** the CLI renders, **Then** all components are transformed and output in a single manifest.

### User Story 2 - Understand Why Resources Were Generated (Priority: P2)

A developer wants to understand which transformers matched their components.

**Acceptance Scenarios**:

1. **Given** a component with Container and Expose, **When** the CLI renders with verbose output, **Then** it shows which transformers matched and why.
2. **Given** a component missing a required label, **When** the CLI renders with verbose output, **Then** it explains why certain transformers didn't match.

### User Story 3 - Handle Unmatched Components (Priority: P2)

A developer has a component that doesn't match any transformer.

**Acceptance Scenarios**:

1. **Given** a component with no matching transformers, **When** the CLI renders, **Then** it errors with list of available transformers and their requirements.
2. **Given** `--strict` mode enabled, **When** a component has unhandled traits, **Then** the CLI errors with the list of unhandled traits.

### User Story 4 - Output Format Control (Priority: P3)

A developer wants to control how rendered manifests are output.

**Acceptance Scenarios**:

1. **Given** a module, **When** running `opm mod build -o yaml`, **Then** output is a single YAML document.
2. **Given** a module, **When** running `opm mod build --split --out-dir ./manifests`, **Then** each resource is written to a separate file.

---

## Functional Requirements

### Render Pipeline

| ID | Requirement |
|----|-------------|
| FR-015 | The render pipeline MUST execute transformers in parallel. |
| FR-016 | Generated resources MUST include OPM tracking labels via TransformerContext. |
| FR-017 | Support `yaml`, `json` output formats. |
| FR-018 | Support `--split` for separate files per resource. |
| FR-023 | Aggregate outputs deterministically. |
| FR-024 | Aggregate errors (fail-on-end). |
| FR-025 | Support verbose logging (human/json). |
| FR-026 | File naming for `--split`: `<lowercase-kind>-<resource-name>.yaml`. |
| FR-027 | Redact sensitive values in verbose logging. |

### ModuleRelease Construction

| ID | Requirement |
|----|-------------|
| FR-040 | `mod build` MUST construct `#ModuleRelease` internally from local module path. |
| FR-041 | If `--values` not provided, look for `values.cue` at module root. Fail if not found. |
| FR-042 | Multiple values files MUST be unified using CUE. Fail on conflicts. |
| FR-043 | `--namespace` flag takes precedence over `#Module.metadata.defaultNamespace`. Fail if neither provided. |
| FR-044 | `--name` flag takes precedence over `#Module.metadata.name`. |

### Error Handling

| ID | Requirement |
|----|-------------|
| FR-019 | Error on unmatched components (aggregated). |
| FR-020 | Error on unhandled traits in `--strict` mode. |
| FR-021 | Warning on unhandled traits in normal mode. |

---

## Non-Functional Requirements

| ID | Requirement |
|----|-------------|
| NFR-001 | No predefined limits on components or transformers. Scale linearly. |

---

## Success Criteria

| ID | Criteria |
|----|----------|
| SC-001 | Module with 5 components renders in under 2 seconds (excluding network). |
| SC-002 | Transformer matching is deterministic - same input produces identical output. |
| SC-003 | 100% of matched components produce valid Kubernetes resources. |
| SC-004 | Error messages for unmatched components include actionable guidance. |
| SC-005 | Verbose output shows transformer matching decisions. |

---

## Edge Cases

| Case | Handling |
|------|----------|
| Two transformers with identical requirements | Error with "multiple exact transformer matches". |
| Transformer produces zero resources | Empty output is valid. |
| Output fails Kubernetes schema validation | Warning logged; apply will fail server-side. |
| Invalid values file | Fail with clear CUE validation error. |
| Values file conflict | Return CUE's native unification error. |

---

## Render Pipeline Detail

```text
+-------------------------------------------------------------------+
|                       Hybrid Render Pipeline                       |
+-------------------------------------------------------------------+
|  Phase 1: Module Loading & Validation                        [Go]  |
|           - Load CUE via cue/load                                  |
|           - Extract release metadata                               |
|           - Build base TransformerContext                          |
+-------------------------------------------------------------------+
|  Phase 2: Provider Loading                                   [Go]  |
|           - Access provider.transformers from CUE                  |
+-------------------------------------------------------------------+
|  Phase 3: Component Matching                                [CUE]  |
|           - CUE evaluates #Matches predicate                       |
|           - CUE computes #matchedTransformers map                  |
|           - Go reads back the computed matching plan               |
+-------------------------------------------------------------------+
|  Phase 4: Parallel Transformer Execution                     [Go]  |
|           - Iterate CUE-computed matches                           |
|           - Unify transformer.#transform + inputs                  |
|           - Workers: isolated cue.Context -> Decode output         |
+-------------------------------------------------------------------+
|  Phase 5: Aggregation & Output                               [Go]  |
|           - Collect results from workers                           |
|           - Aggregate errors (fail-on-end)                         |
|           - Output YAML manifests                                  |
+-------------------------------------------------------------------+
```

### Phase 1: Module Loading & ModuleRelease Construction

- **Initialization**: Load CLI config from `~/.opm/config.cue`
- **Module Loading**: Load `#Module` from local path using resolved registry
- **Values Resolution**: Start with `values.cue` (required), unify with `--values` files
- **ModuleRelease Construction**: Create `#ModuleRelease` on the fly with resolved name/namespace
- **Validation**: Verify constructed `#ModuleRelease` against schema

### Phase 2: Provider Loading

- Determine provider from `--provider` flag or default
- Access provider definition from loaded config
- Validate provider structure (shallow)
- Index transformers for matching

### Phase 3: Component Matching

- Iterate through all components
- Match each component against ALL transformers
- Construct `matchedTransformers` grouping components by transformer

### Phase 4: Parallel Transformer Execution

- Iterate `matchedTransformers` map in parallel
- Inject `TransformerContext` with OPM tracking labels
- Execute `#transform` for each component
- Each execution produces exactly one resource

### Phase 5: Aggregation & Output

- Collect all generated resources
- Aggregate errors and report together
- Serialize to YAML/JSON or files

---

## Transformer Matching Logic

**Concept: Capability vs. Intent**

1. **Capability (Resources & Traits)**: Does the component have the necessary data?
2. **Intent (Labels)**: Does the component have the specific label to disambiguate?

A transformer matches if ALL conditions are met:

1. **Required Labels**: Present in effective labels
2. **Required Resources**: Present in component resources
3. **Required Traits**: Present in component traits

Multiple transformers can match a single component. If no transformers match, it's an error.

---

## Key Entities

- **TransformerContext**: `name`, `namespace`, `version`, `provider`, `timestamp`, `strict`, `labels`
- **MatchedMap**: Internal structure grouping components by transformer
