## Why

When `opm mod status` reports a resource as `NotReady`, users hit a dead end. The status command tells you *that* something is wrong, but not *why*. To diagnose, users must switch to `kubectl describe` on each resource, then `kubectl get events` — often across multiple resource types (Deployments, Pods, ReplicaSets) — mentally piecing together a timeline from scattered Kubernetes events.

OPM manages resources across many kinds, and the most actionable diagnostic information lives on Kubernetes-owned children (Pods, ReplicaSets) that the user has to discover manually. A Deployment's OOMKilled pod, a StatefulSet's scheduling failure, a Job's backoff — these are all events on *child* resources that aren't directly OPM-labeled.

`opm mod events` closes this gap: one command to see everything that happened to a release, across all its resources and their children, in chronological order.

## What Changes

### New command: `opm mod events`

Aggregates Kubernetes events from all resources belonging to an OPM release into a single chronological view.

**Usage examples:**

```bash
# Show events from the last hour (default)
opm mod events --release-name jellyfin -n media

# Show events from the last 30 minutes
opm mod events --release-name jellyfin -n media --since 30m

# Show only warnings
opm mod events --release-name jellyfin -n media --type Warning

# Stream events in real-time
opm mod events --release-name jellyfin -n media --watch

# By release ID instead of name
opm mod events --release-id a1b2c3d4-e5f6-7890 -n media

# JSON output for tooling
opm mod events --release-name jellyfin -n media -o json
```

**Default table output:**

```
LAST SEEN   TYPE      RESOURCE                        REASON          MESSAGE
3h          Normal    Deployment/jellyfin-server       ScalingUp       Scaled up to 3
3h          Normal    ReplicaSet/jellyfin-abc12        SuccessCreate   Created pod abc12-x1
2h          Warning   Pod/jellyfin-abc12-x2            OOMKilled       Container "jf" OOM
1h          Warning   Pod/jellyfin-abc12-x2            BackOff         Restarting container
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--release-name` | | | Release name selector (mutually exclusive with `--release-id`) |
| `--release-id` | | | Release ID selector (mutually exclusive with `--release-name`) |
| `--namespace` | `-n` | | Target namespace |
| `--since` | | `1h` | Time window for events (e.g., `30m`, `2h`, `1d`) |
| `--type` | | all | Filter by event type: `Normal`, `Warning` |
| `--watch` | | `false` | Stream new events in real-time |
| `--output` | `-o` | `table` | Output format: `table`, `json`, `yaml` |
| `--kubeconfig` | | | Kubernetes config file path |
| `--context` | | | Kubernetes context name |
| `--ignore-not-found` | | `false` | Exit 0 when no resources match the selector |

### Event collection strategy

The command discovers events in two phases:

1. **Discover OPM-managed resources** — via `cmdutil.ResolveInventory`, which reads the release inventory Secret and fetches each tracked resource by targeted GET (same pattern as `mod status`, `mod delete`, `mod tree`). The inventory Secret is identified by `--release-name` or `--release-id`.
2. **Walk ownerReferences downward** — for workload resources (Deployment, StatefulSet, DaemonSet, Job), traverse ownerReferences to find Kubernetes-owned children (ReplicaSets, Pods) that aren't OPM-managed but belong to OPM-managed parents. These children are where the most useful events live (OOMKilled, ImagePullBackOff, scheduling failures, etc.).
3. **Query events** — collect all `involvedObject` UIDs from both phases, query `v1.Event` list filtered by namespace and sorted by `lastTimestamp`. Filter to events matching collected UIDs.
4. **Apply filters** — filter by `--since` time window and `--type` if specified.

### Color scheme

Follows existing CLI style conventions from `internal/output/styles.go`:

| Element | Color | Source |
|---------|-------|--------|
| `Warning` type | yellow (`output.ColorYellow`, 220) | Matches existing "configured" status style |
| `Normal` type | faint/dim (`output.Dim()`) | Non-urgent, don't shout |
| Resource names in RESOURCE column | cyan (`output.StyleNoun()`) | Matches existing `styleNoun` convention |

### Watch mode

For `--watch`, use the Kubernetes Watch API on `v1.Event` resources filtered by namespace. Events are filtered in real-time against the set of collected UIDs. New events append to the output (streaming, not full-screen refresh like `mod status --watch`).

Clean exit on Ctrl+C (SIGINT/SIGTERM), exit code 0.

### Shared patterns with existing commands

This command reuses established patterns:

- **`ReleaseSelectorFlags`** (`internal/cmdutil/flags.go`): Shared flag group for `--release-name`/`--release-id`/`-n` with mutual exclusivity validation.
- **`K8sFlags`** (`internal/cmdutil/flags.go`): Shared `--kubeconfig`/`--context` flags.
- **`cmdutil.ResolveInventory`** (`internal/cmdutil/inventory.go`): Shared inventory resolution and live resource discovery. Reads the inventory Secret and fetches live resources — same function used by `mod status`, `mod delete`, and `mod tree`.
- **Output formatting**: Table via `output.NewTable`, JSON via `json.MarshalIndent`, YAML via `yaml.Marshal` — same patterns as `mod status`.

## Capabilities

### New Capabilities

- `mod-events`: The `opm mod events` command — event aggregation from OPM-managed resources and their Kubernetes-owned children, time-windowed and type-filtered, with watch mode and structured output support.

### Modified Capabilities

- `resource-discovery`: Discovery needs a new capability to walk ownerReferences downward from OPM-managed workloads to find Kubernetes-owned children (Pods, ReplicaSets) that aren't tracked in the inventory. Given parent resources, find their Kubernetes-owned children. Note: `mod tree` already implements equivalent ownership walking in `internal/kubernetes/tree.go` (`walkOwnership` and friends) — the events command requires a UID-returning variant (rather than the `ResourceNode`-returning variant used for tree display).

## Impact

- **New files**: `internal/cmd/mod/events.go`, `internal/cmd/mod/events_test.go`, `internal/kubernetes/events.go`, `internal/kubernetes/events_test.go`, `internal/kubernetes/children.go`, `internal/kubernetes/children_test.go`
- **Modified files**: `internal/cmd/mod/mod.go` (register new subcommand)
- **Dependencies**: No new external dependencies — uses existing `client-go` APIs (`v1.EventList`, `v1.Event`, Watch API)
- **SemVer**: MINOR — new command, no breaking changes to existing commands or flags
- **Testing**: Unit tests for event filtering/formatting, integration tests for ownerReference traversal
