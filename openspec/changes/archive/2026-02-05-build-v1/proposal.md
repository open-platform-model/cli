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
- 6-phase render pipeline: load → release build → provider → match → transform → result
- Release building with `#config` pattern for concrete components
- Provider and transformer integration via CUE
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

## Key Design: #config Pattern

Modules use a schema/values separation:

```cue
// module.cue - Schema (constraints only)
#config: {
    web: { image: string, replicas: int & >=1 }
}
values: #config

// values.cue - Defaults
values: web: { image: "nginx:1.25", replicas: 2 }

// components.cue - References #config
#components: web: { spec: container: image: #config.web.image }
```

At build time, `ReleaseBuilder` executes `FillPath(#config, values)` to make all configuration references concrete before component extraction.

**Why this pattern?**

- The `#ModuleRelease` synthesis approach (wrapping modules at runtime) was complex and required importing core schemas
- The `#config` pattern keeps everything self-contained within the module
- CUE's `FillPath` is simple and explicit
- Users get schema validation via `values: #config` declaration

## Namespace Resolution

Namespace is resolved with this precedence (highest first):

1. `--namespace` flag
2. `module.metadata.defaultNamespace`

If neither is provided, build fails with actionable error message.

## Approach

1. Implement Pipeline interface in `internal/build/` package
2. Use 6-phase architecture: load → release build → provider → match → transform → result
3. Use `ReleaseBuilder` to inject values into `#config` and extract concrete components
4. Use CUE SDK for module loading and transformer execution
5. Use `FillPath` for `#component` and `#context` injection in transformers
6. Use parallel goroutines for transformer execution
7. Follow fail-on-end pattern for error aggregation
8. Separate output formatting into `internal/output/` (CLI-specific, not part of Pipeline)

## Success Criteria

| ID | Criteria |
|----|----------|
| SC-001 | Module with 5 components renders in under 2 seconds (excluding network) |
| SC-002 | Transformer matching is deterministic - same input produces identical output |
| SC-003 | 100% of matched components produce valid Kubernetes resources |
| SC-004 | Error messages for unmatched components include actionable guidance |
| SC-005 | Verbose output shows transformer matching decisions |
| SC-006 | deploy-v1 can use Pipeline.Render() without modification to this implementation |
| SC-007 | Modules using #config pattern build successfully with concrete components |

## Complexity Justification (Principle VII)

| Component | Justification |
|-----------|---------------|
| ReleaseBuilder | Separates concrete component extraction from raw module loading; enables clear error messages for non-concrete values |
| Parallel execution | Required for performance with many components |
| Separate output package | CLI-specific concerns (YAML format, split files) don't belong in reusable Pipeline |
| FillPath injection | Clean way to inject #component and #context without complex CUE expression building |

## Risks & Edge Cases

| Case | Handling |
|------|----------|
| Two transformers match same component | Both execute, both produce resources |
| Transformer produces zero resources | Empty resource list is valid |
| Output fails Kubernetes schema validation | Warning logged; apply will fail server-side |
| Invalid values file | Fail with clear CUE validation error |
| Values file conflict | Return CUE's native unification error |
| Non-concrete component after release building | Fail with ReleaseValidationError including component name |
| Module missing `values` field | Fail with descriptive error about #config pattern |
| Module missing `#components` field | Fail with descriptive error |
| No namespace and no defaultNamespace | Fail with actionable error suggesting --namespace or defaultNamespace |

## Non-Goals

- **Pipeline interface design**: Defined in render-pipeline-v1
- **Deployment operations**: Handled by deploy-v1
- **Transformer authoring**: Handled by platform-adapter-spec
- **Bundle rendering**: Deferred to future bundle-spec
- **Remote module resolution**: Build operates on local paths only
- **#ModuleRelease synthesis**: Superseded by #config pattern
