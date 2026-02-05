# Design: CLI Deploy Commands

## Architecture

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                              CLI Commands                                │
├─────────────────┬─────────────────┬─────────────────┬───────────────────┤
│   mod apply     │   mod delete    │    mod diff     │    mod status     │
└────────┬────────┴────────┬────────┴────────┬────────┴─────────┬─────────┘
         │                 │                 │                  │
         ▼                 │                 ▼                  │
┌─────────────────────────────────────────────────────────────────────────┐
│                          internal/build/                                 │
│                                                                          │
│   build.NewPipeline(cfg).Render(ctx, opts) ──▶ *RenderResult            │
│                                                      │                   │
│                                           ┌──────────┴──────────┐       │
│                                           ▼                     ▼       │
│                                    Resources []*Resource    Errors      │
└─────────────────────────────────────────────────────────────────────────┘
         │                 │                 │                  │
         ▼                 ▼                 ▼                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         internal/kubernetes/                             │
├─────────────────┬─────────────────┬─────────────────┬───────────────────┤
│   apply.go      │   delete.go     │    diff.go      │    health.go      │
│                 │   discovery.go  │                 │    status.go      │
└────────┬────────┴────────┬────────┴────────┬────────┴─────────┬─────────┘
         │                 │                 │                  │
         ▼                 ▼                 ▼                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           k8s.io/client-go                               │
│                    (Server-Side Apply, Discovery)                        │
└─────────────────────────────────────────────────────────────────────────┘
```

## Key Design Decisions

### 1. Integration via Go API (Not Subprocess)

**Decision**: Commands call `build.NewPipeline().Render()` directly, not `opm mod build` as subprocess.

**Rationale**:
- Type-safe access to `RenderResult.Resources`
- No serialization/deserialization overhead
- Commands can inspect `RenderResult.Errors` for partial processing
- Enables dry-run without file I/O

```go
// internal/cmd/mod/apply.go
func runApply(cmd *cobra.Command, args []string) error {
    pipeline := build.NewPipeline(cfg)
    
    result, err := pipeline.Render(ctx, build.RenderOptions{
        ModulePath: modulePath,
        Values:     valuesFiles,
        Namespace:  namespace,
    })
    if err != nil {
        return err // Fatal error (module not found, etc.)
    }
    
    if result.HasErrors() {
        return &RenderFailedError{Errors: result.Errors}
    }
    
    // Pass resources to Kubernetes client
    return kubernetes.Apply(ctx, result.Resources, kubernetes.ApplyOptions{
        DryRun: dryRun,
        Wait:   wait,
    })
}
```

### 2. Server-Side Apply with Force Conflicts

**Decision**: Use server-side apply with `force=true` for field conflicts.

**Rationale**:
- OPM takes ownership of all fields it manages
- Prevents drift from manual kubectl edits
- Aligns with GitOps: declared state is desired state
- Logs warning on conflict but proceeds

### 3. Label-Based Resource Discovery

**Decision**: All resources receive OPM labels. `mod delete` discovers via labels, not re-rendering.

**Rationale**:
- Delete works even if module source is deleted
- No re-render needed (faster, simpler)
- Survives transformer changes between versions

**Labels applied**:
```yaml
metadata:
  labels:
    app.kubernetes.io/managed-by: open-platform-model
    module.opmodel.dev/name: <module-name>
    module.opmodel.dev/namespace: <target-namespace>
    module.opmodel.dev/version: <module-version>
    component.opmodel.dev/name: <component-name>
```

### 4. Weighted Resource Ordering

**Decision**: Resources are applied/deleted in weight order (from `pkg/weights/`).

**Rationale**:
- CRDs must exist before custom resources
- Namespaces must exist before namespaced resources
- Webhooks applied last to avoid blocking own resources

Note: Resources are already ordered in `RenderResult.Resources` by build-v1.

### 5. Health Evaluation by Category

**Decision**: Different health logic per resource category.

| Category | Resources | Health Criteria |
|----------|-----------|-----------------|
| Workloads | Deployment, StatefulSet, DaemonSet | `Ready` condition is True |
| Jobs | Job, CronJob | `Complete` condition is True |
| Passive | ConfigMap, Secret, Service | Healthy on creation |
| Custom | CRD instances | `Ready` if present, else passive |

## Data Flow

### Apply Flow

```text
RenderOptions ──▶ Pipeline.Render() ──▶ RenderResult
                                              │
                                              ▼
                                       Resources []*Resource
                                              │
                                              ▼
                                    ┌─────────────────────┐
                                    │ Add OPM Labels      │
                                    │ (if not present)    │
                                    └─────────────────────┘
                                              │
                                              ▼
                                    ┌─────────────────────┐
                                    │ Server-Side Apply   │
                                    │ (in weight order)   │
                                    └─────────────────────┘
                                              │
                                              ▼
                                    ┌─────────────────────┐
                                    │ Wait for Health     │
                                    │ (if --wait)         │
                                    └─────────────────────┘
```

### Delete Flow

```text
--name, --namespace ──▶ Build Label Selector
                              │
                              ▼
                    ┌─────────────────────┐
                    │ Discover Resources  │
                    │ (via labels)        │
                    └─────────────────────┘
                              │
                              ▼
                    ┌─────────────────────┐
                    │ Sort by Weight      │
                    │ (descending)        │
                    └─────────────────────┘
                              │
                              ▼
                    ┌─────────────────────┐
                    │ Delete Each         │
                    │ (with finalizers)   │
                    └─────────────────────┘
                              │
                              ▼
                    ┌─────────────────────┐
                    │ Wait for Gone       │
                    │ (if --wait)         │
                    └─────────────────────┘
```

### Diff Flow

```text
RenderOptions ──▶ Pipeline.Render() ──▶ RenderResult
                                              │
                                              ▼
                                       Resources []*Resource
                                              │
                                              ▼
                    ┌─────────────────────────────────────────┐
                    │ For each resource:                      │
                    │   1. Fetch live state from cluster      │
                    │   2. Compare rendered vs live           │
                    │   3. Collect diffs                      │
                    └─────────────────────────────────────────┘
                                              │
                                              ▼
                    ┌─────────────────────────────────────────┐
                    │ Output with dyff                        │
                    │ (colorized semantic diff)               │
                    └─────────────────────────────────────────┘
```

### Status Flow

```text
--name, --namespace ──▶ Build Label Selector
                              │
                              ▼
                    ┌─────────────────────┐
                    │ Discover Resources  │
                    │ (via labels)        │
                    └─────────────────────┘
                              │
                              ▼
                    ┌─────────────────────┐
                    │ Evaluate Health     │
                    │ (per category)      │
                    └─────────────────────┘
                              │
                              ▼
                    ┌─────────────────────┐
                    │ Output Table        │
                    │ or JSON/YAML        │
                    └─────────────────────┘
```

## Resource Weights

Defined in `pkg/weights/weights.go` (shared with build-v1).

| Weight | Resources |
|--------|-----------|
| -100 | CustomResourceDefinition |
| 0 | Namespace |
| 5 | ClusterRole, ClusterRoleBinding, ResourceQuota, LimitRange |
| 10 | ServiceAccount, Role, RoleBinding |
| 15 | Secret, ConfigMap |
| 20 | StorageClass, PersistentVolume, PersistentVolumeClaim |
| 50 | Service |
| 100 | Deployment, StatefulSet, DaemonSet, ReplicaSet |
| 110 | Job, CronJob |
| 150 | Ingress, NetworkPolicy |
| 200 | HorizontalPodAutoscaler |
| 500 | ValidatingWebhookConfiguration, MutatingWebhookConfiguration |

## Ownership Boundaries

| Owner | Responsibility |
|-------|----------------|
| render-pipeline-v1 | Interface definitions |
| build-v1 | Pipeline implementation, rendering |
| deploy-v1 (this) | Kubernetes operations, health checking |
| Kubernetes API | Scheduling, actual state, admission control |
| User | kubeconfig, RBAC permissions |

## File Changes

| File | Purpose |
|------|---------|
| `internal/kubernetes/client.go` | K8s client initialization |
| `internal/kubernetes/apply.go` | Server-side apply logic |
| `internal/kubernetes/delete.go` | Deletion with label discovery |
| `internal/kubernetes/diff.go` | Diff with dyff integration |
| `internal/kubernetes/health.go` | Resource health evaluation |
| `internal/kubernetes/discovery.go` | Label-based resource discovery |
| `internal/kubernetes/status.go` | Status table output |
| `internal/cmd/mod/apply.go` | Apply command |
| `internal/cmd/mod/delete.go` | Delete command |
| `internal/cmd/mod/diff.go` | Diff command |
| `internal/cmd/mod/status.go` | Status command |
