# Delta Spec: Render Pipeline Interface

## Overview

This delta defines the shared interface for the OPM render pipeline. The interface enables multiple CLI commands to use the same rendering logic while maintaining clear boundaries between rendering and consumption.

## Design Decisions

1. **Interface-based design**: Consumers depend on `Pipeline` interface, not implementation.
2. **RenderResult as contract**: Single struct contains all render output, enabling type-safe consumption.
3. **Fail-on-end in results**: Aggregated errors in `RenderResult.Errors` rather than failing immediately.
4. **Unstructured resources**: Platform-agnostic resource representation using `*unstructured.Unstructured`.

## Clarifications

- **Fatal vs Render errors**: Fatal errors (module not found, invalid config) return from `Render()`. Render errors (unmatched components) are in `RenderResult.Errors`.
- **Resource ordering**: Resources in `RenderResult.Resources` are ordered for sequential apply (respecting dependencies).
- **MatchPlan purpose**: Debugging and verbose output only; consumers SHOULD NOT depend on its structure for logic.

---

## User Stories

### User Story 1 - Build Command Uses Pipeline (Priority: P1)

The build command needs to render a module and output manifests.

**Independent Test**: Build command calls Pipeline.Render() and formats RenderResult.Resources as YAML.

**Acceptance Scenarios**:

1. **Given** a valid module, **When** build calls Pipeline.Render(), **Then** RenderResult contains rendered resources.
2. **Given** a module with render errors, **When** build calls Pipeline.Render(), **Then** RenderResult.Errors contains the errors.
3. **Given** RenderResult with resources, **When** build formats output, **Then** YAML contains all resources in order.

### User Story 2 - Apply Command Uses Pipeline (Priority: P1)

The apply command needs to render a module and deploy to Kubernetes.

**Independent Test**: Apply command calls Pipeline.Render() and passes RenderResult.Resources to kubernetes.Apply().

**Acceptance Scenarios**:

1. **Given** a valid module, **When** apply calls Pipeline.Render(), **Then** it receives same RenderResult as build.
2. **Given** RenderResult.Resources, **When** apply calls kubernetes.Apply(), **Then** resources are deployed in order.
3. **Given** RenderResult with Errors, **When** apply processes result, **Then** it can decide whether to proceed or abort.

### User Story 3 - Diff Command Uses Pipeline (Priority: P2)

The diff command needs to compare rendered resources with live cluster state.

**Independent Test**: Diff command calls Pipeline.Render() and compares RenderResult.Resources with cluster.

**Acceptance Scenarios**:

1. **Given** a valid module, **When** diff calls Pipeline.Render(), **Then** it receives resources for comparison.
2. **Given** RenderResult.Resources, **When** diff fetches live state, **Then** it can compare each resource.
3. **Given** partial RenderResult (some errors), **When** diff processes, **Then** it can still compare successful resources.

---

## Functional Requirements

### Pipeline Interface

| ID | Requirement |
|----|-------------|
| FR-RP-001 | Pipeline MUST expose a `Render(ctx, opts)` method returning `(*RenderResult, error)`. |
| FR-RP-002 | Pipeline MUST return fatal errors (module not found, config invalid) as the error return value. |
| FR-RP-003 | Pipeline MUST return render errors (unmatched components, transform failures) in `RenderResult.Errors`. |
| FR-RP-004 | Pipeline MUST support context cancellation for long-running operations. |

### RenderOptions

| ID | Requirement |
|----|-------------|
| FR-RP-010 | RenderOptions MUST support `ModulePath` for the module directory. |
| FR-RP-011 | RenderOptions MUST support `Values` for additional values files. |
| FR-RP-012 | RenderOptions MUST support `Name` to override module name. |
| FR-RP-013 | RenderOptions MUST support `Namespace` to override default namespace. |
| FR-RP-014 | RenderOptions MUST support `Provider` to select the provider. |
| FR-RP-015 | RenderOptions MUST support `Strict` for strict trait handling. |

### RenderResult

| ID | Requirement |
|----|-------------|
| FR-RP-020 | RenderResult.Resources MUST be ordered for sequential apply (dependencies first). |
| FR-RP-021 | RenderResult.Module MUST contain source module metadata. |
| FR-RP-022 | RenderResult.MatchPlan MUST describe transformer-component matches. |
| FR-RP-023 | RenderResult.Errors MUST aggregate all render errors (fail-on-end). |
| FR-RP-024 | RenderResult.Warnings MUST contain non-fatal warnings. |

### Resource

| ID | Requirement |
|----|-------------|
| FR-RP-030 | Resource.Object MUST be `*unstructured.Unstructured`. |
| FR-RP-031 | Resource.Component MUST identify the source component. |
| FR-RP-032 | Resource.Transformer MUST identify the transformer FQN. |
| FR-RP-033 | Resources MUST include OPM tracking labels (set by transformer). |

### Error Handling

| ID | Requirement |
|----|-------------|
| FR-RP-040 | UnmatchedComponentError MUST include available transformers list. |
| FR-RP-041 | UnhandledTraitError MUST indicate whether strict mode is enabled. |
| FR-RP-042 | TransformError MUST include both component and transformer identification. |

---

## Non-Functional Requirements

| ID | Requirement |
|----|-------------|
| NFR-RP-001 | Interface MUST be stable for at least one major version. |
| NFR-RP-002 | Interface MUST support future Bundle rendering without breaking changes. |

---

## Success Criteria

| ID | Criteria |
|----|----------|
| SC-RP-001 | Build command can implement using only Pipeline interface. |
| SC-RP-002 | Apply command can consume RenderResult without knowledge of rendering internals. |
| SC-RP-003 | Different implementations can satisfy Pipeline interface (for testing/future). |

---

## Edge Cases

| Case | Handling |
|------|----------|
| No resources rendered | RenderResult.Resources is empty slice, not nil. |
| All components failed | RenderResult.Resources is empty, Errors contains all failures. |
| Partial success | RenderResult contains both Resources and Errors. |
| Context cancelled | Return error from Render(), not in RenderResult. |
| Empty module (no components) | RenderResult with empty Resources, no errors. |
