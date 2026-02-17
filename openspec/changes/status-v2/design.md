## Context

The current `opm mod status` implementation lives across three files:

- `internal/cmd/mod_status.go` — command wiring, flag parsing, watch loop
- `internal/kubernetes/status.go` — `GetModuleStatus()`, `StatusResult`, formatting
- `internal/kubernetes/health.go` — `evaluateHealth()`, health status types

The data flow today:

```text
cmd/mod_status.go                kubernetes/status.go              kubernetes/discovery.go
┌──────────────┐                ┌──────────────────┐              ┌────────────────────┐
│ runStatus()  │───────────────▶│ GetModuleStatus() │─────────────▶│ DiscoverResources()│
│              │                │                   │              │                    │
│ • validate   │                │ • discover        │              │ • label selector   │
│   flags      │                │ • evaluate health │              │ • scan all APIs    │
│ • resolve    │                │ • compute age     │              │ • filter by labels │
│   k8s config │                │ • aggregate       │              │ • exclude owned    │
│ • create     │                │                   │              │                    │
│   client     │                │ Returns:          │              │ Returns:           │
│              │                │ StatusResult{     │              │ []*Unstructured    │
│              │                │   Resources[]     │              └────────────────────┘
│              │                │   AggregateStatus │
│              │                │   ModuleID        │
│              │                │   ReleaseID       │
│              │                │ }                 │
│              │◀───────────────│                   │
│ format +     │                └──────────────────┘
│ print        │
└──────────────┘
```

Key constraints from the current implementation:

1. **Status is source-free** — it only queries the cluster via labels, never re-renders the module. This is intentional and must be preserved.
2. **Discovery returns `*unstructured.Unstructured`** — the raw K8s objects are available, so we can extract any field from `.spec`, `.status`, or `.metadata.labels` without additional API calls.
3. **Labels already on resources** include component name (`component.opmodel.dev/name`) and version (`module-release.opmodel.dev/version`) — these exist but are not read by status today.
4. **The table uses `output.NewTable()`** which renders via `lipgloss/table`. Cell content is currently raw strings — color must be applied by rendering styled strings into cell values before passing to the table.
5. **Exit codes are defined in `internal/errors/errors.go`** with existing constants: `ExitSuccess(0)`, `ExitGeneralError(1)`, `ExitValidationError(2)`, `ExitConnectivityError(3)`, `ExitPermissionDenied(4)`, `ExitNotFound(5)`.

## Goals / Non-Goals

**Goals:**

- Add component, version, and resource summary to status output using labels already on cluster resources
- Provide wide format (`-o wide`) with workload-specific columns extracted from the unstructured objects already returned by discovery
- Provide verbose mode (`--verbose`) with pod-level diagnostics for unhealthy workloads via ownerReference walking
- Define semantic exit codes that distinguish "command failed" from "resources unhealthy"
- Apply color to status output using the existing lipgloss palette

**Non-Goals:**

- Drift detection (handled separately by `mod diff`)
- Event aggregation (handled separately by `mod events`)
- Module source or registry queries — status remains source-free
- Changes to the discovery mechanism itself
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

The old `ModuleID` and `ReleaseID` fields (raw UUIDs) are removed from the struct. They provided no user value. If needed for debugging, they remain accessible via `-o json` on the raw resource objects or via the labels directly.

### Decision 2: Component extraction from labels

**Context**: The `component.opmodel.dev/name` label is defined as `LabelComponentName` in `discovery.go` and is stamped by CUE transformers. It's already on the discovered resources.

**Decision**: Read the component label from each resource's labels during status evaluation. No new API calls needed — the unstructured objects from discovery already have their full label set.

```go
// In GetModuleStatus, when building resourceHealth:
labels := res.GetLabels()
component := labels[LabelComponentName] // "" if not present
```

Resources without the label get `""` which renders as `-` in the table.

### Decision 3: Version extraction from labels

**Context**: The `module-release.opmodel.dev/version` label is stamped by the CUE release overlay at render time. It's already on every managed resource.

**Decision**: Extract version from the first resource's labels, same pattern as the existing module ID / release ID extraction. Add a new label constant:

```go
const labelReleaseVersion = "module-release.opmodel.dev/version"
```

### Decision 4: Wide format — extracting workload details from unstructured objects

**Context**: Discovery already returns full `*unstructured.Unstructured` objects. The `.spec` and `.status` fields contain replica counts, container images, PVC capacity, and Ingress hosts. We need to extract these without importing typed K8s API structs (to stay consistent with the unstructured approach).

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

All of these fields are available on the unstructured objects already returned by `DiscoverResources()`. No additional API calls required for wide mode.

### Decision 5: Verbose mode — pod-level diagnostics via ownerReference walking

**Context**: When a Deployment/StatefulSet/DaemonSet is `NotReady`, the user needs to know *why*. The information is on the pods, which are children of the workload via ownerReferences. We need to query pods for unhealthy workloads only.

**Decision**: For each workload with `Status == healthNotReady`, list pods in the same namespace and filter by ownerReference chain. The walking strategy differs by workload type:

```text
Deployment  → owns ReplicaSet(s) → owns Pod(s)     (2-hop)
StatefulSet → owns Pod(s) directly                   (1-hop)
DaemonSet   → owns Pod(s) directly                   (1-hop)
```

For Deployments (2-hop), we need to find the active ReplicaSet first, then find its pods. For StatefulSet/DaemonSet (1-hop), pods are direct children.

```text
┌─────────────────────────────────────────────────────────────────┐
│  Verbose Mode: Pod Discovery Flow (Deployment)                   │
│                                                                   │
│  1. Deployment is NotReady                                        │
│  2. List Pods in namespace with label selector matching           │
│     the Deployment's .spec.selector.matchLabels                   │
│  3. For each Pod:                                                 │
│     a. Extract phase from status.phase                            │
│     b. Check containerStatuses for waiting/terminated reasons     │
│     c. Sum restartCount across all containers                     │
│     d. Check conditions for Ready=True                            │
│  4. Build podInfo for each pod                                    │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

**Why label selector instead of ownerReference walking**: Using the Deployment's `.spec.selector.matchLabels` to list pods directly is simpler than the 2-hop ownerReference chain (Deployment → find active ReplicaSet → find RS's pods). The label selector approach is a single API call and catches all pods regardless of which ReplicaSet owns them. For StatefulSets and DaemonSets, the same approach works — their `.spec.selector.matchLabels` directly identifies their pods.

New function:

```go
func listWorkloadPods(ctx context.Context, client *Client, resource *unstructured.Unstructured) ([]podInfo, error)
```

This function:

1. Reads `.spec.selector.matchLabels` from the workload
2. Lists pods in the same namespace with that label selector
3. Extracts phase, conditions, container statuses, restart counts
4. Returns `[]podInfo`

**Performance**: This makes one additional `core/v1 Pods` list call per unhealthy workload. Since verbose mode is opt-in and typically there are few unhealthy workloads, this is acceptable. If a release has 10 workloads and 2 are unhealthy, that's 2 extra API calls.

### Decision 6: StatusOptions changes — passing mode to the status layer

**Context**: `GetModuleStatus` currently receives `StatusOptions` with namespace, release selector, and output format. It needs to know whether to compute wide/verbose data.

**Decision**: Add mode fields to `StatusOptions`:

```go
type StatusOptions struct {
    Namespace    string
    ReleaseName  string
    ReleaseID    string
    OutputFormat output.Format
    Wide         bool  // compute wide info (replicas, images)
    Verbose      bool  // compute verbose info (pod details for unhealthy workloads)
}
```

`GetModuleStatus` populates `wideInfo` and `verboseInfo` on each resource only when the corresponding flag is true. Default mode skips the extra work entirely.

### Decision 7: Exit codes — mapping to existing constants

**Context**: The proposal specifies exit 0 (healthy), exit 1 (error), exit 2 (not ready), exit 3 (no resources). The existing exit code constants are: `ExitSuccess(0)`, `ExitGeneralError(1)`, `ExitValidationError(2)`, `ExitConnectivityError(3)`, `ExitPermissionDenied(4)`, `ExitNotFound(5)`.

**Options considered**:

1. **Reuse existing constants** — map "not ready" to `ExitValidationError(2)` and "no resources" to `ExitNotFound(5)`
2. **Add new status-specific constants** — `ExitUnhealthy`, `ExitNoResources`
3. **Reuse existing constants with semantic meaning per command** — the numbers are per-contract, and commands can define their own meanings

**Decision**: Option 1 — reuse existing constants. The semantics align well enough:

```text
All healthy     → ExitSuccess (0)          — existing
General error   → ExitGeneralError (1)     — existing, already used
Resources not   → ExitValidationError (2)  — "the state doesn't validate as healthy"
  ready
No resources    → ExitNotFound (5)         — existing, semantically correct
  found
```

This avoids adding new constants. `ExitValidationError` is a reasonable fit — the health check is a form of validation ("does the deployed state match the desired state?"). `ExitNotFound` already exists and is the correct semantic for "nothing matched the selector."

The `--ignore-not-found` flag overrides `ExitNotFound(5)` to `ExitSuccess(0)`, preserving current behavior.

### Decision 8: Color rendering approach

**Context**: The `output.Table` renders cells as raw strings. Lipgloss color is applied by rendering styled strings *before* passing them as cell values. The `lipgloss/table` package handles width calculation correctly for ANSI-escaped strings.

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
    case "NotReady":
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

**Decision**: Render the header as a block of key-value lines with styled values, printed before the table. The header is independent of the table — it's plain `output.Println` calls.

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

Example output (colors represented by annotations):

```text
Release:    jellyfin-media              ← cyan
Version:    1.2.0
Namespace:  media                       ← cyan
Status:     Ready                       ← green  (or "NotReady" in red)
Resources:  6 total (6 ready)
```

### Decision 10: Verbose output — inline pod details below table rows

**Context**: Verbose mode needs to show pod details indented below the parent workload row. The `output.Table` renders a fixed-column table — it doesn't support sub-rows or indented detail lines.

**Options considered**:

1. **Render the table first, then append pod details after each row** — complex, requires tracking line positions
2. **Skip the table for verbose mode and render manually** — full control but loses table alignment
3. **Render the table normally, then print pod details as separate indented blocks keyed by resource name** — simple, clear separation

**Decision**: Option 3 — render the standard table, then for each unhealthy workload, print an indented pod detail block below the table. This keeps the table clean and the pod details scannable.

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

This is simpler to implement, avoids fighting the table renderer, and produces output that's easy to parse visually. The detail blocks only appear for unhealthy workloads when `--verbose` is set.

### Decision 11: `StatusOptions.OutputFormat` and `-o wide`

**Context**: The `-o` flag currently accepts `table`, `yaml`, `json`. We're adding `wide` as a table variant, not a new format.

**Decision**: `wide` is a modifier on the table format, not a separate output format. The `-o` flag values remain `table`, `yaml`, `json`. The `wide` behavior is activated by `-o wide` which sets `StatusOptions.Wide = true` and uses `FormatTable` with the extra columns. This matches the kubectl pattern where `-o wide` is a table variant.

Updated flag validation:

```go
// In NewModStatusCmd, validate output format:
switch outputFlag {
case "table", "yaml", "json", "wide":
    // valid
default:
    // error
}
```

When `outputFlag == "wide"`:

- `opts.OutputFormat = output.FormatTable`
- `opts.Wide = true`

### Decision 12: File organization — where new code lives

**Context**: The change adds extraction functions, pod listing, and styling helpers. These need to go in the right packages per Principle II (Separation of Concerns).

**Decision**:

```text
internal/
├── cmd/
│   └── mod_status.go          MODIFIED: new flags, exit code mapping, wide/verbose wiring
├── kubernetes/
│   ├── status.go              MODIFIED: StatusResult changes, wide/verbose data population
│   ├── health.go              UNCHANGED
│   ├── discovery.go           UNCHANGED
│   ├── wide.go                NEW: extractWideInfo() per-kind extraction
│   └── pods.go                NEW: listWorkloadPods(), podInfo, pod status extraction
└── output/
    └── styles.go              MODIFIED: FormatHealthStatus(), FormatComponent()
```

- `wide.go` keeps the per-kind unstructured field extraction isolated
- `pods.go` keeps the pod-listing and status extraction logic separate from the main status flow
- No changes to `discovery.go` or `health.go` — they stay focused on their responsibilities

## Risks / Trade-offs

### Risk: Pod listing performance in verbose mode

**Risk**: If a release has many unhealthy workloads, verbose mode makes one pod-list API call per unhealthy workload, which could be slow on large clusters.

**Mitigation**: Verbose mode is opt-in (`--verbose`). The call is only made for `NotReady` workloads, not all workloads. In practice, most releases have 1-3 workloads, and only the unhealthy ones trigger pod listing. If performance becomes an issue, we could batch pod listing with a single namespace-wide query and filter client-side — but that's a future optimization.

### Risk: Unstructured field extraction brittleness

**Risk**: Extracting replica counts and images from `*unstructured.Unstructured` using nested field accessors is fragile — field paths could differ across K8s API versions.

**Mitigation**: The field paths used (`spec.replicas`, `status.readyReplicas`, `spec.template.spec.containers[0].image`) are stable across all supported K8s versions (1.24+). They're part of the core API contract. We use safe accessors (`NestedInt64`, `NestedString`) that return zero values on missing fields rather than panicking. Wide info is best-effort — missing fields result in `-` display, not errors.

### Risk: Exit code 2 breaking existing scripts

**Risk**: Scripts that check `$? -eq 0` to mean "status command succeeded" will now get exit 2 when resources exist but are not ready.

**Mitigation**: This is an intentional behavioral change. The old behavior (exit 0 for "command ran, resources unhealthy") was misleading for CI/CD. Document the change in the release notes. The exit code semantics align with kubectl conventions where non-zero indicates the checked condition failed.

### Trade-off: Verbose pod details below table vs. inline

Rendering pod details as a separate block below the table (Decision 10) means the pod info is visually separated from its parent row. In a table with many resources, you have to scroll between the table and the detail blocks.

Accepted because: inline sub-rows would require either a custom table renderer or abandoning `output.Table` entirely. The separate-block approach is simpler, tested by the existing `FormatTransformerMatchVerbose` pattern, and works well for the common case (1-2 unhealthy workloads).

### Trade-off: Removing ModuleID/ReleaseID from StatusResult

The old `ModuleID` and `ReleaseID` fields (UUIDs) are replaced by human-readable `ReleaseName` and `Version`. Users who relied on the UUID fields in JSON/YAML output will see them disappear.

Accepted because: These UUIDs are still on the resources' labels and accessible via `kubectl`. The status command should surface human-readable information. If machine-readable IDs are needed, they can be added back as optional fields later.

## Open Questions

None — all decisions are resolved based on the exploration and discussion. Future enhancements (drift summary, FQN display via annotations) are explicitly deferred and tracked as separate changes.
