## Context

The OPM CLI currently has `mod status` for checking resource health and `mod diff` for detecting configuration drift, but no way to see *why* resources are unhealthy. When a Deployment reports `NotReady`, the user must leave OPM and manually run `kubectl describe` and `kubectl get events` across multiple resource types. The most useful diagnostic events (OOMKilled, ImagePullBackOff, scheduling failures) live on Kubernetes-owned children (Pods, ReplicaSets) that aren't OPM-labeled and aren't visible through OPM's current tooling.

### Current architecture

The existing command pattern follows a consistent structure:

```text
cmd/mod_<command>.go          cobra command, flag parsing, output formatting
  │
  ├─ cmdutil.ReleaseSelectorFlags    shared --release-name/--release-id/-n
  ├─ cmdutil.K8sFlags                shared --kubeconfig/--context
  │
  └─ kubernetes/<operation>.go       business logic
       │
       ├─ DiscoverResources()        label-based resource discovery
       ├─ <operation>()              core logic (e.g., GetModuleStatus, Delete)
       └─ Format<operation>()        output formatting (table/json/yaml)
```

The events command fits cleanly into this pattern. The key new capability is **downward ownerReference traversal** — walking from OPM-managed parents to Kubernetes-owned children to collect their UIDs for event filtering.

### Kubernetes Event API

Kubernetes events are `v1.Event` objects in the same namespace as their `involvedObject`. Key fields:

```text
v1.Event
├── involvedObject.uid        ← matches resources we care about
├── involvedObject.kind       ← e.g., "Pod", "Deployment"
├── involvedObject.name       ← e.g., "jellyfin-server-abc12-x1"
├── type                      ← "Normal" or "Warning"
├── reason                    ← e.g., "OOMKilled", "Scheduled", "Pulling"
├── message                   ← human-readable description
├── lastTimestamp             ← last occurrence
├── count                     ← how many times this event fired
└── firstTimestamp            ← first occurrence
```

Events are queried via `Clientset.CoreV1().Events(namespace).List()` with a `fieldSelector` on `involvedObject.uid` or fetched in bulk and filtered client-side.

## Goals / Non-Goals

**Goals:**

- Aggregate Kubernetes events from all resources belonging to an OPM release into a single chronological view
- Include events from Kubernetes-owned children (Pods, ReplicaSets) of OPM-managed workloads
- Support time-windowed filtering (`--since`), event type filtering (`--type`), and real-time streaming (`--watch`)
- Follow existing CLI patterns: shared flag groups, separation of cmd/business-logic, table/json/yaml output
- Color-coded output consistent with existing style conventions

**Non-Goals:**

- Custom event storage or event history beyond what Kubernetes retains (events are typically garbage-collected after 1 hour)
- Cross-namespace event aggregation (events are scoped to the release namespace)
- Event correlation or root-cause analysis (we present raw events, not diagnostics)
- Metrics or log aggregation (this is events only)

## Decisions

### Decision 1: Event collection via bulk fetch + client-side filtering

**Choice:** Fetch all events in the namespace, then filter client-side by matching `involvedObject.uid` against our collected UIDs.

**Alternatives considered:**

- **(A) Per-resource field selector queries** — one `Events().List(fieldSelector=involvedObject.uid=<uid>)` per resource. Precise, but generates N API calls for N resources. A release with 6 OPM resources + 10 pods + 5 ReplicaSets = 21 API calls.
- **(B) Bulk fetch + client-side filter** — one `Events(namespace).List()` call, filter in memory. One API call regardless of resource count. Events per namespace are typically small (hundreds, not thousands).

**Rationale:** Option B is simpler and faster for typical module sizes. Kubernetes namespaces rarely have more than a few thousand events, and we're already filtering by `--since` which reduces the set further. One API call is always better than N.

### Decision 2: OwnerReference traversal depth

**Choice:** Two levels deep from OPM-managed resources: parent → child → grandchild.

This covers the common chains:

```text
Deployment (OPM-managed)
└── ReplicaSet (K8s-owned, level 1)
    └── Pod (K8s-owned, level 2)    ← events here are most useful

StatefulSet (OPM-managed)
└── Pod (K8s-owned, level 1)        ← events here are most useful

Job (OPM-managed)
└── Pod (K8s-owned, level 1)        ← events here are most useful
```

**Implementation:** For each OPM-managed workload resource, query for child resources that have an `ownerReference` pointing to it. Concretely:

1. For Deployments: list ReplicaSets in namespace, filter by `ownerReferences[].uid == deployment.uid`, then list Pods with `ownerReferences[].uid == replicaset.uid`
2. For StatefulSets: list Pods with `ownerReferences[].uid == statefulset.uid`
3. For DaemonSets: list Pods with `ownerReferences[].uid == daemonset.uid`
4. For Jobs: list Pods with `ownerReferences[].uid == job.uid`

**Alternatives considered:**

- **(A) No traversal** — only events for OPM-labeled resources. Misses the most useful events (Pod-level).
- **(B) Unlimited depth** — follow ownerReferences recursively. Unnecessary complexity — Kubernetes ownership chains are at most 3 levels deep for standard workloads, and we cover the useful ones at depth 2.

**Rationale:** Depth 2 covers all standard Kubernetes workload hierarchies. The implementation uses targeted queries (list Pods by label or ownerReference filter), not generic recursive walking.

### Decision 3: Child resource discovery implementation

**Choice:** Add a new `DiscoverChildren` function to `internal/kubernetes/discovery.go` that takes parent resources and returns their Kubernetes-owned children.

```go
// DiscoverChildren finds Kubernetes-owned child resources of the given parents.
// It walks ownerReferences downward up to maxDepth levels.
// Returns children grouped by parent UID for attribution.
func DiscoverChildren(
    ctx context.Context,
    client *Client,
    parents []*unstructured.Unstructured,
    namespace string,
    maxDepth int,
) ([]*unstructured.Unstructured, error)
```

**How it works:**

```text
                    ┌──────────────────────────────────────────┐
                    │          DiscoverChildren flow            │
                    └──────────────────────────────────────────┘

Input: [Deployment/web, StatefulSet/db, ConfigMap/cfg, Service/svc]
                │
                ▼
   Filter to workload kinds only
   [Deployment/web, StatefulSet/db]
                │
                ▼
   For each workload, query children by kind:
   ┌─────────────────────────────────────────────────────────┐
   │ Deployment/web                                          │
   │   → List ReplicaSets where ownerRef.uid == web.uid      │
   │     Found: [RS/web-abc12, RS/web-old99]                 │
   │   → List Pods where ownerRef.uid in [abc12.uid, old99]  │
   │     Found: [Pod/web-abc12-x1, Pod/web-abc12-x2]         │
   ├─────────────────────────────────────────────────────────┤
   │ StatefulSet/db                                          │
   │   → List Pods where ownerRef.uid == db.uid              │
   │     Found: [Pod/db-0]                                   │
   └─────────────────────────────────────────────────────────┘
                │
                ▼
   Output: [RS/web-abc12, RS/web-old99, Pod/web-abc12-x1,
            Pod/web-abc12-x2, Pod/db-0]
```

This is **not** a generic recursive walker. It uses knowledge of Kubernetes workload hierarchies to make targeted queries:

| Parent Kind | Child Query | Grandchild Query |
|-------------|-------------|------------------|
| Deployment | ReplicaSets by ownerRef | Pods by ownerRef |
| StatefulSet | Pods by ownerRef | - |
| DaemonSet | Pods by ownerRef | - |
| Job | Pods by ownerRef | - |
| CronJob | Jobs by ownerRef | Pods by ownerRef |

**Alternatives considered:**

- **(A) Extend DiscoverResources** — add child traversal as an option on the existing function. Rejected because it would complicate an already-complex function and mix two different concerns (label-based discovery vs. ownerReference traversal).
- **(B) Generic recursive ownerReference walker** — traverse all kinds, not just known workloads. Rejected because it's overengineered and could query unexpected resource types.

### Decision 4: Watch mode implementation

**Choice:** Use the Kubernetes Watch API on `v1.Event` resources, filtering events in real-time.

```text
┌─────────────────────────────────────────────────────────┐
│                   Watch mode flow                        │
└─────────────────────────────────────────────────────────┘

1. Discover OPM resources + children (same as one-shot mode)
2. Collect all UIDs into a Set
3. Start Watch on Events(namespace)
4. For each incoming event:
   └── if event.involvedObject.uid ∈ UID set
       └── if passes --type and --since filters
           └── format and print (append to output)
5. On SIGINT/SIGTERM → clean exit (code 0)
```

Watch mode **appends** new events to the terminal (streaming style, like `kubectl get events --watch`). This is different from `mod status --watch` which clears and redraws the screen. The streaming approach is correct for events because:

- Events are temporal — you want to see the sequence
- Clearing the screen would lose history
- Users expect `--watch` on events to behave like a log tail

**Known limitation:** The UID set is computed once at startup. If new Pods are created after the watch starts (e.g., a scaling event), their UIDs won't be in the set. This is acceptable because:

- The Deployment/ReplicaSet events (which ARE in the set) will show the scaling action
- For long-running watches, the user can restart the command
- Dynamically refreshing the UID set would add significant complexity

### Decision 5: `--since` flag parsing

**Choice:** Accept Go-style duration strings (`30m`, `1h`, `2h30m`, `1d`) with a custom extension for days.

```go
// parseSince converts a since flag value to a time.Time cutoff.
// Supports Go duration syntax (30m, 1h, 2h30m) plus "d" for days.
func parseSince(since string) (time.Time, error)
```

Go's `time.ParseDuration` doesn't support days (`d`). We'll preprocess the string: if it contains `d`, extract the days and convert to hours before parsing. This matches user expectations from tools like Prometheus and `kubectl`.

Examples: `30m`, `1h`, `2h30m`, `1d`, `7d`.

### Decision 6: Output table format

**Choice:** Match `kubectl get events` column structure with OPM styling.

```text
LAST SEEN   TYPE      RESOURCE                          REASON            MESSAGE
3m          Normal    Deployment/jellyfin-server         ScalingUp         Scaled up replica set to 3
3m          Normal    ReplicaSet/jellyfin-server-abc12   SuccessfulCreate  Created pod: abc12-x1
2m          Warning   Pod/jellyfin-server-abc12-x2      OOMKilled         Container jellyfin OOMKilled
1m          Warning   Pod/jellyfin-server-abc12-x2      BackOff           Back-off restarting failed container
```

Column details:

| Column | Source | Formatting |
|--------|--------|------------|
| LAST SEEN | `event.LastTimestamp` | Relative duration (e.g., `3m`, `1h`, `2d`) using existing `formatDuration` |
| TYPE | `event.Type` | `Warning` in yellow, `Normal` in dim gray |
| RESOURCE | `event.InvolvedObject.Kind/Name` | Cyan (matches `styleNoun`) |
| REASON | `event.Reason` | Default (unstyled) |
| MESSAGE | `event.Message` | Default, truncated to terminal width if needed |

Events are sorted by `lastTimestamp` ascending (oldest first, newest at bottom — matches log convention).

### Decision 7: Structured output (JSON/YAML)

**Choice:** For `--output json` and `--output yaml`, emit a structured `EventsResult` with full event details.

```go
type EventsResult struct {
    ReleaseName string        `json:"releaseName,omitempty"`
    ReleaseID   string        `json:"releaseId,omitempty"`
    Namespace   string        `json:"namespace"`
    Events      []EventEntry  `json:"events"`
}

type EventEntry struct {
    LastSeen  string `json:"lastSeen"`            // RFC3339
    Type      string `json:"type"`                // "Normal" or "Warning"
    Kind      string `json:"kind"`                // involvedObject.kind
    Name      string `json:"name"`                // involvedObject.name
    Reason    string `json:"reason"`
    Message   string `json:"message"`
    Count     int32  `json:"count"`               // occurrence count
    FirstSeen string `json:"firstSeen,omitempty"` // RFC3339
}
```

### Decision 8: Package structure

Following the existing separation of concerns:

```text
internal/cmd/mod_events.go           Command definition, flag parsing
internal/cmd/mod_events_test.go      Flag validation tests

internal/kubernetes/events.go        Event collection, filtering, formatting
internal/kubernetes/events_test.go   Event logic tests

internal/kubernetes/children.go      OwnerReference child discovery
internal/kubernetes/children_test.go Child discovery tests
```

`children.go` is separate from `discovery.go` because it has a fundamentally different concern: `discovery.go` finds OPM-managed resources by labels; `children.go` finds Kubernetes-owned resources by ownerReferences. Keeping them separate follows the single-responsibility principle and avoids bloating `discovery.go`.

### Decision 9: Signal handling and graceful shutdown

**Choice:** Same pattern as `mod status --watch` — context cancellation on SIGINT/SIGTERM.

```go
ctx, cancel := context.WithCancel(ctx)
defer cancel()

sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
go func() {
    <-sigCh
    cancel()
}()
```

Exit code 0 on clean interrupt. This matches the existing `runStatusWatch` pattern in `mod_status.go:175-207`.

## Risks / Trade-offs

**[Risk] Kubernetes event retention is short (default 1 hour)**
→ The `--since` default of `1h` matches the typical retention window. We document this limitation in the command help text. If events have been garbage-collected, the command will simply show an empty table (not an error).

**[Risk] Bulk event fetch could be slow in namespaces with many events**
→ Kubernetes event lists are typically small (hundreds). For namespaces with thousands of events, the `--since` filter reduces the working set. If this becomes a problem, we can add server-side `fieldSelector` on `lastTimestamp` in a future optimization.

**[Risk] Watch mode UID set is static**
→ New Pods created after watch starts won't have their events captured. Documented as a known limitation. Acceptable because the parent resource events (Deployment, ReplicaSet) still capture the high-level actions.

**[Risk] OwnerReference traversal generates additional API calls**
→ For a typical module with 2-3 workloads, this means 3-6 additional List calls (ReplicaSets + Pods). This is acceptable. For modules with many workloads, the calls are parallelizable in a future optimization (but we start sequential for simplicity per Principle VII).

**[Trade-off] Targeted child discovery vs. generic recursive walker**
→ We chose targeted (knows about Deployment→RS→Pod chains) over generic (recursively follows all ownerReferences). This means if new workload types emerge (e.g., a custom CRD that owns Pods), we'd need to add explicit support. The upside is predictable behavior and no surprise API calls.
