## Context

`opm mod diff` compares rendered resources (from the CUE pipeline) against live Kubernetes objects fetched via the dynamic client. The comparison in `internal/kubernetes/diff.go` passes both objects directly to `dyffComparer.Compare()`, which marshals both to YAML and runs `dyff.CompareInputFiles`. There is no filtering — the live object includes everything the API server added: `managedFields`, `uid`, `resourceVersion`, `creationTimestamp`, `generation`, `status`, defaulted fields, and normalized values.

The `Diff()` function at `diff.go:177` iterates rendered resources, fetches each live counterpart via `fetchLiveState()`, then calls `comparer.Compare(obj, live)` at line 213. This is where projection needs to be inserted.

The `comparer` interface (`diff.go:90-94`) accepts `*unstructured.Unstructured` and returns a diff string. The interface is clean and doesn't need to change — projection happens before the comparer sees the objects.

## Goals / Non-Goals

**Goals:**

- Eliminate server-managed noise from diff output so that `opm mod diff` immediately after `opm mod apply` (with no changes) shows "No differences found"
- Only show diffs for fields the module author defined in CUE — the same fields that `opm mod apply` would send to the cluster
- Keep the solution simple: pure Go map traversal, no new dependencies

**Non-Goals:**

- Handling value normalization (e.g., `cpu: 8` vs `cpu: 8000m`) — this is a separate concern and would require Kubernetes-aware quantity parsing. Accept it as a known limitation for now.
- Detecting drift in fields OPM doesn't manage (server defaults, controller-managed fields) — explicitly out of scope per user decision
- Adding flags to toggle filtering behavior (e.g., `--show-all`) — YAGNI per Principle VII

## Decisions

### Decision 1: Project live object to rendered field paths (not managedFields)

**Choice**: Walk the rendered object to collect all field paths, then filter the live object to only contain those paths.

**Alternatives considered**:

- **managedFields-based filtering (Approach D)**: Parse `managedFields` entries for `manager: "opm"` using `fieldpath.Set.FromJSON()`. More "Kubernetes-native" but adds complexity: requires parsing the `FieldsV1` trie, handling `k:{}` key-based list elements, and breaks when new fields are added to rendered but not yet applied (they wouldn't be in `managedFields` yet). Also fails on first diff before any apply.
- **Server-side dry-run**: Send a dry-run apply, compare result against live. Most accurate but adds an API call per resource and requires apply permissions for diff.
- **Hardcoded strip list**: Remove a fixed set of known server fields. Simple but incomplete — doesn't handle defaults, requires ongoing maintenance as K8s adds new server fields.

**Rationale**: The rendered object IS the source of truth for "what OPM cares about." Using it as the projection template is simpler, handles new fields before first apply, and requires no external dependencies. It also naturally handles the edge case where a field was removed from the CUE definition — it won't appear in rendered, so it gets stripped from live, and no phantom diff appears.

### Decision 2: Two-layer filtering — metadata strip + field projection

**Choice**: Apply filtering in two stages:

1. **Metadata strip** (unconditional): Remove well-known server-only fields from live: `metadata.managedFields`, `metadata.uid`, `metadata.resourceVersion`, `metadata.creationTimestamp`, `metadata.generation`, and the top-level `status` block. These are never present in rendered output and are always noise.

2. **Field projection** (structural): Walk the rendered object's structure and retain only matching paths in the live object. This catches everything the metadata strip misses: defaulted fields (`podManagementPolicy`, `dnsPolicy`, etc.), controller-injected annotations, and any other server-side additions.

**Rationale**: The metadata strip is a cheap O(1) operation that eliminates the largest noise contributors (especially `managedFields` which can be hundreds of lines). Field projection then handles the long tail. Separating them makes testing clearer — you can verify each layer independently.

### Decision 3: Pre-filter in Diff(), not inside the comparer

**Choice**: Add a `projectLiveToRendered(rendered, live map[string]interface{}) map[string]interface{}` function. Call it in `Diff()` before `comparer.Compare()`.

**Alternative**: Wrap `dyffComparer` in a `filteredComparer` that does projection internally.

**Rationale**: Keeps the `comparer` interface focused on comparison only. The projection is a data concern, not a comparison concern. It's also easier to unit test — `projectLiveToRendered` is a pure function with no dependencies.

### Decision 4: Recursive map walk for projection

**Choice**: Implement projection as a recursive function that walks two `map[string]interface{}` trees in parallel. For each key in rendered:

- If the value is a `map[string]interface{}`, recurse into the corresponding map in live
- If the value is a `[]interface{}` (list), handle based on content: for associative lists (list of maps with identifying keys), match elements by their identifying fields; for scalar lists, keep the live list as-is
- For scalar values, keep the live value at that path

**List matching strategy**: For lists of maps (e.g., containers, ports, env vars), Kubernetes uses specific fields as keys (e.g., `name` for containers, `containerPort`+`protocol` for ports). Rather than hardcoding these, use a simpler heuristic: if both rendered and live list elements are maps, attempt to match by the `name` field first (covers the majority of K8s lists). If no `name` field, fall back to index-based matching. This handles containers, ports, volumes, volumeMounts, and env vars — which are the most common associative lists.

**Rationale**: Pure Go, no dependencies, straightforward to test with table-driven tests. The `name`-field heuristic covers the vast majority of real-world K8s resources without needing a schema-aware approach.

## Risks / Trade-offs

**[Value normalization still produces false diffs]** → Accept for now. Document as known limitation. Resource quantity normalization (`8` → `8000m`) and boolean normalization (`"true"` → `true`) will still appear. Module authors can work around this by using canonical forms in CUE. A future change could add quantity normalization as a separate concern.

**[List matching by `name` field may miss edge cases]** → Mitigated by falling back to index-based matching. The only lists where this could produce unexpected results are associative lists keyed by fields other than `name` (e.g., `ports` keyed by `containerPort`+`protocol`). In practice these are rare and the index fallback produces a reasonable diff even if not perfectly aligned.

**[Empty annotations map in diff]** → When rendered includes `metadata.annotations: {}` (empty map) but live has server-added annotations, the projection keeps the empty map from rendered and strips the server annotations from live. This is correct behavior — but if the rendered object has `annotations: {}` and the live object has no annotations at all, dyff may show a spurious diff for the empty map. Handle by stripping empty maps from both sides after projection.
