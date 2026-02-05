# Proposal: CLI Build Command

## Intent

Implement the `opm mod build` command that renders OPM modules into platform-specific manifests (primarily Kubernetes). This is the core rendering pipeline that transforms abstract module definitions into deployable resources.

## Scope

**In scope:**

- `opm mod build` command implementation
- Render pipeline (load → match → transform → output)
- Provider and transformer integration
- ModuleRelease construction from local module
- YAML/JSON output formats
- Split file output (`--split`). Splits into multiple files.
- Verbose/debug output modes
- Strict mode for trait enforcement

**Out of scope:**

- Deployment to cluster (see deploy-v1)
- New transformer definitions (see platform-adapter-spec)
- Bundle rendering (future)
- Remote module fetching (uses local path only)

## Dependencies

This change depends on and references:

- **platform-adapter-spec**: Provider and Transformer definitions (`#Provider`, `#Transformer`, `#TransformerContext`)
- **core**: Configuration loading from `~/.opm/config.cue`
- **deploy-v1**: Deployment lifecycle commands (apply, delete, status)
- **catalog/v0/core**: Module, ModuleRelease, Component definitions

## Approach

1. Implement render pipeline in Go with CUE SDK for module loading and transformer execution
2. Use parallel goroutines for transformer execution with isolated CUE contexts
3. Leverage existing `#Provider` and `#Transformer` definitions from platform specs
4. Follow fail-on-end pattern for error aggregation
5. Construct `#ModuleRelease` internally from local module path + values files

## Success Criteria

| ID | Criteria |
|----|----------|
| SC-001 | Module with 5 components renders in under 2 seconds (excluding network) |
| SC-002 | Transformer matching is deterministic - same input produces identical output |
| SC-003 | 100% of matched components produce valid Kubernetes resources |
| SC-004 | Error messages for unmatched components include actionable guidance |
| SC-005 | Verbose output shows transformer matching decisions |

## Risks & Edge Cases

| Case | Handling |
|------|----------|
| Two transformers with identical requirements | Error with "multiple exact transformer matches" |
| Transformer produces zero resources | Empty output is valid |
| Output fails Kubernetes schema validation | Warning logged; apply will fail server-side |
| Invalid values file | Fail with clear CUE validation error |
| Values file conflict | Return CUE's native unification error |

## Non-Goals

- **Deployment**: Handled by `deploy-v1` (`mod apply`, `mod delete`, `mod status`)
- **Transformer Authoring**: Handled by `platform-adapter-spec`
- **Bundle Rendering**: Deferred to future bundle-spec
- **Remote Module Resolution**: Build operates on local paths only
