## Why

`opm mod diff` currently compares raw rendered manifests against raw live Kubernetes objects. The API server decorates live objects with server-managed fields (`managedFields`, `uid`, `resourceVersion`, `creationTimestamp`, `status`), applies defaults (`podManagementPolicy: OrderedReady`, `dnsPolicy: ClusterFirst`, `terminationGracePeriodSeconds: 30`), and normalizes values (`cpu: 8` → `cpu: 8000m`). This makes every resource appear "modified" even immediately after `opm mod apply` with zero changes — the diff output is unusable.

## What Changes

- **Add field projection to diff comparison**: Before comparing rendered vs live objects, project (filter) the live object to only contain field paths that exist in the rendered object. Fields present on live but absent from rendered are server-managed noise and get stripped before comparison.
- **Strip server-generated metadata unconditionally**: Remove `metadata.managedFields`, `metadata.uid`, `metadata.resourceVersion`, `metadata.creationTimestamp`, `metadata.generation`, and `status` from the live object before comparison, as these are never present in rendered output.
- Diff output after this change shows **only fields the module author defined** — changes that would actually be applied by `opm mod apply`.

## Capabilities

### New Capabilities

- `diff-field-projection`: Filtering logic that projects live Kubernetes objects to only the field paths present in the rendered manifest, eliminating server-managed noise from diff output.

### Modified Capabilities

- `mod-diff`: The existing diff spec needs a new requirement covering field filtering behavior — specifically that server-managed fields and API-server defaults MUST NOT appear in diff output.

## Impact

- **Code**: `internal/kubernetes/diff.go` — the `Compare` method (or a preprocessing step before it) needs field projection logic
- **SemVer**: PATCH — this is a bug fix (diff showing incorrect "modifications"). No API, flag, or behavioral contract changes. Output becomes more accurate, not different in structure.
- **Dependencies**: No new dependencies. The projection is pure map traversal over `map[string]interface{}`.
- **Risk**: Low. The comparer interface is already abstracted and well-tested. Projection is additive preprocessing.
