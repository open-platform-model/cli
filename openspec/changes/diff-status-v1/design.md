# Design: mod diff & mod status

## Context

The CLI currently has `mod diff` and `mod status` registered as stubs in `internal/cmd/mod_stubs.go`. Both print "not yet implemented". The `internal/kubernetes/` package does not exist yet — it will be introduced by `deploy-v1` (which owns apply, delete, client init, discovery, and resource labeling). This change implements the two read-only inspection commands on top of that shared infrastructure.

Key constraints:

- Commands live as flat files in `internal/cmd/` with `mod_` prefix (e.g., `mod_diff.go`, `mod_status.go`)
- `internal/kubernetes/` will be created by deploy-v1; this change adds `diff.go`, `health.go`, and `status.go` to that package
- The build pipeline (`build.NewPipeline().Render()`) provides rendered resources — diff needs this; status does not
- `homeport/dyff` is not yet a dependency and must be added

## Goals / Non-Goals

**Goals:**

- Implement `opm mod diff` that compares locally rendered resources against live cluster state
- Implement `opm mod status` that discovers deployed resources by OPM labels and reports health
- Produce clear, colorized, human-readable output for both commands
- Support structured output formats (table, yaml, json) for `mod status`
- Support `--watch` mode for continuous status monitoring

**Non-Goals:**

- Apply or delete operations (deploy-v1)
- Kubernetes client initialization or label injection (deploy-v1)
- Custom health check definitions (future)
- Drift detection or alerting (future)
- Bundle-level diff or status (future)

## Decisions

### 1. Diff Uses dyff for Semantic Comparison

**Decision**: Use `homeport/dyff` for YAML-aware semantic diffing instead of text-based diff.

**Alternatives considered**:

- **Text diff (go-diff)**: Simple but produces noisy output — field reordering shows as changes, no type awareness
- **Custom diff**: Full control but high implementation cost for marginal benefit
- **dyff**: Purpose-built for Kubernetes YAML, handles field ordering, produces colorized human-readable output

**Rationale**: dyff already understands Kubernetes resource structure, handles field ordering correctly, and produces output familiar to Kubernetes users. It's a well-maintained library with minimal dependency footprint.

### 2. Diff Compares Per-Resource, Not Whole-Manifest

**Decision**: Fetch and compare each resource individually rather than dumping the entire namespace.

**Rationale**:

- Precise: only compares resources the module owns
- Handles mixed namespaces correctly
- Can show "new" (not yet deployed) and "orphaned" (deployed but no longer rendered) resources separately
- Requires cluster access per resource, but resource count per module is typically small (<50)

### 3. Status Uses Label Discovery (Not Re-Render)

**Decision**: `mod status` discovers resources via OPM labels, not by re-rendering the module.

**Rationale**:

- Works even if module source has changed or been deleted
- Faster — no CUE evaluation needed
- Shows actual deployed state, not intended state
- Consistent with `mod delete` approach from deploy-v1

### 4. Health Evaluation by Resource Category

**Decision**: Evaluate health differently based on resource kind.

| Category | Resources | Health Criteria |
|----------|-----------|-----------------|
| Workloads | Deployment, StatefulSet, DaemonSet | `Ready` condition is True |
| Jobs | Job | `Complete` condition is True |
| CronJobs | CronJob | Always healthy (scheduled) |
| Passive | ConfigMap, Secret, Service, PVC | Healthy on creation |
| Custom | CRD instances | `Ready` condition if present, else passive |

**Rationale**: A single health check doesn't work across resource types. Kubernetes itself uses different readiness signals per resource kind. This categorization covers the common cases without over-engineering.

### 5. Watch Mode Uses Polling, Not Informers

**Decision**: `mod status --watch` polls on an interval rather than using client-go informers.

**Alternatives considered**:

- **Informers/watches**: Real-time but complex setup, requires managing cache, stop channels, and reconnection
- **Polling (2s interval)**: Simpler, good enough for interactive CLI use, no long-lived connections

**Rationale**: For an interactive CLI command, 2-second polling is indistinguishable from real-time. Informers add significant complexity for a feature that will typically run for seconds to minutes. The polling approach is easier to test and debug.

### 6. Diff Handles Three Resource States

**Decision**: Diff output categorizes resources into three groups.

| State | Meaning | Display |
|-------|---------|---------|
| Modified | Exists locally and on cluster, content differs | Show dyff output |
| Added | Exists locally, not on cluster | Show as "new resource" |
| Orphaned | Exists on cluster (by label), not in local render | Show as "will be removed on next apply" |

**Rationale**: Users need to understand the full picture before applying. Just showing modifications misses new resources and leaves users unaware of orphans from previous applies.

## Risks / Trade-offs

**[Risk] deploy-v1 not implemented yet** → Both commands depend on `internal/kubernetes/client.go` and `internal/kubernetes/discovery.go` from deploy-v1. Mitigation: Define clear interfaces for the client and discovery dependencies. Implement against those interfaces so this change can proceed in parallel. Integration happens when deploy-v1 lands.

**[Risk] dyff API stability** → dyff is an external dependency. Mitigation: Wrap dyff behind a thin adapter in `internal/kubernetes/diff.go` so the library can be swapped without changing command code.

**[Risk] Large modules produce verbose diff output** → A module with many resources could produce overwhelming diff output. Mitigation: Add a summary line at the top ("N resources modified, M added, K orphaned") so users can assess scope before reading details. Consider `--summary` flag in future if needed.

**[Trade-off] Polling vs informers for watch** → Polling uses more API calls but is dramatically simpler. Acceptable for a CLI tool; could be revisited if used programmatically.

**[Trade-off] Per-resource fetching for diff** → Makes N API calls instead of 1 list call. Acceptable because module resource counts are typically small and this gives precise per-resource comparison.
