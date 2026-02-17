## Context

The `internal/kubernetes` package has grown to 8 source files (`apply.go`, `delete.go`, `diff.go`, `discovery.go`, `status.go`, `client.go`, `health.go`, `warnings.go`). During this growth, several utility patterns were duplicated across files rather than shared:

- `gvrFromObject` (apply.go:163-171) and `gvrFromUnstructured` (delete.go:137-145) are identical functions
- The namespaced-vs-cluster-scoped dynamic client branching appears 4 times across 3 files
- `expandTilde` is implemented identically in both `config/paths.go:35-55` and `kubernetes/client.go:189-209`

Additionally, the package has accumulated export inconsistencies (some result types exported, others not) and YAGNI option struct fields that are defined but never read, violating Principle VII.

The `cmdutil` package has a redundant `K8sClientOpts` struct that mirrors `kubernetes.ClientOptions` field-for-field, adding indirection without decoupling.

## Goals / Non-Goals

**Goals:**

- Eliminate duplicated code within `internal/kubernetes/`
- Create a shared `ResourceClient` method to DRY up the namespaced-vs-cluster-scoped branching
- Consolidate `expandTilde` into a single source of truth
- Make export patterns consistent across the package
- Remove YAGNI option fields
- Remove redundant type indirection in `cmdutil`
- Fix stale documentation and integration tests
- All changes are mechanical refactoring — zero behavior changes

**Non-Goals:**

- Decoupling `internal/kubernetes` from `internal/build` (the `[]*build.Resource` parameter) — deferred to a separate change
- Splitting `internal/output` into sub-packages — deferred, higher risk
- Reducing `internal/output` coupling in kubernetes operations (logging during apply/delete) — deferred
- Creating a shared test helper package (`internal/testutil`) — deferred

## Decisions

### Decision 1: Create `resource.go` with shared GVR and client helpers

Create `internal/kubernetes/resource.go` containing all shared resource utilities.

**What moves from `apply.go` to `resource.go`:**

- `gvrFromObject` (lines 163-171) — renamed to `gvrFromUnstructured`
- `knownKindResources` map (lines 175-213) — the 33-entry Kind→plural lookup table
- `kindToResource` (lines 215-222)
- `heuristicPluralize` (lines 224-238)
- `isVowel` (line 240-242)

**What is deleted from `delete.go`:**

- `gvrFromUnstructured` (lines 137-145) — now in `resource.go`

**What is new in `resource.go`:**

```go
// ResourceClient returns the appropriate dynamic resource client for the given
// GVR and namespace. If namespace is empty, returns a cluster-scoped client.
func (c *Client) ResourceClient(gvr schema.GroupVersionResource, ns string) dynamic.ResourceInterface {
    if ns != "" {
        return c.Dynamic.Resource(gvr).Namespace(ns)
    }
    return c.Dynamic.Resource(gvr)
}
```

**What stays in `apply.go`:**

- `boolPtr` (line 244-246) — only used by `applyResource`, not shared

**Why a new file:** The function is used by `apply.go`, `delete.go`, and `diff.go`. Placing it in any one of those creates a non-obvious cross-file dependency. A dedicated `resource.go` file makes the shared nature explicit.

**Why `gvrFromUnstructured` over `gvrFromObject`:** The name `gvrFromUnstructured` is more descriptive — it clarifies the parameter type.

**Required imports for `resource.go`:**

```go
import (
    "strings"

    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
    "k8s.io/apimachinery/pkg/runtime/schema"
    "k8s.io/client-go/dynamic"
)
```

**Caller updates — `apply.go` `applyResource` function (lines 104-160):**

Before (lines 110-120, GET branch):
```go
if ns != "" {
    existing, err := client.Dynamic.Resource(gvr).Namespace(ns).Get(ctx, obj.GetName(), metav1.GetOptions{})
    if err == nil {
        existingVersion = existing.GetResourceVersion()
    }
} else {
    existing, err := client.Dynamic.Resource(gvr).Get(ctx, obj.GetName(), metav1.GetOptions{})
    if err == nil {
        existingVersion = existing.GetResourceVersion()
    }
}
```

After:
```go
existing, err := client.ResourceClient(gvr, ns).Get(ctx, obj.GetName(), metav1.GetOptions{})
if err == nil {
    existingVersion = existing.GetResourceVersion()
}
```

Before (lines 139-147, PATCH branch):
```go
if ns != "" {
    result, patchErr = client.Dynamic.Resource(gvr).Namespace(ns).Patch(
        ctx, obj.GetName(), types.ApplyPatchType, data, patchOpts,
    )
} else {
    result, patchErr = client.Dynamic.Resource(gvr).Patch(
        ctx, obj.GetName(), types.ApplyPatchType, data, patchOpts,
    )
}
```

After:
```go
result, patchErr = client.ResourceClient(gvr, ns).Patch(
    ctx, obj.GetName(), types.ApplyPatchType, data, patchOpts,
)
```

Also change line 105 from `gvrFromObject(obj)` to `gvrFromUnstructured(obj)`.

**Caller updates — `delete.go` `deleteResource` function (lines 122-135):**

Before (lines 131-134):
```go
if ns != "" {
    return client.Dynamic.Resource(gvr).Namespace(ns).Delete(ctx, obj.GetName(), deleteOpts)
}
return client.Dynamic.Resource(gvr).Delete(ctx, obj.GetName(), deleteOpts)
```

After:
```go
return client.ResourceClient(gvr, ns).Delete(ctx, obj.GetName(), deleteOpts)
```

**Caller updates — `diff.go` `fetchLiveState` function (lines 162-182):**

Before (lines 171-175):
```go
if ns != "" {
    live, err = client.Dynamic.Resource(gvr).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
} else {
    live, err = client.Dynamic.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
}
```

After:
```go
live, err = client.ResourceClient(gvr, ns).Get(ctx, name, metav1.GetOptions{})
```

This also simplifies `fetchLiveState` to remove the intermediate `var live` / `var err` declarations.

**Tests — create `resource_test.go`:**

Move existing tests from `apply_test.go`:
- `TestKindToResource` (tests `kindToResource`)
- `TestHeuristicPluralize` (tests `heuristicPluralize`)
- `TestGvrFromObject` (rename to `TestGvrFromUnstructured`, tests `gvrFromUnstructured`)

Add new test:
- `TestResourceClient` — verify namespace-scoped vs cluster-scoped dispatch

### Decision 2: Consolidate `expandTilde` into `config.ExpandTilde`

**Delete from `kubernetes/client.go`:** The `expandTilde` function (lines 186-209).

**Change in `kubernetes/client.go` `resolveKubeconfig` function (line 183):**

Before:
```go
return expandTilde(path)
```

After:
```go
return config.ExpandTilde(path)
```

**Add import to `kubernetes/client.go`:**

```go
"github.com/opmodel/cli/internal/config"
```

**Delete from `kubernetes/client_test.go`:** The entire `TestExpandTilde` function (lines 15-62). These cases are already covered by `config/paths_test.go` (which has 8 test cases including all 6 that `client_test.go` tests).

### Decision 3: Remove `cmdutil.K8sClientOpts`, accept `kubernetes.ClientOptions` directly

**Change `cmdutil/k8s.go`** — delete `K8sClientOpts` struct (lines 8-13), change function signature:

Before (full file):
```go
type K8sClientOpts struct {
    Kubeconfig  string
    Context     string
    APIWarnings string
}

func NewK8sClient(opts K8sClientOpts) (*kubernetes.Client, error) {
    client, err := kubernetes.NewClient(kubernetes.ClientOptions{
        Kubeconfig:  opts.Kubeconfig,
        Context:     opts.Context,
        APIWarnings: opts.APIWarnings,
    })
    if err != nil {
        return nil, &oerrors.ExitError{Code: oerrors.ExitConnectivityError, Err: err}
    }
    return client, nil
}
```

After (full file):
```go
func NewK8sClient(opts kubernetes.ClientOptions) (*kubernetes.Client, error) {
    client, err := kubernetes.NewClient(opts)
    if err != nil {
        return nil, &oerrors.ExitError{Code: oerrors.ExitConnectivityError, Err: err}
    }
    return client, nil
}
```

**Call sites to update (4 files):**

`mod_apply.go` (lines 116-120):
```go
// Before
k8sClient, err := cmdutil.NewK8sClient(cmdutil.K8sClientOpts{
    Kubeconfig:  kf.Kubeconfig,
    Context:     kf.Context,
    APIWarnings: opmConfig.Config.Log.Kubernetes.APIWarnings,
})

// After
k8sClient, err := cmdutil.NewK8sClient(kubernetes.ClientOptions{
    Kubeconfig:  kf.Kubeconfig,
    Context:     kf.Context,
    APIWarnings: opmConfig.Config.Log.Kubernetes.APIWarnings,
})
```

Same pattern for:
- `mod_delete.go` (lines 111-115)
- `mod_diff.go` (lines 91-95)
- `mod_status.go` (lines 124-128)

**Test update — `cmdutil/k8s_test.go`:** Change `K8sClientOpts` to `kubernetes.ClientOptions` in test call sites.

### Decision 4: Export `deleteResult` and `statusResult`

**`delete.go`:** Rename `deleteResult` → `DeleteResult` (line 35). Update:
- Return type of `Delete()` function (line 49): `*deleteResult` → `*DeleteResult`
- Constructor in `Delete()` (line 50): `&deleteResult{}` → `&DeleteResult{}`

**`status.go`:** Rename `statusResult` → `StatusResult` (line 57). Update:
- Return type of `GetModuleStatus()` (line 70): `*statusResult` → `*StatusResult`
- Constructor in `GetModuleStatus()` (line 97): `&statusResult{}` → `&StatusResult{}`
- Parameter types of `FormatStatus()` and `FormatStatusTable()`: `*statusResult` → `*StatusResult`
- Parameter types of `formatStatusJSON()` and `formatStatusYAML()`: `*statusResult` → `*StatusResult`

**Command-layer impact:** None. `mod_delete.go` and `mod_status.go` use type inference (`:=`) so they don't name the type. They access exported fields which remain unchanged.

### Decision 5: Export diff state type to match its constants

The original analysis suggested unexporting `ResourceModified`, `ResourceAdded`, `ResourceOrphaned`. However, `mod_diff.go` (lines 133, 141, 148) references these constants from the `cmd` package:

```go
case kubernetes.ResourceModified:
case kubernetes.ResourceAdded:
case kubernetes.ResourceOrphaned:
```

Unexporting them would cause a compile error. Instead, fix the inconsistency the other way — export the type:

**`diff.go`:** Rename `resourceState` → `ResourceState` (line 21). Export `resourceUnchanged` → `ResourceUnchanged` (line 31). The constants `ResourceModified`, `ResourceAdded`, `ResourceOrphaned` already are exported and stay as-is.

### Decision 6: Remove YAGNI option fields

Fields that are defined but never read in any code path:

| Field | File | Line | Reason for removal |
|-------|------|------|--------------------|
| `ApplyOptions.Wait` | apply.go | 25 | Never checked in `Apply()` or `applyResource()` |
| `ApplyOptions.Timeout` | apply.go | 28 | Never checked anywhere |
| `DeleteOptions.Wait` | delete.go | 31 | Never checked in `Delete()` |
| `StatusOptions.Watch` | status.go | 33 | Never read by `GetModuleStatus()` (watch logic is in cmd layer) |
| `StatusOptions.Kubeconfig` | status.go | 36 | Never read (client is pre-created) |
| `StatusOptions.Context` | status.go | 39 | Never read (client is pre-created) |

**Command-layer impact — fields that are currently passed but no longer exist:**

`mod_apply.go` line 151-155 — currently passes `Wait` and `Timeout`:
```go
// Before
kubernetes.ApplyOptions{
    DryRun:  dryRun,
    Wait:    wait,
    Timeout: timeout,
}

// After
kubernetes.ApplyOptions{
    DryRun: dryRun,
}
```

The `wait` and `timeout` local variables in `mod_apply.go` (lines 24-25) and the CLI flags (lines 70-73) remain — they are accepted by the CLI but are no-ops. This is existing behavior unchanged by this refactoring.

`mod_delete.go` line 135-141 — currently passes `Wait`:
```go
// Before
kubernetes.DeleteOptions{
    ReleaseName: rsf.ReleaseName,
    Namespace:   namespace,
    ReleaseID:   rsf.ReleaseID,
    DryRun:      dryRun,
    Wait:        wait,
}

// After
kubernetes.DeleteOptions{
    ReleaseName: rsf.ReleaseName,
    Namespace:   namespace,
    ReleaseID:   rsf.ReleaseID,
    DryRun:      dryRun,
}
```

`mod_status.go` lines 134-140 — currently passes `Watch`:
```go
// Before
kubernetes.StatusOptions{
    Namespace:    namespace,
    ReleaseName:  rsf.ReleaseName,
    ReleaseID:    rsf.ReleaseID,
    OutputFormat: outputFormat,
    Watch:        watch,
}

// After
kubernetes.StatusOptions{
    Namespace:    namespace,
    ReleaseName:  rsf.ReleaseName,
    ReleaseID:    rsf.ReleaseID,
    OutputFormat: outputFormat,
}
```

The `watch` variable is still used by `mod_status.go` line 143 (`if watch { return runStatusWatch(...) }`).

### Decision 7: Remove `config.ResolvedValue`

**Delete from `config/config.go` (lines 70-84):**

```go
// ResolvedValue tracks a configuration value and its resolution chain.
// Used for logging config resolution with --verbose (FR-019).
type ResolvedValue struct {
    Key      string
    Value    any
    Source   string
    Shadowed map[string]any
}
```

**Delete from `config/config_test.go` (lines 41-57):** The `TestResolvedValue` function:

```go
func TestResolvedValue(t *testing.T) {
    rv := ResolvedValue{
        Key:    "registry",
        Value:  "registry.example.com",
        Source: "env",
        Shadowed: map[string]any{
            "config":  "config-registry.example.com",
            "default": "",
        },
    }
    // ...
}
```

### Decision 8: Replace `containsSlash` with `strings.Contains`

**In `discovery.go` line 225:**

Before:
```go
if containsSlash(r.Name) {
```

After:
```go
if strings.Contains(r.Name, "/") {
```

`strings` is already imported in `discovery.go` (used elsewhere — verify, if not, add the import).

**Delete from `discovery.go` (lines 249-257):** The `containsSlash` function.

**Delete from `discovery_test.go`:** The `TestContainsSlash` test function.

Note: verify `strings` is imported in `discovery.go`. If not, add it.

### Decision 9: Fix stale AGENTS.md

**Line 31:** Change:
```
- Single test: `go test ./pkg/loader -v -run TestName`
```
To:
```
- Single test: `go test ./internal/build -v -run TestName`
```

**Lines 123-124:** Change:
```
- `pkg/loader/` - CUE module loading
- `pkg/version/` - Version info
```
To:
```
- `internal/build/` - CUE module loading and render pipeline
- `internal/version/` - Version info
```

### Decision 10: Fix stale integration tests

**File:** `internal/kubernetes/diff_integration_test.go`

This file references types and constants that have been renamed since it was written. The following substitutions are needed:

| Line(s) | Old | New | Reason |
|---------|-----|-----|--------|
| 69-72 | `DeleteOptions{ModuleName: moduleName, Namespace: namespace}` | `DeleteOptions{ReleaseName: moduleName, Namespace: namespace}` | Field renamed from `ModuleName` to `ReleaseName` |
| 112-115 | `StatusOptions{Name: moduleName, Namespace: namespace}` | `StatusOptions{ReleaseName: moduleName, Namespace: namespace}` | Field renamed from `Name` to `ReleaseName` |
| 118 | `HealthReady` | `healthReady` | Constant was never exported; use unexported name |
| 121-124 | `DeleteOptions{ModuleName: moduleName, Namespace: namespace}` | `DeleteOptions{ReleaseName: moduleName, Namespace: namespace}` | Same field rename |
| 176-179 | `StatusOptions{Name: "nonexistent-module", Namespace: "default"}` | `StatusOptions{ReleaseName: "nonexistent-module", Namespace: "default"}` | Same field rename |
| 182 | `HealthUnknown` | `healthUnknown` | Constant was never exported; use unexported name |

After YAGNI removal (Decision 6), `StatusOptions` no longer has `Watch`, `Kubeconfig`, or `Context` fields, but the integration test doesn't set those, so no further changes needed there.

## Risks / Trade-offs

- **[Risk] New `kubernetes` -> `config` import for `ExpandTilde`** -> Mitigation: `config.ExpandTilde` is a pure function in `paths.go` (only imports `os`, `path/filepath`), so this creates minimal coupling. If it feels wrong later, extraction to `internal/fsutil` is trivial.
- **[Risk] Removing YAGNI fields from option structs breaks future Wait implementation** -> Mitigation: Adding fields back is a backward-compatible change. The fields have been unused since creation and there is no active plan to implement them.
- **[Risk] Renaming `deleteResult`/`statusResult` to exported could break external consumers** -> Mitigation: These types are in `internal/` — no external consumers exist.
- **[Risk] Stale integration tests may mask real failures** -> Mitigation: Fixing the test signatures surfaces any actual regressions. Integration tests are gated behind build tags so they don't block CI.

## Open Questions

_(None — all decisions are mechanical refactoring with clear before/after states.)_
