## Context

Every instance command that reads a deployed instance goes through one function, `inventory.DiscoverResourcesFromInventory` (`internal/inventory/discover.go`): one GET per inventory entry, eight in flight, results kept in inventory order. It sorts each entry into one of two buckets: live (the GET returned the object) or missing (NotFound). A third outcome, any other GET error, falls into neither bucket and is logged at debug level only. Its callers:

| Caller | Path | What the silent drop does |
| --- | --- | --- |
| `query.ResolveInventory` (`internal/workflow/query/status.go:63`) | `instance delete`, `status`, `tree`, `events` | delete never sees the entry and removes the `ModuleInstance` (orphan); the others show fewer resources |
| `query.EvaluateInstanceHealth` (`internal/workflow/query/list.go:57`) | `instance list` | the entry counts toward neither total nor ready |
| `runInstanceDiff` (`internal/cmd/instance/diff.go:108`) | `instance diff` | orphan detection never considers the entry |

`kubernetes.Delete` already has the rule this change extends: an in-loop re-read that fails with anything but NotFound is a per-resource error (`DeleteResult.Errors`), `executeInstanceDelete` then keeps the `ModuleInstance`, and `reportInstanceDelete` exits 1 (`N resource(s) failed to delete`, or `could not be checked` on a dry run). The gap is only that discovery errors never reach it.

## Goals / Non-Goals

**Goals:**

- No tracked resource disappears from a command's view because its read failed.
- `instance delete` never removes the `ModuleInstance` while an object it tracks could not be read.
- `status` and `list` never report `Ready` for an instance with an unread object.

**Non-Goals:**

- Classifying the read error (transient vs terminal) or retrying it. A later library change types registry and cluster errors; this change treats every non-NotFound read error alike, as the in-loop re-read already does.
- Refusing the whole delete up front. See Research & Decisions.
- A new health status or aggregate value (`Degraded`, `Unreadable`).
- The operator-owned delete path and the operator itself.
- Dropping the function's always-nil error result. Every call site changes anyway, but its removal is unrelated cleanup.

## Decisions

### 1. Discovery returns a third bucket

```go
// UnreadableEntry is an inventory entry whose live object could not be read:
// its GET failed with an error other than NotFound.
type UnreadableEntry struct {
	Entry InventoryEntry
	Err   error
}

// DiscoverResourcesFromInventory ... Every entry lands in exactly one of
// live, missing or unreadable, each in inventory order.
func DiscoverResourcesFromInventory(ctx context.Context, client *kubernetes.Client, inv *Record) (
	live []*unstructured.Unstructured, missing []InventoryEntry, unreadable []UnreadableEntry, err error)
```

The goroutine records `entryResult{readErr: getErr}` instead of returning silently; the collection loop appends it to `unreadable`. The debug log line stays. The error result stays always-nil, as today.

`query.ResolveInventory` returns the slice as a fifth result, between `missing` and `err`, and does not print it: delete reports it differently from the read-only commands.

A shared helper prints the read-only warning, so the wording is the same in every command:

```go
// WarnUnreadable logs one warning per tracked resource that could not be read.
func WarnUnreadable(logger *log.Logger, unreadable []inventory.UnreadableEntry)
// line: "could not read tracked resource" kind=<Kind> namespace=<ns> name=<name> error=<err>
```

### 2. Delete: an unreadable entry is a per-resource failure

`internal/kubernetes` cannot import `internal/inventory` (the import runs the other way), so `Delete` takes its own type:

```go
// UnreadableResource is a tracked resource whose discovery read failed.
type UnreadableResource struct {
	Group, Kind, Namespace, Name string
	Err                          error
}

type DeleteOptions struct {
	// ...
	// Unreadable lists tracked resources the caller's discovery could not
	// read. Each is a per-resource error (DeleteResult.Errors), except a
	// protected kind, which is left behind as it would be if read.
	Unreadable []UnreadableResource
}
```

`inventory.UnreadableEntry` gets `func (u UnreadableEntry) Resource() kubernetes.UnreadableResource`, and `executeInstanceDelete` passes the converted slice. `executeInstanceDelete` and `deleteResolvedInstance` take the unreadable slice as a new parameter; `runInstanceDelete` takes it from `ResolveInventory`.

In `Delete`, before the live loop:

- protected kind (`IsProtectedKind(Group, Kind)`): append to `LeftBehind` with `ProtectedKindReason`, the same outcome `checkDeletable` gives a readable one. Delete would never delete it, so failing on it would block every re-run for nothing.
- otherwise: log `reading <Kind>/<name>: <err>` at warn level (the in-loop wording) and append a `resourceError`.

The no-resources check becomes `len(resources) == 0 && len(opts.Unreadable) == 0 && !opts.InventoryRecordExists`. The loop body moves into a helper if gocyclo needs it.

Nothing else in `executeInstanceDelete` changes: `len(deleteResult.Errors) > 0` already keeps the `ModuleInstance`, and `reportInstanceDelete` already exits 1. One addition: on a real run with errors, `reportInstanceDelete` prints

```text
The ModuleInstance was kept, so it still tracks these resources.
Fix the cause (for example missing RBAC) and re-run; re-running is safe.
```

through `output.Details`, before returning the exit error. It serves the in-loop errors too, which were silent about the kept instance.

The operator-owned branch ignores the slice: it deletes only the `ModuleInstance`, and the operator prunes with its own credentials.

### 3. Status: an unreadable resource is an `Unknown` row

`kubernetes.StatusOptions` gains `UnreadableResources []UnreadableResource`, filled by `query.BuildStatusOptions` (a new parameter). `GetInstanceStatus` appends each as a row with `Status: HealthUnknown`, `Age: "<unknown>"`, sets `allReady = false`, and counts it in the no-resources check. `HealthUnknown` already means "the health state could not be determined", it is not healthy, so the aggregate becomes `NotReady`, the summary counts the row as not ready, and `PrintInstanceStatus` exits 2 through the existing branch. The structured outputs carry the row like any other. `runInstanceStatus` calls `WarnUnreadable` before printing, so the read error itself reaches the user.

### 4. List: an unreadable resource counts as not ready

`QuickInstanceHealth(resources, missingCount)` becomes `QuickInstanceHealth(resources, unhealthyCount)` with the doc comment saying it counts tracked resources that are missing or could not be read; `EvaluateInstanceHealth` passes `len(missing) + len(unreadable)`. An instance with an unreadable resource therefore shows `NotReady (r/t)`. `EvaluateInstanceHealth` runs one goroutine per instance and prints at warn level, outside the `logDiscoveryFailures` switch:

```text
WARN instance "demo" in "apps": could not read 2 tracked resource(s); run 'opm instance status' for details
```

One line per instance keeps a cluster-wide list readable; `status` names each resource.

### 5. Diff, tree, events: warn

`runInstanceDiff` calls `WarnUnreadable` (through the instance logger) and then one line: `orphan detection could not check N tracked resource(s)`. `tree` and `events` call `WarnUnreadable` after `ResolveInventory`. Their output is otherwise unchanged and their exit codes do not change: they still describe what they could read, and now say what they could not. When `tree` can read none of the tracked resources it prints the warnings and then exits 5 (`no resources found`) as before. `status` exits 2 in that case because its verdict is health, and an unread resource is not healthy; `tree` only draws what exists, and the warnings name the cause, so it keeps its not-found code rather than borrowing status's.

### Exit codes

| Command | Unreadable tracked resource | Exit |
| --- | --- | --- |
| `instance delete` (real run or dry run) | per-resource failure, `ModuleInstance` kept | 1 |
| `instance status` | `Unknown` row, aggregate `NotReady` | 2 |
| `instance list`, `diff`, `tree`, `events` | warning only | unchanged (tree with nothing readable: 5) |

### Example output

```text
$ opm instance delete demo -n apps --force
INFO deleting resources in namespace "apps"
WARN reading ConfigMap/settings: configmaps "settings" is forbidden: User "dev" cannot get resource "configmaps" in the namespace "apps"
INFO Deployment/apps/web          deleted
WARN 1 resource(s) had errors
ERRO ConfigMap/settings in apps: configmaps "settings" is forbidden: ...
The ModuleInstance was kept, so it still tracks these resources.
Fix the cause (for example missing RBAC) and re-run; re-running is safe.
Error: 1 resource(s) failed to delete
$ echo $?
1
```

## Research & Decisions

### Refuse the whole delete, or delete what can be read

**Context**: the plan entry says delete "refuses, keeps the ModuleInstance and exits non-zero with each entry printed". That can mean refusing before any delete, or deleting the readable resources and keeping the `ModuleInstance`.

**Explored**: `kubernetes.Delete` and `executeInstanceDelete` at origin/main 5180cad1; the main `deploy` spec requirement "Instance delete re-checks live ownership before each delete" (scenario "Read error keeps the ModuleInstance"); cli issue #283, which names "keep the `ModuleInstance` when any exist" as the fix.

**Options considered**:

1. Refuse up front: nothing is deleted while any entry is unreadable. Simple to state, but a persistent denial on one object blocks the delete of every other object, and it gives a discovery read error a different outcome from the same error one call later in the re-read.
2. Per-resource failure: delete the readable resources, keep the `ModuleInstance`, exit 1. Matches the existing re-read rule, so the outcome does not depend on which of two reads failed. The re-run is safe: the kept inventory still tracks the unread objects, and the deleted ones come back as missing.

**Decision**: option 2.

**Rationale**: it closes the orphan (the record survives) with one rule for every read error, and #283 asks for exactly "keep the ModuleInstance". The plan's "refuses" is read as "refuses to delete the ModuleInstance", which is also how the research entry words it.

### How to show "degraded"

**Context**: the plan asks status and list to "mark the instance degraded". The CLI has no `Degraded` health value; the aggregate is `Ready`, `NotReady` or `Unknown`.

**Options considered**:

1. A new `HealthUnreadable` row status or `Degraded` aggregate: precise, but new vocabulary in the JSON/YAML output and the colour map, for one case.
2. `HealthUnknown` rows and the existing `NotReady` aggregate: `Unknown` is documented as "could not be determined", and not-ready already drives exit 2 and `NotReady (r/t)`.

**Decision**: option 2. "Degraded" is delivered as not ready, plus the warning that names the read error.

### Tree and events

**Context**: the plan names status, list and diff. Tree and events read through the same `ResolveInventory`, whose signature this change extends.

**Decision**: they call the same warning helper. Leaving them silent would keep the bug in two commands whose call sites the change already edits; they gain one line each and no new behaviour beyond the warning.
