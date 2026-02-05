# Proposal: Render Pipeline Interface

## Intent

Define a shared interface for the OPM render pipeline that transforms modules into platform-specific resources. This interface enables multiple CLI commands (build, apply, diff) to use the same rendering logic without duplication.

## SemVer Impact

**N/A** - This is an internal interface specification. No direct user-facing changes.

## Scope

**In scope:**

- `RenderResult` interface contract
- `Pipeline` interface for rendering operations
- Shared data types (Resource, ModuleMetadata, MatchPlan)
- Error types used across build and deploy operations
- Package organization for reusability

**Out of scope:**

- CLI command implementation (see build-v1)
- Kubernetes deployment operations (see deploy-v1)
- Transformer definitions (see platform-adapter-spec)
- Output formatting (YAML/JSON/split files)

## Dependencies

This change is a **foundational interface** that:

- **Is implemented by**: build-v1 (in `internal/build/`)
- **Is consumed by**: deploy-v1 (apply, diff, status commands)
- **References**: platform-adapter-spec (Transformer, Provider definitions)

## Approach

1. Define `RenderResult` as the contract between rendering and consumers
2. Define `Pipeline` interface that implementations must satisfy
3. Establish clear boundaries: rendering produces resources, consumers decide what to do with them
4. Design for future extensibility (Bundle support) without over-engineering

## Complexity Justification (Principle VII)

| Component | Justification |
|-----------|---------------|
| Shared interface | Required to avoid duplication between build, apply, diff commands |
| RenderResult struct | Single contract enables independent evolution of rendering vs consumption |
| MatchPlan in result | Enables verbose output and debugging without coupling to render internals |

## Success Criteria

| ID | Criteria |
|----|----------|
| SC-001 | build-v1 can implement Pipeline interface without modification to this spec |
| SC-002 | deploy-v1 can consume RenderResult without knowing rendering internals |
| SC-003 | Interface is stable enough that Bundle support can be added without breaking changes |

## Risks & Edge Cases

| Case | Handling |
|------|----------|
| Future Bundle rendering | RenderResult includes enough metadata to support multiple modules |
| Provider evolution | Interface doesn't expose provider internals; only consumes transformer output |
| Error aggregation | RenderResult.Errors enables fail-on-end pattern across the interface |
