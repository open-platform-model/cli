## Context

The CLI currently has no way to list deployed module releases. Users must know the exact release name or ID to run `status`, `delete`, or `diff`. The inventory system already stores per-release state as labeled Kubernetes Secrets, so all the data needed for a list view already exists — we just lack the discovery function and command to surface it.

The existing inventory CRUD (`GetInventory`, `FindInventoryByReleaseName`) is single-release scoped. The health evaluation logic in `kubernetes/health.go` is unexported. Both need small expansions to support the list command.

## Goals / Non-Goals

**Goals:**

- List all deployed module releases in a namespace with health status
- Support all-namespaces listing with `-A`
- Parallel health evaluation to keep latency reasonable
- Reuse existing inventory labels and health logic — no new discovery mechanisms
- Support table, wide, json, yaml output formats

**Non-Goals:**

- Sorting flags (`--sort-by`) — YAGNI, default sort by name is sufficient
- Filtering by module name — can be added later if needed
- Watch mode — not needed for a list command
- Offline/cached listing — if the cluster is unreachable, the Secret is unreachable too

## Decisions

### Decision: Always show status column

**Choice**: Status is always shown, not behind a flag.

**Rationale**: The first thing users want to know is "are my modules healthy?" — this is why you run `list` in the first place. The cost is N GETs per release to discover live resources, but this is bounded and parallelizable. `kubectl get pods` always shows status; we follow that convention.

**Alternative considered**: `--health` opt-in flag to keep default list fast. Rejected because the extra latency (bounded parallel GETs) is acceptable for a human-facing command, and omitting status makes the output significantly less useful.

### Decision: Namespace column hidden unless -A

**Choice**: The NAMESPACE column is shown only when `--all-namespaces` / `-A` is used.

**Rationale**: Matches `kubectl` convention. When listing in a single namespace, the namespace is implicit from the flag/config. When listing across all namespaces, it becomes essential context.

### Decision: ListInventories in the inventory package

**Choice**: Add `ListInventories(ctx, client, namespace)` to `internal/inventory/` rather than inlining the K8s list call in the command.

**Rationale**: Follows the existing pattern where all inventory Secret operations live in `internal/inventory/`. The command layer orchestrates; it doesn't implement K8s calls directly (Principle II: Separation of Concerns).

### Decision: Export health primitives

**Choice**: Export `evaluateHealth` as `EvaluateHealth`, export `healthStatus` as `HealthStatus` and its constants. Add `QuickReleaseHealth` aggregation function.

**Rationale**: The status command, tree command, and now list command all need health evaluation. Exporting the existing logic avoids duplication. `QuickReleaseHealth` is a thin aggregation wrapper — it doesn't duplicate evaluation logic, just counts results.

**Alternative considered**: Add a `GetReleaseSummary` function to `kubernetes/` that wraps the full `GetReleaseStatus` flow. Rejected because it couples list to the full status pipeline (wide info, pod diagnostics) when we only need the aggregate.

### Decision: Parallel resource discovery with bounded concurrency

**Choice**: Fetch resources for all releases in parallel using a bounded goroutine pool (e.g., 5 workers).

**Rationale**: Sequential fetching would make the command O(releases * resources) in latency. With 5 releases averaging 8 resources each, sequential takes ~5s. Parallel with 5 workers brings this to ~1-2s. The bound prevents overwhelming the API server.

### Decision: Status format "Ready (N/N)"

**Choice**: Display status as `Ready (5/5)` or `NotReady (3/5)` combining aggregate and counts.

**Rationale**: Gives the user both the aggregate state and the proportion in a single glance. More informative than just "Ready" or "NotReady" alone.

## Risks / Trade-offs

**[Performance at scale]** With many releases (50+), even parallel discovery may be slow.
Mitigation: Bounded worker pool prevents API server overload. The `list` command is human-facing and infrequent; 5-10s is acceptable for large clusters.

**[Health evaluation is snapshot]** Status is point-in-time. Resources may change health between list and follow-up action.
Mitigation: This is inherent to any non-watching command. Users can run `opm mod status` for detailed per-release health.

**[Exporting health types widens the API surface]** Other packages can now depend on `HealthStatus` type.
Mitigation: The type is a simple string enum with stable values. Breaking changes are unlikely.
