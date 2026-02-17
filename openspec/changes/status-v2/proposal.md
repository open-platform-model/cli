## Why

The current `opm mod status` command is essentially `kubectl get` with OPM label filtering. It answers one question — "are my resources alive?" — with a flat table of Kind/Name/Namespace/Status/Age. That's a starting point, not a destination.

When we analyzed the questions people actually ask about deployed modules, we identified seven levels of insight:

```
L1: Is it alive?           ← Current status answers this
L2: Is it healthy?         ← Partially (Ready/NotReady, but no detail)
L3: Is it current?         ← Not in status (separate: mod diff)
L4: What happened?         ← Not available (separate: mod events)
L5: What IS this thing?    ← Barely (module ID, release ID — no version, no components)
L6: How is it doing?       ← Not available (no replicas, no pod details, no restarts)
L7: What SHOULD I do?      ← Future (version upgrades, deprecations)
```

This change targets L2, L5, and L6 — making status genuinely useful without requiring users to fall back to `kubectl describe` when something is wrong. L3 and L4 are handled by separate changes (`mod-tree`, `mod-events`). L7 is future work.

The key pain point: when status shows `NotReady`, the user hits a dead end. They have to leave OPM and start spelunking with kubectl to find out *why*. Status should tell them why.

This is a **MINOR** change — new flags with sensible defaults, no breaking changes to existing behavior. The only potentially impactful change is the addition of semantic exit codes (exit 2 for unhealthy resources), which could affect scripts that currently treat exit 0 as "command ran successfully" regardless of health.

## What Changes

### Richer metadata header

The status header currently shows Module ID and Release ID (UUIDs). These are not particularly useful to humans. Replace with actionable metadata sourced from labels already on the resources:

```
Current:
  Module ID:  c1cbe76d-5687-5a47-bfe6-83b081b15413
  Release ID: a1b2c3d4-e5f6-7890-abcd-ef1234567890

Proposed:
  Release:    jellyfin-media
  Version:    1.2.0
  Namespace:  media
  Status:     Ready
  Resources:  6 total (6 ready)
```

All data sourced from existing labels — no new annotations or module source required:

- Release name: `module-release.opmodel.dev/name`
- Version: `module-release.opmodel.dev/version`
- Namespace: from the resource metadata
- Status: existing aggregate health computation
- Resource count: computed from discovered resources

### Component column in default table

Add a COMPONENT column populated from the `component.opmodel.dev/name` label already stamped by CUE transformers. This groups resources by their logical component, making it easy to identify which part of a module owns each resource.

```
KIND          NAME               COMPONENT   STATUS   AGE
Deployment    jellyfin-server    server      Ready    5d
Service       jellyfin-svc       server      Ready    5d
ConfigMap     jellyfin-config    server      Ready    5d
StatefulSet   jellyfin-db        database    Ready    5d
PVC           jellyfin-data      database    Ready    5d
Ingress       jellyfin-ing       ingress     Ready    5d
```

Resources without a component label show `-` in the column.

### Wide output format (`-o wide`)

Extends the table with workload-specific details. Additional columns vary by resource kind:

```
KIND          NAME               COMPONENT   STATUS   REPLICAS   IMAGE                        AGE
Deployment    jellyfin-server    server      Ready    3/3        jellyfin/jellyfin:10.8.13    5d
Service       jellyfin-svc       server      Ready    -          -                            5d
ConfigMap     jellyfin-config    server      Ready    -          -                            5d
StatefulSet   jellyfin-db        database    Ready    1/1        postgres:16                  5d
PVC           jellyfin-data      database    Ready    -          10Gi (Bound)                 5d
Ingress       jellyfin-ing       ingress     Ready    -          jellyfin.local               5d
```

Extra columns by resource kind:

- **Deployment, StatefulSet, DaemonSet**: REPLICAS (`ready/desired`), IMAGE (first container image)
- **PVC**: capacity and phase in REPLICAS column (e.g., `10Gi (Bound)`)
- **Ingress**: first host rule in IMAGE column
- **All others**: `-` for both columns

This requires querying the workload specs (already fetched during discovery) — no additional K8s API calls for the resources themselves.

### Verbose mode (`--verbose`)

When a workload is unhealthy, verbose mode drills down to the pod level to show *why*. This is the biggest UX improvement — turning `NotReady` from a dead end into an actionable diagnostic.

```
Deployment    jellyfin-server    server      NotReady   5d
  Pods: 1/3 ready
    jellyfin-server-abc12-x1    Running     (ready)
    jellyfin-server-abc12-x2    CrashLoop   OOMKilled (512Mi limit), 5 restarts
    jellyfin-server-abc12-x3    Pending     Insufficient memory
```

Implementation requires walking ownerReferences from workloads → ReplicaSets → Pods. This involves additional K8s API calls (listing pods by owner), so it's opt-in via the flag.

Pod details include:

- Pod name (short, without the full hash prefix where possible)
- Pod phase (Running, Pending, Failed, CrashLoopBackOff)
- Condition reason when unhealthy (OOMKilled, ImagePullBackOff, Insufficient resources, etc.)
- Restart count for containers with restarts

Only unhealthy workloads get the pod drill-down. Ready workloads just show their normal row. This keeps output focused on what needs attention.

### Semantic exit codes

```
Exit 0: All resources healthy
Exit 1: General error (connectivity, invalid flags, etc.)
Exit 2: Resources not ready (command succeeded, but health check failed)
Exit 3: No resources found
```

Enables CI/CD scripting: `opm mod status ... || handle_failure $?`

Currently, `--ignore-not-found` returns exit 0 when no resources are found. This behavior is preserved — the flag overrides exit 3 to exit 0.

### Color-coded output

Apply the existing lipgloss color palette from `internal/output/styles.go`:

| Element | Color | Constant |
|---------|-------|----------|
| STATUS: Ready/Complete | green | `colorGreen` (82) |
| STATUS: NotReady | red | `colorRed` (196) |
| STATUS: Unknown | yellow | `ColorYellow` (220) |
| COMPONENT values | cyan | `ColorCyan` (14) — matches existing noun styling |
| Resource names (NAME column) | cyan | `ColorCyan` (14) — matches `FormatResourceLine` pattern |
| Structural/separator elements | dim gray | `colorDimGray` (240) |

Color output follows standard terminal conventions: disabled when stdout is not a TTY or when `NO_COLOR` env var is set.

## Capabilities

### New Capabilities

- `status-wide-output`: Wide table format (`-o wide`) showing workload-specific columns (replicas, images, PVC capacity, Ingress hosts)
- `status-verbose`: Verbose mode (`--verbose`) with pod-level drill-down for unhealthy workloads showing pod phase, condition reasons, and restart counts
- `status-exit-codes`: Semantic exit codes (0/1/2/3) for CI/CD integration

### Modified Capabilities

- `mod-status`: Existing status behavior enhanced — component column added to default table, richer metadata header replacing raw UUIDs, color-coded output. No existing flags removed or changed. Default output format remains `table`.

## Impact

- **Packages modified**: `internal/cmd/mod_status.go`, `internal/kubernetes/status.go`, `internal/kubernetes/health.go`, `internal/output/styles.go`
- **Data model changes**: `StatusResult` and `resourceHealth` structs gain new fields (component, version, replica info, pod details). Existing JSON/YAML output gains new fields (additive, non-breaking).
- **New K8s API calls**: Default and wide modes require no new API calls beyond existing resource discovery. Verbose mode adds pod listing via `core/v1` Pods API with ownerReference filtering.
- **No new dependencies**: Uses existing `client-go`, `lipgloss`, and `output` packages.
- **No config changes**: All new behavior is flag-driven with backward-compatible defaults.
- **Exit code change**: Scripts relying on exit 0 meaning "status ran" (regardless of health) will see exit 2 when resources are not ready. This is the intended behavior for CI/CD, but could affect existing automation. The `--ignore-not-found` flag continues to suppress exit 3 → exit 0.
