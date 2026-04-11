# ADR-005: Interface-Based Render Pipeline

## Status

Accepted

## Context

Multiple CLI commands (build, apply, diff, status) need to render CUE modules into Kubernetes manifests. The render pipeline involves five phases: preparation, provider loading, build, component matching, and resource generation.

Consumers of render output need both the generated resources and metadata about the module and release — names, UUIDs, labels, and components. Module identity (what was built) and release identity (how it was deployed) are semantically distinct: module UUID and release UUID differ, and module name and release name can differ when `--name` overrides the default.

Errors during rendering fall into two categories: fatal errors (module not found, invalid config) that prevent any output, and render errors (unmatched components, non-concrete fields) that should be reported alongside partial results. Without a deliberate separation, callers either bail too early or miss accumulated problems.

A legacy implementation in `internal/legacy/` has grown entangled with command concerns, making it hard to test individual commands in isolation and difficult to ensure consistent output handling across the command set.

## Decision

Define a `Pipeline` interface that all consumers depend on, rather than coupling to a concrete implementation. Return all render output through a single `RenderResult` struct containing: generated resources as `*unstructured.Unstructured` (platform-agnostic), aggregated errors, warnings, and metadata.

Split metadata into two types:

- `ModuleMetadata` — Name, DefaultNamespace, FQN, Version, UUID, Labels, Annotations, Components. Fields like FQN and Version describe the source module and do not appear on the release.
- `ReleaseMetadata` — Name, Namespace, UUID, Labels, Annotations, Components. Fields describe the deployed instance and its runtime identity.

Use fail-on-end error aggregation: render errors are collected in `RenderResult.Errors` rather than failing immediately, so users see all problems at once. Fatal errors (module not found, config invalid) return from `Render()` directly without a result.

Delegate build-phase validation to `ModuleRelease` receiver methods (`ValidateValues()`, `Validate()`) rather than standalone functions, keeping validation co-located with the type it validates.

Delegate generate-phase execution to the match plan (`matchPlan.Execute()`) rather than constructing a separate Executor, reducing the number of moving parts in the pipeline.

Replace the legacy `internal/legacy/` package with `internal/pipeline/`. All pipeline imports must migrate to the new package path.

## Consequences

**Positive:** Interface isolation enables testing without a real pipeline implementation; commands depend on behavior, not structure.

**Positive:** A single `RenderResult` contract gives consumers type-safe access to all render output without needing to know pipeline internals.

**Positive:** The module/release metadata split prevents confusion between source identity and deployment identity, which matters when `--name` overrides or when the same module is released multiple times in an environment.

**Positive:** Fail-on-end aggregation lets users fix multiple issues in one iteration rather than discovering problems one at a time.

**Negative:** `RenderResult` has many fields — consumers must understand which fields are relevant to their use case and avoid drawing incorrect conclusions from fields that do not apply (for example, accessing release UUID when only a build was requested).

**Negative:** Receiver-method delegation requires the `ModuleRelease` to be constructed earlier in the pipeline, which constrains the ordering of initialization steps.

**Trade-off:** Representing resources as `*unstructured.Unstructured` is portable across platforms and avoids generated type coupling, but loses type-safe field access and requires callers to use string-keyed map traversal when inspecting specific fields.
