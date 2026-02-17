## Context

The `opm mod tree` command fills the gap between flat resource lists (`mod status`) and structural understanding. Users need to see:

- How components group resources (OPM compositional layer)
- How Kubernetes controllers create child resources (K8s ownership layer)
- The health/status of each node in the hierarchy

The implementation leverages existing infrastructure:

- `kubernetes.DiscoverResources` for finding OPM-managed resources by labels
- Component labels (`component.opmodel.dev/name`) already stamped by CUE transformers
- Kubernetes `ownerReferences` for parent→child relationships
- `kubernetes.evaluateHealth` for status per resource
- Lipgloss color palette from `internal/output/styles.go`

**Constraints:**

- Must follow Separation of Concerns (Principle II): command layer orchestrates, business logic in `internal/kubernetes/tree.go`
- Must reuse existing flag patterns (`ReleaseSelectorFlags`, `K8sFlags`)
- Tree rendering must work without color (CI/non-TTY environments)
- JSON/YAML output must be machine-parseable and stable (no color codes)

**Stakeholders:**

- Platform operators debugging multi-component modules
- End-users verifying module deployment structure
- CI/CD pipelines consuming JSON output for validation

## Goals / Non-Goals

**Goals:**

1. Visualize OPM component hierarchy with colored tree rendering
2. Walk Kubernetes ownership chains (Deployment→ReplicaSet→Pod, StatefulSet→Pod)
3. Show health status and replica counts at each level
4. Support depth control (0=summary, 1=resources, 2=K8s children)
5. Support structured output (JSON, YAML) for machine consumption
6. Reuse existing discovery and health evaluation logic
7. Handle resources without component labels gracefully (group under `(no component)`)

**Non-Goals:**

1. Showing spec-level references (Ingress→Service, Service→Endpoints) — too fragile, too many edge cases
2. Real-time updates / watch mode — `mod status --watch` already provides this for flat view
3. Interactive selection/drilling — this is a read-only view, not a TUI
4. Custom tree formatting/themes — use existing lipgloss palette only
5. Modifying resources or triggering actions — tree is purely observational

## Decisions

### Decision 1: Two-phase data model (tree building → rendering)

**Choice:** Separate tree structure building from output formatting.

**Data Flow:**

```text
DiscoverResources (unstructured K8s objects)
         ↓
BuildTree (TreeResult with Components/Resources/Children)
         ↓
    ┌────┴────┐
    ↓         ↓
FormatTree  ToJSON/ToYAML
(colored)   (structured)
```

**Rationale:**

- Enables depth filtering during tree building (avoid fetching pods if depth=0)
- JSON/YAML serialization needs a stable struct, not terminal escape codes
- Testing is easier: test tree building separately from rendering
- Follows existing pattern in `mod status` (StatusResult → FormatStatus)

**Alternatives considered:**

- Stream rendering (build + format in one pass): Rejected — harder to test, can't support multiple output formats
- Render-only (work directly with unstructured): Rejected — mixes business logic with presentation

**Type Signature:**

```go
// TreeResult is the intermediate data structure
type TreeResult struct {
    Release    ReleaseInfo
    Components []Component
}

type Component struct {
    Name      string
    Resources []ResourceNode
}

type ResourceNode struct {
    Kind       string
    Name       string
    Namespace  string
    Status     healthStatus
    Replicas   string              // "3/3" for workloads, empty for others
    Children   []ResourceNode      // K8s-owned children (Pods, ReplicaSets)
}
```

---

### Decision 2: Component grouping strategy

**Choice:** Group by `component.opmodel.dev/name` label. Resources without the label go into a special `(no component)` group rendered last.

**Grouping Algorithm:**

```text
1. Discover all OPM resources (already filtered by managed-by + release selector)
2. Iterate resources, extract component label:
   - If present: add to componentMap[labelValue]
   - If absent: add to specialComponentMap["(no component)"]
3. Sort component names alphabetically (except "(no component)" always last)
4. Within each component, sort resources by:
   - Weight ascending (same order as apply)
   - Then by name (alphabetically)
```

**Rationale:**

- Component label is already present (stamped by CUE transformers)
- Alphabetical sorting is predictable and stable
- Weight-based sorting within components matches apply order (familiar mental model)
- `(no component)` group surfaces misconfigured resources (missing labels) without failing

**Alternatives considered:**

- Require all resources to have component labels: Rejected — too brittle, breaks on hand-edited resources
- Random/discovery order: Rejected — output would be unstable, hard to diff
- Group by Kind first, then component: Rejected — loses compositional view

**Edge Cases:**

- Empty component label (`component.opmodel.dev/name: ""`): Treated as missing, goes to `(no component)`
- Resources with same Kind+Name in different components: Allowed (different namespaces or controller-managed children)

---

### Decision 3: Ownership walking via ownerReferences

**Choice:** Walk `metadata.ownerReferences` to find child resources. Query cluster on-demand for children of workloads.

**Walking Algorithm:**

```text
For each OPM resource:
  If kind ∈ {Deployment, StatefulSet, DaemonSet, Job}:
    Query cluster for resources with ownerReferences pointing to this resource:
      - List Pods with labelSelector (controller-specific labels)
      - List ReplicaSets with ownerReference UID match (Deployment only)
    For each child:
      Recurse: extract status, replicas, children (ReplicaSet→Pod)
  Else:
    No children (passive resources don't own other resources)
```

**K8s API Calls:**

| Parent Kind | Child Query | Filter |
|-------------|-------------|--------|
| Deployment | ReplicaSets | `ownerReferences[].uid == deployment.uid` |
| Deployment | Pods (via RS) | `ownerReferences[].uid == replicaset.uid` |
| StatefulSet | Pods | `ownerReferences[].uid == statefulset.uid` |
| DaemonSet | Pods | `ownerReferences[].uid == daemonset.uid` |
| Job | Pods | `ownerReferences[].uid == job.uid` |

**Rationale:**

- OwnerReferences are Kubernetes-native, always accurate (controller-managed)
- Querying on-demand (not upfront) respects `--depth` flag (skip pod queries at depth=0/1)
- Deployment→ReplicaSet→Pod matches kubectl's mental model
- StatefulSet pods are direct children (no ReplicaSet layer)

**Alternatives considered:**

- Label selectors only: Rejected — doesn't work for all controllers, labels can be wrong
- Parse spec.selector and match manually: Rejected — fragile, controller-specific logic
- Include all ownerReferences regardless of kind: Rejected — creates noise (Endpoints owned by Service, etc.)

**Performance:** Each workload adds 1-2 cluster queries. For a module with 3 Deployments + 1 StatefulSet = ~6 queries. Acceptable for interactive use.

---

### Decision 4: Depth filtering at build time

**Choice:** Filter depth during tree building, not during rendering.

**Implementation:**

```go
func BuildTree(ctx, client, resources, depth) TreeResult {
    // depth=0: components only, skip iterating resources
    if depth == 0 {
        return componentSummary(resources)
    }
    
    // depth=1: resources, skip ownership walking
    if depth == 1 {
        return buildResourceTree(resources, walkChildren=false)
    }
    
    // depth=2: full tree with ownership walking
    return buildResourceTree(resources, walkChildren=true)
}
```

**Rationale:**

- Avoids unnecessary cluster queries for pods/replicasets when depth < 2
- Faster execution for summary views
- Clearer separation: BuildTree decides WHAT to fetch, FormatTree decides HOW to display

**Alternatives considered:**

- Render-time filtering (fetch everything, hide in output): Rejected — wastes cluster queries
- Separate functions per depth: Rejected — code duplication, hard to test

---

### Decision 5: Tree rendering with box-drawing characters

**Choice:** Use Unicode box-drawing for tree chrome. Colors via lipgloss. Graceful degradation for non-TTY.

**Box-Drawing Vocabulary:**

```text
├── (branch with continuation)
└── (final branch)
│   (vertical continuation line)
    (indent, no line)
```

**Rendering Rules:**

```text
Component (cyan, bold):
  ├── Resource (white, status colored)
  │   ├── Child (dim gray, status colored)
  │   └── Child
  └── Resource

Chrome (├── └── │) → colorDimGray (240)
Component name → ColorCyan (14) + Bold
OPM resource Kind/Name → default white
K8s child Kind/Name → colorDimGray (240)
Status → per healthStatus (Ready=green, NotReady=red, etc.)
```

**Status Color Mapping:**

| Status | Color | Constant |
|--------|-------|----------|
| Ready | Green | `colorGreen` (82) |
| Running | Green | `colorGreen` (82) |
| Bound | Green | `colorGreen` (82) |
| Complete | Green | `colorGreen` (82) |
| NotReady | Red | `colorRed` (196) |
| Pending | Yellow | `ColorYellow` (220) |
| CrashLoopBackOff | Red | `colorRed` (196) |
| Unknown | Yellow | `ColorYellow` (220) |

**Non-TTY Handling:**

```go
// In FormatTree:
if !lipgloss.IsTerminal() {
    // Render without colors, plain ASCII tree
    return formatPlainTree(tree)
}
return formatColoredTree(tree)
```

**Rationale:**

- Box-drawing is standard (kubectl uses it for CRD schemas, k9s for navigation)
- Lipgloss already provides `IsTerminal()` detection
- Graceful degradation ensures CI/pipeline compatibility

**Alternatives considered:**

- ASCII-only (`|`, `+--`): Rejected — less readable, looks dated
- Always colored: Rejected — breaks CI logs, JSON output would contain escape codes
- Custom theme system: Rejected — violates YAGNI, existing palette is sufficient

**Example Output (colored):**

```text
jellyfin-media (opmodel.dev/community/jellyfin@1.2.0)
│
├── server                              [cyan, bold]
│   ├── Deployment/jellyfin-server  Ready  3/3   [white + green status]
│   │   ├── ReplicaSet/jellyfin-server-abc  3 pods  [dim]
│   │   │   ├── Pod/jellyfin-server-abc-x1  Running  [dim + green]
│   │   │   ├── Pod/jellyfin-server-abc-x2  Running  [dim + green]
│   │   │   └── Pod/jellyfin-server-abc-x3  Running  [dim + green]
│   │   └── ReplicaSet/jellyfin-server-old  0 pods   [dim]
│   ├── Service/jellyfin-svc  Ready         [white + green]
│   └── ConfigMap/jellyfin-config  Ready    [white + green]
│
├── database                            [cyan, bold]
│   ├── StatefulSet/jellyfin-db  Ready  1/1  [white + green]
│   │   └── Pod/jellyfin-db-0  Running       [dim + green]
│   └── PVC/jellyfin-data  Bound  10Gi       [white + green]
│
└── ingress                             [cyan, bold]
    └── Ingress/jellyfin-ing  Ready         [white + green]
```

---

### Decision 6: Replica count extraction

**Choice:** Extract replica counts from workload status fields. Format as `current/desired`.

**Extraction Logic:**

```go
func getReplicaCount(resource *unstructured.Unstructured) string {
    kind := resource.GetKind()
    
    switch kind {
    case "Deployment", "StatefulSet", "DaemonSet":
        replicas, _ := unstructured.NestedInt64(resource.Object, "status", "replicas")
        ready, _ := unstructured.NestedInt64(resource.Object, "status", "readyReplicas")
        return fmt.Sprintf("%d/%d", ready, replicas)
    
    case "Job":
        succeeded, _ := unstructured.NestedInt64(resource.Object, "status", "succeeded")
        completions, _ := unstructured.NestedInt64(resource.Object, "spec", "completions")
        return fmt.Sprintf("%d/%d", succeeded, completions)
    
    case "ReplicaSet":
        replicas, _ := unstructured.NestedInt64(resource.Object, "status", "replicas")
        return fmt.Sprintf("%d pods", replicas)
    
    default:
        return "" // No replica concept
    }
}
```

**Field Paths:**

| Kind | Desired | Current | Field Path |
|------|---------|---------|------------|
| Deployment | `spec.replicas` | `status.readyReplicas` | Standard |
| StatefulSet | `spec.replicas` | `status.readyReplicas` | Standard |
| DaemonSet | (nodes) | `status.numberReady` | `status.desiredNumberScheduled` / `status.numberReady` |
| Job | `spec.completions` | `status.succeeded` | Standard |
| ReplicaSet | `spec.replicas` | `status.replicas` | Show as "N pods" |

**Rationale:**

- Uses Kubernetes standard status fields (stable API)
- Matches kubectl output format (`kubectl get deploy` shows READY column as `3/3`)
- Empty string for non-workload resources (clean output)

**Alternatives considered:**

- Show only current count: Rejected — missing context (is 1 pod desired or degraded?)
- Parse conditions for replica info: Rejected — replica fields are authoritative

---

### Decision 7: Structured output schema (JSON/YAML)

**Choice:** Define a stable JSON schema independent of tree rendering. Serialize TreeResult directly.

**Schema (simplified):**

```json
{
  "release": {
    "name": "jellyfin-media",
    "namespace": "media",
    "module": "opmodel.dev/community/jellyfin",
    "version": "1.2.0"
  },
  "components": [
    {
      "name": "server",
      "resourceCount": 3,
      "status": "Ready",
      "resources": [
        {
          "kind": "Deployment",
          "name": "jellyfin-server",
          "namespace": "media",
          "status": "Ready",
          "replicas": "3/3",
          "children": [
            {
              "kind": "ReplicaSet",
              "name": "jellyfin-server-abc",
              "children": [
                {"kind": "Pod", "name": "jellyfin-server-abc-x1", "status": "Running"},
                {"kind": "Pod", "name": "jellyfin-server-abc-x2", "status": "Running"},
                {"kind": "Pod", "name": "jellyfin-server-abc-x3", "status": "Running"}
              ]
            }
          ]
        }
      ]
    }
  ]
}
```

**Rationale:**

- Nested structure matches visual tree hierarchy
- `resourceCount` and component-level `status` enable summary views without parsing resources
- Recursive `children` array supports arbitrary depth
- No color codes, stable field names

**Alternatives considered:**

- Flat list with parent IDs: Rejected — harder to consume, loses tree semantics
- Separate `pods`, `replicasets` arrays: Rejected — breaks ownership hierarchy

---

### Decision 8: Error handling and edge cases

**Choice:** Graceful degradation. Partial data is better than total failure.

**Error Scenarios:**

| Scenario | Behavior |
|----------|----------|
| No resources found | Return `noResourcesFoundError` (same as mod status/delete) |
| Component label missing | Group under `(no component)` |
| ownerReferences invalid/orphaned | Skip child, log debug message |
| Pod query fails (RBAC) | Show resource without children, log warning |
| Health status unknown | Show `Unknown` status (yellow) |
| Replica fields missing | Show empty replicas string |

**Error Handling Pattern:**

```go
// In ownership walking:
pods, err := listPods(ctx, client, ownerUID)
if err != nil {
    output.Debug("failed to list pods for workload",
        "kind", resource.GetKind(),
        "name", resource.GetName(),
        "error", err,
    )
    return ResourceNode{...} // Return without children
}
```

**Rationale:**

- Partial tree is useful even if some children can't be fetched
- Debug logging preserves troubleshooting information
- Matches existing CLI pattern (mod status doesn't fail on partial data)

**Alternatives considered:**

- Fail entire command on any error: Rejected — too brittle, bad UX
- Silent failures: Rejected — hard to debug RBAC/cluster issues

---

## Risks / Trade-offs

### Risk 1: Cluster query performance for large modules

**Scenario:** Module with 20 Deployments × 3 ReplicaSets × 10 Pods = 600 pods. At depth=2, this triggers 60+ cluster queries.

**Mitigation:**

- Default depth=2 is opt-in for full detail; users can use `--depth 1` for faster summary
- Queries are namespace-scoped and filtered by ownerReference (indexed in K8s)
- Could add `--timeout` flag in future if needed
- Performance acceptable for typical modules (< 10 workloads)

**Trade-off:** Completeness vs. speed. We prioritize completeness (users who want tree view need the data). Fast path is `mod status`.

---

### Risk 2: Brittle ownership detection for custom controllers

**Scenario:** A CRD creates pods via a custom controller that doesn't set ownerReferences properly.

**Mitigation:**

- Tree only shows resources with proper ownerReferences (Kubernetes best practice)
- Custom resources without standard ownership won't show children — this is correct behavior (tree reflects K8s state, not CRD semantics)
- Documentation notes this limitation

**Trade-off:** Simplicity vs. universal coverage. We choose simplicity (standard ownerReferences only). Covering all custom controllers is impossible.

---

### Risk 3: Output instability across Kubernetes versions

**Scenario:** `status.readyReplicas` field renamed or moved in future K8s API version.

**Mitigation:**

- Use `unstructured.NestedInt64` with error handling (fails gracefully if field missing)
- Fields used (`status.replicas`, `status.readyReplicas`, etc.) are stable across K8s 1.19+
- Integration tests against multiple K8s versions (already done for other commands)

**Trade-off:** API stability vs. feature richness. We rely on stable status fields only.

---

### Risk 4: Tree rendering breaks in narrow terminals

**Scenario:** Terminal width < 80 columns, long resource names get truncated or wrapped awkwardly.

**Mitigation:**

- Use simple indentation (3 spaces per level), minimal horizontal space
- Truncate long names with `...` if terminal width < 80 (detect via `lipgloss.Width`)
- JSON/YAML output unaffected (no width constraints)

**Trade-off:** Readability in narrow terminals vs. full names. We prioritize full names (users can resize terminal or use `-o json`).

---

## Migration Plan

**Deployment Steps:**

1. Merge `internal/kubernetes/tree.go` (tree building logic) — safe, no command exposure
2. Merge `internal/cmd/mod_tree.go` (command registration) — exposes new command
3. Update docs and `opm mod --help` output
4. Announce in release notes (MINOR version bump)

**Rollback Strategy:**

- Remove command registration from `internal/cmd/mod.go` (1-line change)
- Deprecate in next MINOR, remove in next MAJOR if needed (unlikely)

**No data migration needed** — command is read-only, no state or config changes.

---

## Open Questions

### Q1: Should we show PVC → PV relationships?

**Context:** PersistentVolumeClaims reference PersistentVolumes, but PVs are cluster-scoped. Tree is currently namespace-scoped.

**Options:**

- A: Show PVC with PV name as annotation (e.g., `PVC/data → pv-abc123`)
- B: Query PV and show as child (breaks namespace scoping)
- C: Don't show PV (current decision)

**Recommendation:** Start with C (don't show PV). Re-evaluate if users request it. PV relationships are visible in `kubectl describe pvc`.

---

### Q2: Should depth > 2 be supported for future extensions?

**Context:** Depth=2 shows Deployment→ReplicaSet→Pod. Could depth=3 show Pod→Container or PVC→PV?

**Decision:** Reserve depth values > 2 for future use. Current implementation validates `--depth` ∈ {0,1,2}. Depth=3+ can be added later without breaking changes (MINOR bump).

---

### Q3: Should we add `--format wide` for additional columns (like kubectl)?

**Context:** kubectl supports `-o wide` for extra columns (IP, node, etc.).

**Options:**

- A: Add `--format wide` flag (sibling to `-o table|json|yaml`)
- B: Use `-o wide` (breaks convention — `-o` is for format type)
- C: Don't support wide (current decision)

**Recommendation:** Start with C. Tree view is already information-dense. If requested, add `--show-nodes` or similar specific flags rather than generic "wide".

---

## Implementation Phases

**Phase 1: Core tree building (MVP)**

- `internal/kubernetes/tree.go`:
  - `TreeResult`, `Component`, `ResourceNode` types
  - `BuildTree()` function with component grouping
  - Ownership walking for Deployment, StatefulSet
  - Depth filtering (0, 1, 2)
- Tests: component grouping, ownership walking (mocked client)

**Phase 2: Rendering and output**

- `internal/kubernetes/tree.go`:
  - `FormatTree()` with box-drawing and colors
  - `FormatTreeJSON()` and `FormatTreeYAML()`
  - TTY detection and plain output
- Tests: rendering with/without colors, JSON schema validation

**Phase 3: Command integration**

- `internal/cmd/mod_tree.go`:
  - Flag parsing (reuse `ReleaseSelectorFlags`, `K8sFlags`)
  - `--depth` flag with validation
  - Output format routing
  - Error handling and exit codes
- `internal/cmd/mod.go`: Register subcommand
- Tests: flag validation, integration test with kind cluster

**Phase 4: Polish**

- DaemonSet and Job ownership walking
- Edge case handling (missing labels, RBAC errors)
- Help text and examples
- Documentation updates

**Estimated LOC:**

- `internal/kubernetes/tree.go`: ~350 lines
- `internal/cmd/mod_tree.go`: ~150 lines
- Tests: ~300 lines
- **Total: ~800 lines** (vs. 400-500 estimated in proposal — includes tests)
