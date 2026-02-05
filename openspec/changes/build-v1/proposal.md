# Proposal: CLI Build Command

## Intent

Implement the `opm mod build` command and the `internal/build` package that renders OPM modules into platform-specific manifests (primarily Kubernetes). This change implements the Pipeline interface defined in render-pipeline-v1.

## SemVer Impact

**MINOR** - Adds new command `mod build` without breaking existing functionality.

## Scope

**In scope:**

- `opm mod build` command implementation
- `internal/build/` package implementing Pipeline interface
- `internal/output/` package for manifest formatting
- Render pipeline phases: load → match → transform → result
- Provider and transformer integration via CUE
- ModuleRelease construction from local module
- YAML/JSON output formats
- Split file output (`--split`)
- Verbose output modes

**Out of scope:**

- Pipeline interface definition (see render-pipeline-v1)
- Deployment to cluster (see deploy-v1)
- New transformer definitions (see platform-adapter-spec)
- Bundle rendering (future)
- Remote module fetching (uses local path only)

## Dependencies

| Dependency | Relationship |
|------------|--------------|
| render-pipeline-v1 | Implements: Pipeline interface, RenderResult |
| platform-adapter-spec | References: #Provider, #Transformer definitions |
| config-v1 | Uses: Configuration loading, provider resolution |
| deploy-v1 | Consumed by: apply, diff, status commands |

## Approach

1. Implement Pipeline interface in `internal/build/` package
2. Use CUE SDK for module loading and transformer execution
3. Use parallel goroutines for transformer execution with isolated CUE contexts
4. Leverage existing `#Provider` and `#Transformer` definitions from platform specs
5. Follow fail-on-end pattern for error aggregation
6. Separate output formatting into `internal/output/` (CLI-specific, not part of Pipeline)

## Success Criteria

| ID | Criteria |
|----|----------|
| SC-001 | Module with 5 components renders in under 2 seconds (excluding network) |
| SC-002 | Transformer matching is deterministic - same input produces identical output |
| SC-003 | 100% of matched components produce valid Kubernetes resources |
| SC-004 | Error messages for unmatched components include actionable guidance |
| SC-005 | Verbose output shows transformer matching decisions |
| SC-006 | deploy-v1 can use Pipeline.Render() without modification to this implementation |

## Complexity Justification (Principle VII)

| Component | Justification |
|-----------|---------------|
| Parallel execution | Required for performance with many components; isolated CUE contexts prevent memory issues |
| Separate output package | CLI-specific concerns (YAML format, split files) don't belong in reusable Pipeline |
| CUE-based matching | Matching logic in CUE ensures consistency with transformer definitions |

## Risks & Edge Cases

| Case | Handling |
|------|----------|
| Two transformers with identical requirements | Error with "multiple exact transformer matches" |
| Transformer produces zero resources | Empty resource list is valid |
| Output fails Kubernetes schema validation | Warning logged; apply will fail server-side |
| Invalid values file | Fail with clear CUE validation error |
| Values file conflict | Return CUE's native unification error |

## Non-Goals

- **Pipeline interface design**: Defined in render-pipeline-v1
- **Deployment operations**: Handled by deploy-v1
- **Transformer authoring**: Handled by platform-adapter-spec
- **Bundle rendering**: Deferred to future bundle-spec
- **Remote module resolution**: Build operates on local paths only
