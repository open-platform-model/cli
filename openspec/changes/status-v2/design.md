## Context

The current `opm mod status` implementation lives across three files:

- `internal/cmd/mod/status.go` — command wiring, flag parsing, watch loop
- `internal/kubernetes/status.go` — `GetReleaseStatus()`, `StatusResult`, formatting
- `internal/kubernetes/health.go` — `evaluateHealth()`, health status types

The data flow today:

```text
cmd/mod/status.go              inventory/                kubernetes/status.go
┌───────────────┐                ┌────────────────────┐    ┌────────────────────┐
│ runStatus()   │──────────────▶│ GetInventory() or  │    │ GetReleaseStatus() │
│               │                │ FindInventoryBy    │    │                    │
│ • resolve     │                │ ReleaseName()      │    │ • evaluate health  │
│   k8s config  │                └────────┬───────────┘    │ • compute age      │
│ • load        │                         │                │ • populate         │
│   inventory   │                         ▼                │   component from   │
│ • build       │                ┌────────────────────┐    │   ComponentMap     │
│   ComponentMap│──────────────▶│ DiscoverResources  │    │ • populate         │
│ • extract     │                │ FromInventory()    │    │   version/name     │
│   version     │                │ (targeted GET per  │    │   from opts        │
│               │                │  inventory entry)  │    │                    │
│               │                │                    │    │ Returns:           │
│               │                │ Returns:           │    │ StatusResult{      │
│               │                │ live []*Unstructured    │   Resources[]      │
│               │                │ missing []Entry    │    │   ReleaseName      │
│               │                └────────┬───────────┘    │   Version          │
│               │───────────────────────▶│                │   AggregateStatus  │
│ format +      │◀────────────────────────────────────────│   Summary          │
│ print         │                                          │ }                  │
└───────────────┘                                          └────────────────────┘
```

Key constraints from the current implementation:

1. **Status is inventory-driven** — resources are discovered via the inventory Secret (targeted GET per entry), not via label scanning. The command layer pre-fetches live resources and passes them via `StatusOptions.InventoryLive`. No `discovery.go` exists in `internal/kubernetes/`.
2. **Component names come from the inventory** — `InventoryEntry.Component` stores the component name at apply time. The command layer builds a `ComponentMap` from inventory entries and passes it through `StatusOptions`. No label constants are needed for component lookup.
3. **Release metadata comes from the inventory** — `ReleaseName` and `Namespace` are already in `StatusOptions`; `Version` is sourced from `inv.Changes[latest].Source.Version` in the command layer.
4. **Discovery returns `*unstructured.Unstructured`** — the raw K8s objects are available for wide-mode field extraction. For missing resources (tracked in inventory but no longer on the cluster), there is no live object.
5. **The table uses `output.NewTable()`** which renders via `lipgloss/table`. Cell content is currently raw strings — color must be applied by rendering styled strings into cell values before passing to the table.
6. **Exit codes are defined in `internal/errors/errors.go`** with existing constants: `ExitSuccess(0)`, `ExitGeneralError(1)`, `ExitValidationError(2)`, `ExitConnectivityError(3)`, `ExitPermissionDenied(4)`, `ExitNotFound(5)`.

## Goals / Non-Goals

**Goals:**

- Add component, version, and resource summary to status output using data already available in the inventory
- Provide wide format (`-o wide`) with workload-specific columns extracted from the unstructured objects already returned by discovery
- Provide verbose mode (`--verbose`) with pod-level diagnostics for unhealthy workloads via label selector-based pod listing
- Define semantic exit codes that distinguish "command failed" from "resources unhealthy"
- Apply color to status output using the existing lipgloss palette

**Non-Goals:**

- Drift detection (handled separately by `mod diff`)
- Event aggregation (handled separately by `mod events`)
- Module source or registry queries — status remains source-free
- Changes to the inventory discovery mechanism itself
- Custom resource detailed status (beyond existing Ready condition check)

## Decisions

### Decision 1: Data model changes to `StatusResult` and `resourceHealth`

**Context**: The current `resourceHealth` struct has 5 fields: Kind, Name, Namespace, Status, Age. We need component, version, replica info, and pod details.

**Options considered**:

1. **Extend `resourceHealth` with all new fields** — simple, one struct
2. **Create separate `WideInfo` and `VerboseInfo` sub-structs** — cleaner separation, only populated when needed

**Decision**: Option 2 — separate sub-structs. Wide and verbose data are optional and mode-dependent. Keeping them in sub-structs makes the JSON/YAML output cleaner (omitempty) and the code paths clearer.

```go
type resourceHealth struct {
    Kind      string       `json:"kind" yaml:"kind"`
    Name      string       `json:"name" yaml:"name"`
    Namespace string       `json:"namespace" yaml:"namespace"`
    Component string       `json:"component,omitempty" yaml:"component,omitempty"`
    Status    healthStatus `json:"status" yaml:"status"`
    Age       string       `json:"age" yaml:"age"`
    Wide      *wideInfo    `json:"wide,omitempty" yaml:"wide,omitempty"`
    Verbose   *verboseInfo `json:"verbose,omitempty" yaml:"verbose,omitempty"`
}

type wideInfo struct {
    Replicas string `json:"replicas,omitempty" yaml:"replicas,omitempty"` // "3/3", "10Gi (Bound)"
    Image    string `json:"image,omitempty" yaml:"image,omitempty"`       // "nginx:1.25", "app.local"
}

type verboseInfo struct {
    Pods []podInfo `json:"pods,omitempty" yaml:"pods,omitempty"`
}

type podInfo struct {
    Name     string `json:"name" yaml:"name"`
    Phase    string `json:"phase" yaml:"phase"`       // Running, Pending, Failed
    Ready    bool   `json:"ready" yaml:"ready"`
    Reason   string `json:"reason,omitempty" yaml:"reason,omitempty"`     // OOMKilled, ImagePullBackOff
    Restarts int    `json:"restarts" yaml:"restarts"`
}
```

The `StatusResult` header also changes:

```go
type StatusResult struct {
    ReleaseName     string           `json:"releaseName" yaml:"releaseName"`
    Version         string           `json:"version,omitempty" yaml:"version,omitempty"`
    Namespace       string           `json:"namespace" yaml:"namespace"`
    Resources       []resourceHealth `json:"resources" yaml:"resources"`
    AggregateStatus healthStatus     `json:"aggregateStatus" yaml:"aggregateStatus"`
    Summary         statusSummary    `json:"summary" yaml:"summary"`
}

type statusSummary struct {
    Total    int `json:"total" yaml:"total"`
    Ready    int `json:"ready" yaml:"ready"`
    NotReady int `json:"notReady" yaml:"notReady"`
}
```

The old `ModuleID` and `ReleaseID` fields (raw UUIDs) are removed from the struct. They provided no user value. If needed for debugging, they remain accessible on the raw inventory Secret.

### Decision 2: Component extraction from inventory

**Context**: Previously the design referenced a `LabelComponentName` label constant from `internal/kubernetes/discovery.go`. That file no longer exists — resource discovery is inventory-driven. The component name is already recorded in `InventoryEntry.Component` at apply time.

**Decision**: The command layer builds a `ComponentMap map[string]string` keyed by `"Kind/Namespace/Name"` from the inventory entries before calling `GetReleaseStatus`, and passes it through `StatusOptions`:

```go
// In runStatus, after loading inventory:
componentMap := make(map[string]string)
if len(inv.Index) > 0 {
    if change, ok := inv.Changes[inv.Index[0]]; ok {
        for _, entry := range change.Inventory.Entries {
            key := entry.Kind + "/" + entry.Namespace + "/" + entry.Name
            componentMap[key] = entry.Component
        }
    }
}
statusOpts.ComponentMap = componentMap
```

In `GetReleaseStatus`, when building `resourceHealth`:

```go
key := res.GetKind() + "/" + res.GetNamespace() + "/" + res.GetName()
component := opts.ComponentMap[key] // "" if not present
```

Resources not in the map get `""` which renders as `-` in the table. This is more reliable than label reading: the component is recorded at apply time regardless of whether the CUE transformer stamps a label on the live resource.

No label constants are needed for component lookup.

### Decision 3: Version and release metadata from inventory

**Context**: Previously the design referenced a `labelReleaseVersion` constant for `module-release.opmodel.dev/version` label extraction from live resources. Label-based discovery no longer exists.

**Decision**: All release metadata is sourced from the inventory, which is already loaded by the command layer:

- **ReleaseName**: Already in `StatusOptions.ReleaseName` (from `rsf.ReleaseName` or inventory)
- **Namespace**: Already in `StatusOptions.Namespace` (from resolved k8s config)
- **Version**: Extracted from `inv.Changes[inv.Index[0]].Source.Version` in the command layer and passed via new `StatusOptions.Version` field

```go
// In runStatus, after loading inventory:
var version string
if len(inv.Index) > 0 {
    if change, ok := inv.Changes[inv.Index[0]]; ok {
        version = change.Source.Version
    }
}
statusOpts.Version = version
```

In `GetReleaseStatus`:

```go
result.ReleaseName = opts.ReleaseName
result.Version     = opts.Version
result.Namespace   = opts.Namespace
```

No label constants are needed for version or release name extraction.

### Decision 4: Wide format — extracting workload details from unstructured objects

**Context**: Discovery returns full `*unstructured.Unstructured` objects. The `.spec` and `.status` fields contain replica counts, container images, PVC capacity, and Ingress hosts. We need to extract these without importing typed K8s API structs.

**Decision**: Use `unstructured.NestedInt64`, `unstructured.NestedString`, and `unstructured.NestedSlice` to pull fields from the raw objects. Create extraction functions per resource kind:

```go
func extractWideInfo(resource *unstructured.Unstructured) *wideInfo
```

Extraction logic by kind:

```text
┌──────────────────┬──────────────────────────────────┬──────────────────────────────────┐
│ Kind             │ REPLICAS                         │ IMAGE                            │
├──────────────────┼──────────────────────────────────┼──────────────────────────────────┤
│ Deployment       │ status.readyReplicas /           │ spec.template.spec               │
│                  │ spec.replicas                    │   .containers[0].image           │
├──────────────────┼──────────────────────────────────┼──────────────────────────────────┤
│ StatefulSet      │ status.readyReplicas /           │ spec.template.spec               │
│                  │ spec.replicas                    │   .containers[0].image           │
├──────────────────┼──────────────────────────────────┼──────────────────────────────────┤
│ DaemonSet        │ status.numberReady /             │ spec.template.spec               │
│                  │ status.desiredNumberScheduled    │   .containers[0].image           │
├──────────────────┼──────────────────────────────────┼──────────────────────────────────┤
│ PVC              │ status.capacity.storage +        │ -                                │
│                  │ " (" + status.phase + ")"        │                                  │
├──────────────────┼──────────────────────────────────┼──────────────────────────────────┤
│ Ingress          │ -                                │ spec.rules[0].host               │
├──────────────────┼──────────────────────────────────┼──────────────────────────────────┤
│ All others       │ -                                │ -                                │
└──────────────────┴──────────────────────────────────┴──────────────────────────────────┘
```

All fields are available on the unstructured objects already returned by `DiscoverResourcesFromInventory`. No additional API calls required for wide mode.

`extractWideInfo` is never called for `MissingResource` entries — they have no live object. Missing entries always have `Wide = nil`.

### Decision 5: Verbose mode — pod-level diagnostics via label selector

**Context**: When a Deployment/StatefulSet/DaemonSet is `NotReady`, the user needs to know *why*. The information is on the pods, which are children of the workload.

**Decision**: For each workload with `Status == healthNotReady`, list pods in the same namespace using the workload's `.spec.selector.matchLabels`. The walking strategy differs by workload type:

```text
Deployment  → owns ReplicaSet(s) → owns Pod(s)     (2-hop, use label selector shortcut)
StatefulSet → owns Pod(s) directly                   (1-hop)
DaemonSet   → owns Pod(s) directly                   (1-hop)
```

**Why label selector instead of ownerReference walking**: Using the workload's `.spec.selector.matchLabels` to list pods directly is simpler and a single API call. For all three workload types, their `.spec.selector.matchLabels` directly identifies their pods.

```go
func listWorkloadPods(ctx context.Context, client *Client, resource *unstructured.Unstructured) ([]podInfo, error)
```

This function:

1. Reads `.spec.selector.matchLabels` from the workload
2. Lists pods in the same namespace with that label selector
3. Extracts phase, conditions, container statuses, restart counts
4. Returns `[]podInfo`

`listWorkloadPods` is never called for `MissingResource` entries — they have no live object and no pod selector. Missing entries always have `Verbose = nil`.

**Performance**: One additional `core/v1 Pods` list call per unhealthy workload. Since verbose mode is opt-in and typically there are few unhealthy workloads, this is acceptable.

### Decision 6: StatusOptions changes — passing mode to the status layer

**Context**: `GetReleaseStatus` needs to know the release version (from inventory), a component name map (from inventory entries), and whether to compute wide/verbose data.

**Decision**: Extend `StatusOptions` with new fields:

```go
type StatusOptions struct {
    Namespace        string
    ReleaseName      string
    ReleaseID        string
    Version          string                // sourced from inv.Changes[latest].Source.Version
    ComponentMap     map[string]string     // "Kind/Namespace/Name" → component name
    OutputFormat     output.Format
    InventoryLive    []*unstructured.Unstructured
    MissingResources []MissingResource
    Wide             bool  // compute wide info (replicas, images)
    Verbose          bool  // compute verbose info (pod details for unhealthy workloads)
}
```

`GetReleaseStatus` populates `wideInfo` and `verboseInfo` on each resource only when the corresponding flag is true. `MissingResource` entries never have wide or verbose data populated.

### Decision 7: Exit codes — mapping to existing constants

**Context**: The proposal specifies exit 0 (healthy), exit 1 (error), exit 2 (not ready), exit 3 (no resources). `noResourcesFoundError` is not a Kubernetes API error, so `ExitCodeFromK8sError` would map it to `ExitGeneralError(1)` — incorrect.

**Decision**: Reuse existing constants with an explicit `IsNoResourcesFound` check before delegating to `ExitCodeFromK8sError`:

```text
All healthy     → ExitSuccess (0)          — existing
General error   → ExitGeneralError (1)     — existing, already used
Resources not   → ExitValidationError (2)  — "the deployed state doesn't validate as healthy"
  ready
No resources    → ExitNotFound (5)         — existing, semantically correct
  found
```

In `fetchAndPrintStatus`:

```go
if kubernetes.IsNoResourcesFound(err) {
    // handle --ignore-not-found override, else ExitNotFound(5)
}
// fall through to ExitCodeFromK8sError for K8s API errors
```

The `--ignore-not-found` flag overrides `ExitNotFound(5)` to `ExitSuccess(0)`, preserving current behavior.

### Decision 8: Color rendering approach

**Context**: The `output.Table` renders cells as raw strings. Lipgloss color is applied by rendering styled strings *before* passing them as cell values.

**Decision**: Create health-status-specific style functions in `internal/output/styles.go` and apply them when building table rows. Do not modify the `Table` struct itself — color goes into the cell strings.

New functions in `output/styles.go`:

```go
// FormatHealthStatus renders a health status with color.
func FormatHealthStatus(status string) string

// FormatComponent renders a component name in cyan.
func FormatComponent(name string) string
```

Color mapping:

```go
func FormatHealthStatus(status string) string {
    switch status {
    case "Ready", "Complete":
        return lipgloss.NewStyle().Foreground(colorGreen).Render(status)
    case "NotReady", "Missing":
        return lipgloss.NewStyle().Foreground(colorRed).Render(status)
    case "Unknown":
        return lipgloss.NewStyle().Foreground(ColorYellow).Render(status)
    default:
        return status
    }
}
```

`FormatComponent` uses `styleNoun.Render(name)` (cyan), returning `-` unstyled for empty component names.

### Decision 9: Header rendering

**Context**: The current header uses `fmt.Sprintf` for "Module ID: ..." and "Release ID: ..." lines. The new header needs release name, version, namespace, status, and resource summary.

**Decision**: Render the header as a block of key-value lines with styled values, printed before the table.

```go
func formatStatusHeader(result *StatusResult) string {
    var sb strings.Builder
    sb.WriteString(fmt.Sprintf("Release:    %s\n", styleNoun.Render(result.ReleaseName)))
    if result.Version != "" {
        sb.WriteString(fmt.Sprintf("Version:    %s\n", result.Version))
    }
    sb.WriteString(fmt.Sprintf("Namespace:  %s\n", styleNoun.Render(result.Namespace)))
    sb.WriteString(fmt.Sprintf("Status:     %s\n", FormatHealthStatus(string(result.AggregateStatus))))
    sb.WriteString(fmt.Sprintf("Resources:  %d total (%d ready", result.Summary.Total, result.Summary.Ready))
    if result.Summary.NotReady > 0 {
        notReady := lipgloss.NewStyle().Foreground(colorRed).Render(
            fmt.Sprintf("%d not ready", result.Summary.NotReady))
        sb.WriteString(fmt.Sprintf(", %s", notReady))
    }
    sb.WriteString(")\n")
    return sb.String()
}
```

### Decision 10: Verbose output — inline pod details below table rows

**Context**: Verbose mode needs to show pod details indented below the parent workload row.

**Decision**: Render the standard table, then for each unhealthy workload, print an indented pod detail block below the table. This keeps the table clean and the pod details scannable.

```text
KIND          NAME               COMPONENT   STATUS     AGE
Deployment    jellyfin-server    server      NotReady   5d
Service       jellyfin-svc       server      Ready      5d
ConfigMap     jellyfin-config    server      Ready      5d

Deployment/jellyfin-server (1/3 ready):
    jellyfin-server-abc12-x1    Running       (ready)
    jellyfin-server-abc12-x2    CrashLoop     OOMKilled (512Mi limit), 5 restarts
    jellyfin-server-abc12-x3    Pending       Insufficient memory
```

This is simpler to implement, avoids fighting the table renderer, and produces output that's easy to parse visually. Detail blocks only appear for unhealthy workloads when `--verbose` is set.

### Decision 11: `StatusOptions.OutputFormat` and `-o wide`

**Context**: The `-o` flag currently accepts `table`, `yaml`, `json`. We're adding `wide` as a table variant.

**Decision**: `FormatWide Format = "wide"` is added to `internal/output/format.go` as a first-class format constant alongside `FormatTable`, `FormatJSON`, `FormatYAML`, `FormatDir`. `ParseFormat` and `Valid()` accept it. This matches the kubectl pattern where `-o wide` is a table variant.

When `outputFlag == "wide"`:

- `opts.OutputFormat = output.FormatWide`
- `opts.Wide = true` (set in `runStatus` after parsing)

`FormatStatus` maps `FormatWide` to the wide table renderer.

### Decision 12: File organization — where new code lives

**Decision**:

```text
internal/
├── output/
│   └── format.go              MODIFIED: add FormatWide constant
├── cmd/mod/
│   └── status.go              MODIFIED: new flags, exit code mapping, wide/verbose wiring,
│                                        ComponentMap building, Version extraction
├── kubernetes/
│   ├── status.go              MODIFIED: StatusResult changes, wide/verbose data population
│   ├── health.go              UNCHANGED
│   ├── wide.go                NEW: extractWideInfo() per-kind extraction
│   └── pods.go                NEW: listWorkloadPods(), podInfo, pod status extraction
└── output/
    └── styles.go              MODIFIED: FormatHealthStatus(), FormatComponent()
```

- `wide.go` keeps the per-kind unstructured field extraction isolated
- `pods.go` keeps the pod-listing and status extraction logic separate from the main status flow
- No changes to `internal/inventory/` — the discovery mechanism is unchanged

### Decision 13: Table rendering — plain columns vs. bordered

**Context**: The initial implementation used `output.Table` with `lipgloss.NormalBorder()`, which
renders box-drawing border characters (`│ ─ ┌ ┐ └ ┘`). The proposal examples and kubectl
conventions require plain, space-padded columns with no border characters.

**Decision**: Replace the lipgloss bordered renderer in `output.Table.String()` with a plain
column-aligned implementation:

- Column widths computed from max content width using `lipgloss.Width()` (ANSI-aware — correctly
  measures cells that contain color escape codes)
- Headers rendered bold cyan via lipgloss
- 3-space gap between columns (kubectl convention)
- Last column is never padded (no trailing whitespace)
- No border characters, no separator lines between header and data rows

The `tableStyle` struct and `defaultTableStyle()` function are removed. The `NewTable()` and
`Row()` call API is unchanged — no callers require modification.

### Decision 14: Verbose pod phase display and reason filtering

**Context**: Three bugs were found in the initial verbose output implementation:

1. `extractPodInfoFromPod` always stored `pod.Status.Phase` in `Phase` and put the waiting
   reason in `Reason`. Both were rendered, producing e.g. `Running … CrashLoopBackOff` instead
   of the spec-required `CrashLoop`.
2. `"Completed"` (`lastState.terminated.reason`) was shown as a diagnostic reason. It means the
   container exited with code 0 — normal exit, not an error — and is not useful as a diagnostic.
3. `formatVerboseBlocks` used hardcoded `%-50s %-12s` padding, producing large amounts of
   whitespace for short pod names.

**Decision**:

- **Phase override**: when a container has a waiting reason, `info.Phase` is overridden with
  `mapWaitingReason(reason)`. `CrashLoopBackOff` → `"CrashLoop"` (per spec); all other waiting
  reasons pass through unchanged. The `Reason` field is reserved for the last-terminated detail
  (OOMKilled, Error, etc.).
- **"Completed" filtering**: `lastState.terminated.reason == "Completed"` is skipped when
  populating `info.Reason`. A pod that keeps exiting with code 0 will show its restart count
  without the misleading "Completed" label.
- **Dynamic padding**: `formatVerboseBlocks` computes `nameWidth` and `phaseWidth` per block
  from actual content lengths, using 3-space gaps (consistent with the table renderer).
- **Detail column format**: when a termination reason is available (`Reason != ""`), it is
  used as the primary detail text (e.g. `"OOMKilled, 5 restarts"`). `"(not ready)"` is the
  fallback only when no specific reason exists. Ready pods always show `"(ready)"`.

## Risks / Trade-offs

### Risk: Pod listing performance in verbose mode

**Risk**: If a release has many unhealthy workloads, verbose mode makes one pod-list API call per unhealthy workload.

**Mitigation**: Verbose mode is opt-in (`--verbose`). The call is only made for `NotReady` workloads. In practice, most releases have 1-3 workloads. Future optimization: batch pod listing with a single namespace-wide query filtered client-side.

### Risk: Unstructured field extraction brittleness

**Risk**: Extracting replica counts and images from `*unstructured.Unstructured` using nested field accessors is fragile across K8s API versions.

**Mitigation**: Field paths used (`spec.replicas`, `status.readyReplicas`, `spec.template.spec.containers[0].image`) are stable across all supported K8s versions (1.24+). Safe accessors (`NestedInt64`, `NestedString`) return zero values on missing fields rather than panicking. Wide info is best-effort — missing fields result in `-` display, not errors.

### Risk: Exit code 2 breaking existing scripts

**Risk**: Scripts that check `$? -eq 0` to mean "status command succeeded" will now get exit 2 when resources exist but are not ready.

**Mitigation**: Intentional behavioral change. Document in release notes. Aligns with kubectl conventions.

### Trade-off: Component from inventory vs. live labels

Sourcing `Component` from the inventory (`InventoryEntry.Component`) means the component shown reflects the apply-time assignment, not any label that may exist on the live resource. If a resource was patched outside OPM and its label changed, the inventory value would differ.

Accepted because: the inventory is the ground truth for OPM-managed resources. Label-based discovery no longer exists, and the inventory component is always present (no missing-label edge case).

### Trade-off: Verbose pod details below table vs. inline

Rendering pod details as a separate block below the table means the pod info is visually separated from its parent row.

Accepted because: inline sub-rows would require either a custom table renderer or abandoning `output.Table` entirely. The separate-block approach is simpler, tested by the existing `FormatTransformerMatchVerbose` pattern.

### Trade-off: Removing ModuleID/ReleaseID from StatusResult

The old `ModuleID` and `ReleaseID` fields (UUIDs) are replaced by human-readable `ReleaseName` and `Version`. Users who relied on the UUID fields in JSON/YAML output will see them disappear.

Accepted because: UUIDs are still on the inventory Secret and accessible via `kubectl`. Status should surface human-readable information.

## Open Questions

None — all decisions are resolved. Future enhancements (drift summary, FQN display via annotations) are explicitly deferred and tracked as separate changes.
