# Proposal: CLI Deploy Commands

## Intent

Implement deployment lifecycle commands: `opm mod apply`, `opm mod delete`, `opm mod diff`, and `opm mod status`. These commands manage the full lifecycle of modules on Kubernetes clusters by consuming rendered resources from the build pipeline.

## SemVer Impact

**MINOR** - Adds new commands (`apply`, `delete`, `diff`, `status`) without breaking existing functionality.

## Scope

**In scope:**

- `opm mod apply` - Deploy module to cluster with server-side apply
- `opm mod delete` - Remove module from cluster with label discovery
- `opm mod diff` - Preview changes before deployment
- `opm mod status` - Check health of deployed resources
- Integration with build pipeline via `Pipeline.Render()` Go API
- Server-side apply with field ownership
- Resource ordering (weighted apply/delete)
- Resource labeling for module discovery
- Resource health checking

**Out of scope:**

- Rendering/transformation logic (see render-pipeline-v1, build-v1)
- Bundle deployment (future)
- Custom resource health definitions

## Dependencies

| Dependency | Relationship |
|------------|--------------|
| render-pipeline-v1 | Consumes: Pipeline interface, RenderResult, Resource types |
| build-v1 | Uses: Pipeline implementation via `build.NewPipeline()` |
| config-v1 | Uses: Configuration loading, Kubernetes settings |

## Approach

1. Commands call `build.NewPipeline().Render()` to get `RenderResult`
2. Pass `RenderResult.Resources` to `internal/kubernetes/` for operations
3. Use k8s.io/client-go for Kubernetes interactions
4. Implement server-side apply for idempotent operations
5. Use resource weights (from `pkg/weights/`) for ordered apply/delete
6. Label all resources for module-based discovery
7. Use dyff for colorized diff output

## Complexity Justification (Principle VII)

| Component | Justification |
|-----------|---------------|
| Server-side apply | Required for idempotent operations with field ownership tracking |
| Weighted ordering | Essential for CRD/workload dependencies |
| Health checking | Necessary for deployment validation |
| Resource labeling | Required for `mod delete` discovery without maintaining state |
| Integration via Go API | Enables type-safe resource handling and avoids subprocess overhead |

## Success Criteria

| ID | Criteria |
|----|----------|
| SC-001 | `mod apply` is fully idempotent - multiple runs produce no changes |
| SC-002 | `mod delete` removes all module resources without re-rendering |
| SC-003 | `mod diff` accurately reflects delta between local and cluster |
| SC-004 | `mod status` correctly reports Ready/NotReady within 60 seconds |
| SC-005 | New user can init, build, and apply a module in under 3 minutes |

## Risks & Edge Cases

| Case | Handling |
|------|----------|
| Cluster unreachable | Fail fast with clear connectivity error |
| RBAC permission denied | Pass through Kubernetes API error |
| Field conflict on apply | Log warning, take ownership (force apply) |
| Module deleted from disk | `mod delete` still works via label discovery |
| Partial render (some errors) | `mod diff` can still compare successful resources |
