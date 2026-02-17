## Why

### The Problem

The `opm mod status` command shows a flat list of resources with health status, but provides no insight into:

1. **Module composition**: Which resources belong to which component?
2. **Structural hierarchy**: How are components organized within the module?
3. **Kubernetes ownership chains**: What pods/replicasets exist under workloads?
4. **Replica distribution**: Which pods are running? How many are ready?

When debugging or understanding a deployed module, users must mentally reconstruct the tree by:

- Running `kubectl get all` and cross-referencing labels
- Using `kubectl describe` to find ownerReferences
- Manually grouping resources by component labels
- Checking pod status separately for each workload

This is tedious and error-prone, especially for modules with multiple components.

### What Users Actually Need

Platform operators and end-users need to **see the module's structure at a glance**:

- "What components does this module have?"
- "What resources belong to each component?"
- "Which pods are running under this Deployment?"
- "Is the database StatefulSet managing its pods correctly?"

A hierarchical tree view answers all these questions in one command. It's the compositional complement to `mod status` (which shows flat health) and `mod diff` (which shows configuration drift).

### Why Now

The infrastructure is already in place:

- OPM labels (`component.opmodel.dev/name`) are stamped by CUE transformers
- `kubernetes.DiscoverResources` finds all module resources by labels
- Kubernetes ownerReferences provide the parent-child relationships
- Lipgloss provides terminal colors for readability

We're just connecting the dots with a new command that leverages existing data.

## What Changes

### New Command: `opm mod tree`

A new command that visualizes the compositional and ownership hierarchy of a deployed module.

**Basic Usage:**

```bash
opm mod tree --release-name jellyfin -n media
```

**Output (default depth=2):**

```
jellyfin-media (opmodel.dev/community/jellyfin@1.2.0)
│
├── server
│   ├── Deployment/jellyfin-server          Ready  3/3
│   │   ├── ReplicaSet/jellyfin-server-abc  3 pods
│   │   │   ├── Pod/jellyfin-server-abc-x1  Running
│   │   │   ├── Pod/jellyfin-server-abc-x2  Running
│   │   │   └── Pod/jellyfin-server-abc-x3  Running
│   │   └── ReplicaSet/jellyfin-server-old  0 pods
│   ├── Service/jellyfin-svc                Ready
│   └── ConfigMap/jellyfin-config           Ready
│
├── database
│   ├── StatefulSet/jellyfin-db             Ready  1/1
│   │   └── Pod/jellyfin-db-0               Running
│   └── PVC/jellyfin-data                   Bound  10Gi
│
└── ingress
    └── Ingress/jellyfin-ing                Ready
```

### Component Grouping

Resources are grouped by the `component.opmodel.dev/name` label (already stamped by CUE transformers). Resources without a component label are grouped under a special `(no component)` section at the end.

### Kubernetes Ownership Walking

For workload resources (Deployment, StatefulSet, DaemonSet, Job), the command:

1. Walks ownerReferences to find child ReplicaSets and Pods
2. Shows replica counts (e.g., `3/3` for Deployment ready replicas)
3. Shows pod status (Running, Pending, CrashLoopBackOff, etc.)
4. Displays old ReplicaSets with 0 replicas (common after deployments)

For StatefulSets:

- Shows managed Pods directly (no ReplicaSet layer)
- Shows VolumeClaimTemplates → PVC relationships

For Services:

- Optionally shows Endpoints with address count (future enhancement, not in v1)

### Depth Control

The `--depth` flag controls tree depth:

| Depth | What's Shown | Example Use Case |
|-------|--------------|------------------|
| 0 | Components only with resource counts | Quick module overview |
| 1 | Components + OPM-managed resources | See module structure without K8s noise |
| 2 (default) | + K8s-owned children (Pods, ReplicaSets) | Full hierarchy for debugging |

**Example at depth=0:**

```
jellyfin-media (opmodel.dev/community/jellyfin@1.2.0)
├── server      3 resources  Ready
├── database    2 resources  Ready
└── ingress     1 resource   Ready
```

**Example at depth=1:**

```
jellyfin-media (opmodel.dev/community/jellyfin@1.2.0)
├── server
│   ├── Deployment/jellyfin-server   Ready  3/3
│   ├── Service/jellyfin-svc         Ready
│   └── ConfigMap/jellyfin-config    Ready
├── database
│   ├── StatefulSet/jellyfin-db      Ready  1/1
│   └── PVC/jellyfin-data            Bound  10Gi
└── ingress
    └── Ingress/jellyfin-ing         Ready
```

### Color Scheme

Using the existing lipgloss color palette from `internal/output/styles.go`:

| Element | Color | Lipgloss Constant | Rationale |
|---------|-------|-------------------|-----------|
| Release header | White, bold | (default) | Root emphasis |
| Component names | Cyan, bold | `ColorCyan` (14) | Matches existing noun styling |
| OPM-managed resources | Default white | (default) | Primary content |
| K8s-owned children | Dim gray | `colorDimGray` (240) | Secondary detail |
| Status: Ready/Running/Bound | Green | `colorGreen` (82) | Healthy state |
| Status: NotReady/CrashLoop/Pending | Red | `colorRed` (196) | Needs attention |
| Status: Complete | Green | `colorGreen` (82) | Successful termination |
| Status: Unknown | Yellow | `ColorYellow` (220) | Unclear state |
| Tree chrome (├── └── │) | Dim gray | `colorDimGray` (240) | Structure, not content |

### Release Selector Flags

Reuses existing `cmdutil.ReleaseSelectorFlags` and `cmdutil.K8sFlags`:

**Required (one of):**

- `--release-name <name>` — Select by release name
- `--release-id <uuid>` — Select by release UUID

**Required:**

- `-n, --namespace <ns>` — Target namespace

**Optional:**

- `--kubeconfig <path>` — Kubernetes config file
- `--context <ctx>` — Kubernetes context
- `--depth <0-2>` — Tree depth (default: 2)
- `-o, --output <format>` — Output format: `table` (default), `json`, `yaml`

### Structured Output Formats

**JSON** (`-o json`):

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
      "resources": [
        {
          "kind": "Deployment",
          "name": "jellyfin-server",
          "status": "Ready",
          "replicas": "3/3",
          "children": [
            {
              "kind": "ReplicaSet",
              "name": "jellyfin-server-abc",
              "pods": [
                {"name": "jellyfin-server-abc-x1", "status": "Running"},
                {"name": "jellyfin-server-abc-x2", "status": "Running"},
                {"name": "jellyfin-server-abc-x3", "status": "Running"}
              ]
            }
          ]
        }
      ]
    }
  ]
}
```

**YAML** (`-o yaml`):
Similar structure to JSON, formatted as YAML.

### Error Handling

| Scenario | Behavior | Exit Code |
|----------|----------|-----------|
| No resources found | Error: "no resources found for release ..." | 3 |
| Invalid flags (both --release-name and --release-id) | Error: "mutually exclusive" | 1 |
| Missing required flag | Error: usage message | 1 |
| Cluster unreachable | Error: "failed to connect to cluster" | 3 |
| Success | Tree output | 0 |

### Examples

**Basic tree:**

```bash
opm mod tree --release-name my-app -n production
```

**Component summary only:**

```bash
opm mod tree --release-name my-app -n production --depth 0
```

**Without K8s-owned children:**

```bash
opm mod tree --release-name my-app -n production --depth 1
```

**JSON output:**

```bash
opm mod tree --release-name my-app -n production -o json
```

**Using release ID instead of name:**

```bash
opm mod tree --release-id a1b2c3d4-e5f6-7890-abcd-ef1234567890 -n production
```

## Capabilities

### New Capabilities

- `mod-tree`: The `opm mod tree` command — component-grouped hierarchical tree view of deployed module resources with Kubernetes ownership walking, depth control, colored terminal output, and structured output formats (JSON/YAML)

### Modified Capabilities

_(none — this is a new command that reuses existing resource discovery, label selectors, and K8s client infrastructure without changing their behavior)_

## Impact

### New Files

- `internal/cmd/mod_tree.go` — Command implementation, flag parsing, orchestration
- `internal/cmd/mod_tree_test.go` — Command flag validation tests
- `internal/kubernetes/tree.go` — Tree building logic, ownership walking, formatting
- `internal/kubernetes/tree_test.go` — Tree building and formatting tests

### Modified Files

- `internal/cmd/mod.go` — Register `tree` subcommand via `NewModTreeCmd()`

### Dependencies

**Existing (reused):**

- `kubernetes.DiscoverResources` — Finds OPM-managed resources by labels
- `kubernetes.Client` — K8s API access
- `cmdutil.ReleaseSelectorFlags` — `--release-name`/`--release-id` validation
- `cmdutil.K8sFlags` — `--kubeconfig`/`--context` flags
- `output.ColorCyan`, `output.colorGreen`, etc. — Lipgloss color palette
- `k8s.io/apimachinery/pkg/apis/meta/v1/unstructured` — Resource access
- `k8s.io/client-go/dynamic` — Dynamic K8s queries

**New patterns:**

- Walking `ownerReferences` arrays to find child resources
- Grouping resources by `component.opmodel.dev/name` label
- Recursive tree rendering with indentation and box-drawing characters

### Package Organization

Following existing patterns:

- **Command layer** (`internal/cmd/mod_tree.go`): Flag parsing, config resolution, output formatting
- **Business logic** (`internal/kubernetes/tree.go`): Tree building, ownership walking, component grouping
- **Shared utilities** (`internal/output/`): Color styles (already exist, reused)

### SemVer Classification

**MINOR** (0.x.0) — New command, no breaking changes, no existing behavior modified

### Complexity Justification (Principle VII)

**New flags:**

- `--depth` — Necessary for controlling output verbosity. Default of 2 (full tree) provides maximum insight, while 0 and 1 offer lighter views for quick checks.
- No other new flags — reuses existing release selector and K8s flags.

**New command justification:**

- Fills a gap that no combination of existing commands covers
- `mod status` shows flat health (no hierarchy)
- `mod diff` shows configuration drift (no runtime state)
- `kubectl get all` shows resources (no component grouping, no OPM awareness)
- The tree view is the natural visual complement to these commands

**Complexity added:**

- Ownership walking adds ~100-150 LOC in `internal/kubernetes/tree.go`
- Tree rendering adds ~100 LOC for formatting and box-drawing
- Total estimated: ~400-500 LOC across 4 files
- This is justified by the significant UX improvement for module inspection

### Testing Strategy

- Unit tests for component grouping logic
- Unit tests for ownership walking (mocked K8s client)
- Unit tests for tree rendering and color application
- Unit tests for depth filtering (0, 1, 2)
- Unit tests for JSON/YAML output formats
- Flag validation tests (mutual exclusivity, required flags)
- Integration tests against kind cluster (reuse existing test infrastructure)

### Documentation Updates

- Add `opm mod tree --help` text with examples
- Update CLI README with new command
- Add tree command to command reference docs
