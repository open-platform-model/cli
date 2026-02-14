# Delete Operation Test Plan

Comprehensive edge-case tests for `opm mod delete` and `kubernetes.Delete()`.

Each test includes enough detail to implement independently. Tests are grouped
by category. Within each category, tests are ordered by priority (highest first).

---

## Conventions

- **Unit tests** for `kubernetes.Delete` / `deleteResource` / `gvrFromUnstructured`
  go in `internal/kubernetes/delete_test.go` (new file).
- **Unit tests** for `runDelete` / `confirmDelete` / flag validation
  go in `internal/cmd/mod_delete_test.go` (new file).
- Use `k8s.io/client-go/dynamic/fake` for mocking the dynamic client.
- Use `k8s.io/client-go/kubernetes/fake` for mocking the clientset (discovery).
- Use the existing `makeUnstructured` helper from `discovery_test.go` for
  constructing test resources.
- Follow table-driven style with `testify/assert`.
- Each test function name follows the pattern `Test<Unit>_<Scenario>`.

---

## Category 1: Input Validation & Panic Prevention

These tests catch crashes and malformed-input handling. They require no
Kubernetes mocking — they exercise the command layer or pure functions.

### T1.1 — Release ID shorter than 8 characters panics

**Bug, not just a gap.** Three locations slice `[:8]` without a length guard.

```
File:    internal/cmd/mod_delete.go:109
Code:    logName = fmt.Sprintf("release:%s", deleteReleaseIDFlag[:8])
```

**File:** `internal/cmd/mod_delete_test.go`

**Setup:**

1. Create a `NewModDeleteCmd()`.
2. Set args: `--release-id abc -n default --force`
   (release ID "abc" is 3 characters, well under 8).

**Action:**
Call `cmd.Execute()`.

**Assertions:**

- The command must NOT panic.
- It should either return a validation error or use the full short ID as-is.

**Notes:**
The fix is to guard with `min(len(id), 8)` or similar before slicing.
The same pattern exists in `mod_status.go:112,159,222` — write equivalent
tests in `mod_status_test.go` once the fix pattern is established.

---

### T1.2 — Nil opmConfig dereference

**Bug.** `mod_delete.go:117` accesses `opmConfig.Config.Log.Kubernetes.APIWarnings`
without a nil check. Compare with `mod_apply.go:105-107` which does check.

```
File:    internal/cmd/mod_delete.go:117
Code:    APIWarnings: opmConfig.Config.Log.Kubernetes.APIWarnings,
```

**File:** `internal/cmd/mod_delete_test.go`

**Setup:**

1. Create a `NewModDeleteCmd()`.
2. Do NOT call the root command's `PersistentPreRunE` (so `opmConfig` stays nil).
3. Set args: `--name my-app -n default --force`.

**Action:**
Call `cmd.Execute()`.

**Assertions:**

- Must NOT panic with nil pointer dereference.
- Should return a clear error like "OPM config not loaded" or fall back
  to default API warning level.

---

### T1.3 — Mutual exclusivity: both --name and --release-id

**File:** `internal/cmd/mod_delete_test.go`

**Setup:**

1. Create `NewModDeleteCmd()`.
2. Set args: `--name my-app --release-id abc123 -n default`.

**Action:**
Call `cmd.Execute()`.

**Assertions:**

- Returns error.
- Error message contains `"mutually exclusive"`.
- Exit code is `ExitGeneralError` (2).

---

### T1.4 — Neither --name nor --release-id

**File:** `internal/cmd/mod_delete_test.go`

**Setup:**

1. Create `NewModDeleteCmd()`.
2. Set args: `-n default`.

**Action:**
Call `cmd.Execute()`.

**Assertions:**

- Returns error.
- Error message contains `"either --name or --release-id is required"`.

---

### T1.5 — Missing required --namespace flag

**File:** `internal/cmd/mod_delete_test.go`

**Setup:**

1. Create `NewModDeleteCmd()`.
2. Set args: `--name my-app`.

**Action:**
Call `cmd.Execute()`.

**Assertions:**

- Returns error.
- Error message contains `"required flag"`.

---

### T1.6 — gvrFromUnstructured with empty GVK

**File:** `internal/kubernetes/delete_test.go`

**Setup:**

1. Create an `unstructured.Unstructured{}` with no APIVersion or Kind set.

**Action:**
Call `gvrFromUnstructured(obj)`.

**Assertions:**

- Does NOT panic.
- Returns a `GroupVersionResource` with empty Group, Version.
- Resource field is whatever `kindToResource("")` returns.
  (Note: `heuristicPluralize("")` will panic on `lower[len(lower)-2]` —
  this is a separate bug in `apply.go:233`. If fixed, the result should
  be a reasonable default like `""` or `"s"`.)

---

## Category 2: Happy Path (Baseline Coverage)

These are the fundamental paths that must work. They establish the mocking
pattern for all subsequent tests.

### T2.1 — Delete by module name: namespaced resources

**File:** `internal/kubernetes/delete_test.go`

**Setup:**

1. Create a fake dynamic client pre-populated with 3 resources in namespace
   "production", all labeled with `app.kubernetes.io/managed-by=open-platform-model`
   and `module.opmodel.dev/name=my-app`:
   - ConfigMap "app-config"
   - Deployment "app-deploy"
   - Service "app-svc"
2. Create a fake discovery client that returns these resource types.
3. Build a `Client` with both fakes.

**Action:**

```go
result, err := Delete(ctx, client, DeleteOptions{
    ModuleName: "my-app",
    Namespace:  "production",
})
```

**Assertions:**

- `err` is nil.
- `result.Deleted == 3`.
- `result.Errors` is empty.
- `result.Resources` has length 3.
- Verify each resource is gone from the fake client
  (GET returns NotFound).

---

### T2.2 — Delete by release ID

**File:** `internal/kubernetes/delete_test.go`

**Setup:**
Same as T2.1 but resources are labeled with
`module-release.opmodel.dev/uuid=a1b2c3d4-...` instead of module name.

**Action:**

```go
result, err := Delete(ctx, client, DeleteOptions{
    ReleaseID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    Namespace: "production",
})
```

**Assertions:**

- `err` is nil.
- `result.Deleted == 3`.

---

### T2.3 — Dry run does not delete resources

**File:** `internal/kubernetes/delete_test.go`

**Setup:**
Same as T2.1.

**Action:**

```go
result, err := Delete(ctx, client, DeleteOptions{
    ModuleName: "my-app",
    Namespace:  "production",
    DryRun:     true,
})
```

**Assertions:**

- `err` is nil.
- `result.Deleted == 3` (counted as "would be deleted").
- Resources still exist in the fake client (GET returns the object).

---

### T2.4 — No resources found returns typed error

**File:** `internal/kubernetes/delete_test.go`

**Setup:**

1. Empty fake dynamic client (no resources).
2. Fake discovery client with some API resources.

**Action:**

```go
result, err := Delete(ctx, client, DeleteOptions{
    ModuleName: "nonexistent",
    Namespace:  "production",
})
```

**Assertions:**

- `result` is nil.
- `err` is not nil.
- `IsNoResourcesFound(err)` is true.
- Error message contains `"nonexistent"` and `"production"`.

---

### T2.5 — Cluster-scoped resource deletion

**File:** `internal/kubernetes/delete_test.go`

**Setup:**

1. Fake dynamic client with a ClusterRole (cluster-scoped, no namespace)
   labeled with OPM labels.
2. Fake discovery returns `ClusterRole` as a non-namespaced resource.

**Action:**

```go
result, err := Delete(ctx, client, DeleteOptions{
    ModuleName: "my-app",
    Namespace:  "production",
})
```

**Assertions:**

- `err` is nil.
- `result.Deleted == 1`.
- The ClusterRole was deleted via the cluster-scoped API path
  (`delete.go:127`, NOT `delete.go:125`).

---

### T2.6 — Reverse weight ordering

**File:** `internal/kubernetes/delete_test.go`

**Setup:**

1. Fake dynamic client with:
   - Namespace (weight 0)
   - ConfigMap (weight 15)
   - Deployment (weight 100)
   - ValidatingWebhookConfiguration (weight 500)
2. Inject a reactor on the fake client that records the order of Delete calls.

**Action:**

```go
result, err := Delete(ctx, client, DeleteOptions{
    ModuleName: "my-app",
    Namespace:  "production",
})
```

**Assertions:**

- Delete calls are in order: Webhook, Deployment, ConfigMap, Namespace.
- This verifies `sortByWeightDescending` is applied before the deletion loop.

---

## Category 3: Race Conditions & TOCTOU

### T3.1 — Resource deleted externally between discovery and delete

The most common race: another user or controller deletes a resource after
OPM discovers it but before OPM issues the delete call.

**File:** `internal/kubernetes/delete_test.go`

**Setup:**

1. Fake dynamic client with 2 resources: ConfigMap "cm-1" and Secret "secret-1".
2. Add a reactor on the fake client that intercepts Delete for "secret-1" and
   returns a `NotFound` error (simulating external deletion):

   ```go
   fakeClient.PrependReactor("delete", "secrets", func(action testing.Action) (bool, runtime.Object, error) {
       return true, nil, apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, "secret-1")
   })
   ```

**Action:**

```go
result, err := Delete(ctx, client, DeleteOptions{
    ModuleName: "my-app",
    Namespace:  "production",
})
```

**Assertions:**

- `err` is nil (Delete should not fail entirely).
- `result.Deleted == 1` (ConfigMap was deleted).
- `result.Errors` has 1 entry for "secret-1".
- The error in `result.Errors[0].Err` wraps a NotFound.

**Discussion:**
This test documents the current behavior. A follow-up improvement would be
to treat NotFound during deletion as success (the goal was to remove the
resource, and it's gone). If that change is made, update assertions to:

- `result.Deleted == 2`
- `result.Errors` is empty.

---

### T3.2 — Concurrent delete: API returns Conflict

Another `opm mod delete` running simultaneously could cause the API to return
a 409 Conflict (e.g., if preconditions were used) or other transient errors.

**File:** `internal/kubernetes/delete_test.go`

**Setup:**

1. Fake dynamic client with 2 resources.
2. Reactor returns `Conflict` error for the first resource, then succeeds for
   the second:

   ```go
   callCount := 0
   fakeClient.PrependReactor("delete", "*", func(action testing.Action) (bool, runtime.Object, error) {
       callCount++
       if callCount == 1 {
           return true, nil, apierrors.NewConflict(schema.GroupResource{}, "res", fmt.Errorf("conflict"))
       }
       return false, nil, nil // fall through to default handler
   })
   ```

**Action:**
Call `Delete(...)`.

**Assertions:**

- `err` is nil.
- `result.Deleted == 1` (second resource).
- `result.Errors` has 1 entry (first resource).
- The loop did NOT abort — it continued to the second resource.

---

### T3.3 — Context cancelled mid-deletion

User hits Ctrl+C or a CI pipeline timeout fires during the deletion loop.

**File:** `internal/kubernetes/delete_test.go`

**Setup:**

1. Fake dynamic client with 3 resources.
2. Create a `context.WithCancel`.
3. Reactor cancels the context after the first successful delete:

   ```go
   fakeClient.PrependReactor("delete", "*", func(action testing.Action) (bool, runtime.Object, error) {
       cancel() // cancel context after first delete
       return false, nil, nil
   })
   ```

**Action:**

```go
result, err := Delete(ctx, client, DeleteOptions{...})
```

**Assertions:**

- The 2nd and 3rd deletes fail with a context.Canceled error.
- `result.Deleted == 1`.
- `result.Errors` has 2 entries, both with `context.Canceled`.
- The function returns (does not hang).

**Discussion:**
This tests the current behavior where the loop does not check `ctx.Err()`
between iterations. A potential improvement: check `ctx.Err()` at the top of
the loop and return early. If that change is made, update assertions to check
for an early return with partial results.

---

### T3.4 — Resource in Terminating state (already being deleted)

A previous failed delete left resources with `deletionTimestamp` set but
finalizers preventing completion. Calling delete again should succeed from
the API perspective but the resource is not actually gone.

**File:** `internal/kubernetes/delete_test.go`

**Setup:**

1. Fake dynamic client with a resource that has `metadata.deletionTimestamp`
   set and a `finalizers` entry:

   ```go
   obj := makeUnstructured("v1", "ConfigMap", "stuck-cm", "default")
   now := metav1.Now()
   obj.SetDeletionTimestamp(&now)
   obj.SetFinalizers([]string{"some-controller/finalizer"})
   ```

2. The fake delete call returns success (Kubernetes API returns 200 for
   delete on a Terminating resource).

**Action:**
Call `Delete(...)`.

**Assertions:**

- `err` is nil.
- `result.Deleted == 1`.
- The resource is counted as "deleted" even though it still exists
  (finalizer blocks actual removal).

**Discussion:**
This documents current behavior. If `--wait` is implemented, this test
should be extended to verify that wait detects the resource is still present.

---

## Category 4: API Error Handling

### T4.1 — Discovery failure (network error)

**File:** `internal/kubernetes/delete_test.go`

**Setup:**

1. Fake discovery client that returns an error from
   `ServerPreferredResources`:

   ```go
   fakeDiscovery.PrependReactor("get", "resource", func(...) {
       return true, nil, fmt.Errorf("connection refused")
   })
   ```

   Alternatively, construct a `Client` with a discovery that fails.

**Action:**

```go
result, err := Delete(ctx, client, DeleteOptions{
    ModuleName: "my-app",
    Namespace:  "production",
})
```

**Assertions:**

- `result` is nil.
- `err` is not nil.
- Error message contains `"discovering module resources"`.
- `IsNoResourcesFound(err)` is false (it's a different kind of error).

---

### T4.2 — Partial delete failure (RBAC)

User has permission to delete some resource types but not others.

**File:** `internal/kubernetes/delete_test.go`

**Setup:**

1. Fake dynamic client with ConfigMap "cm-1" and Secret "secret-1".
2. Reactor returns `Forbidden` for secrets:

   ```go
   fakeClient.PrependReactor("delete", "secrets", func(...) (bool, runtime.Object, error) {
       return true, nil, apierrors.NewForbidden(
           schema.GroupResource{Resource: "secrets"}, "secret-1", fmt.Errorf("RBAC"))
   })
   ```

**Action:**
Call `Delete(...)`.

**Assertions:**

- `err` is nil (Delete itself does not fail on per-resource errors).
- `result.Deleted == 1` (ConfigMap succeeded).
- `result.Errors` has 1 entry.
- `result.Errors[0].Kind == "Secret"`.
- `result.Errors[0].Err` contains "Forbidden" or "RBAC".

---

### T4.3 — All resources fail to delete

**File:** `internal/kubernetes/delete_test.go`

**Setup:**

1. Fake dynamic client with 2 resources.
2. Reactor returns `InternalServerError` for all deletes.

**Action:**
Call `Delete(...)`.

**Assertions:**

- `err` is nil (the function returns result, not a top-level error).
- `result.Deleted == 0`.
- `result.Errors` has 2 entries.

---

### T4.4 — API server becomes unavailable mid-loop

**File:** `internal/kubernetes/delete_test.go`

**Setup:**

1. Fake dynamic client with 3 resources.
2. Reactor: first delete succeeds, second and third return
   `ServiceUnavailable`:

   ```go
   callCount := 0
   fakeClient.PrependReactor("delete", "*", func(...) (bool, runtime.Object, error) {
       callCount++
       if callCount >= 2 {
           return true, nil, apierrors.NewServiceUnavailable("server shutting down")
       }
       return false, nil, nil
   })
   ```

**Action:**
Call `Delete(...)`.

**Assertions:**

- `result.Deleted == 1`.
- `result.Errors` has 2 entries, both ServiceUnavailable.
- The loop did not break early — all 3 resources were attempted.

---

### T4.5 — Admission webhook rejects delete

A `ValidatingAdmissionWebhook` can reject delete operations.

**File:** `internal/kubernetes/delete_test.go`

**Setup:**

1. Fake dynamic client with a Deployment.
2. Reactor returns a 403 with reason `"admission webhook denied the request"`:

   ```go
   fakeClient.PrependReactor("delete", "deployments", func(...) (bool, runtime.Object, error) {
       return true, nil, apierrors.NewForbidden(
           schema.GroupResource{Resource: "deployments"}, "app",
           fmt.Errorf("admission webhook \"policy.example.com\" denied the request"))
   })
   ```

**Action:**
Call `Delete(...)`.

**Assertions:**

- `result.Deleted == 0`.
- `result.Errors[0].Err` message contains `"admission webhook"`.

---

## Category 5: Discovery Edge Cases (affecting Delete)

### T5.1 — Discovery silently skips resource types without list permission

If the user can't `list` a certain resource type, discovery skips it with
no error (`discovery.go:162-164`). Delete then reports success while
resources the user can't see remain in the cluster.

**File:** `internal/kubernetes/delete_test.go`

**Setup:**

1. Fake dynamic client with ConfigMap "cm-1" (listable) and Secret "secret-1"
   (list returns Forbidden).
2. Reactor on list:

   ```go
   fakeClient.PrependReactor("list", "secrets", func(...) (bool, runtime.Object, error) {
       return true, nil, apierrors.NewForbidden(...)
   })
   ```

**Action:**
Call `Delete(...)`.

**Assertions:**

- `err` is nil.
- `result.Deleted == 1` (only ConfigMap discovered and deleted).
- Secret "secret-1" still exists (never discovered, never deleted).
- No error reported for the skipped Secret.

**Discussion:**
This documents a potential data-loss-by-omission scenario. The test proves
that Delete can silently miss resources. A possible improvement: return
warnings for resource types that couldn't be listed.

---

### T5.2 — Owned resources are excluded from deletion

`ExcludeOwned: true` is hardcoded in `delete.go:64`. Resources with
`ownerReferences` (e.g., Pods owned by a ReplicaSet) should not be
deleted directly — the controller manages them.

**File:** `internal/kubernetes/delete_test.go`

**Setup:**

1. Fake dynamic client with:
   - Deployment "my-deploy" (no ownerReferences).
   - ReplicaSet "my-deploy-abc" with ownerReference pointing to "my-deploy".
   - Pod "my-deploy-abc-xyz" with ownerReference pointing to "my-deploy-abc".
   All labeled with OPM labels.

**Action:**
Call `Delete(...)`.

**Assertions:**

- Only the Deployment is deleted (no ownerReferences).
- ReplicaSet and Pod are excluded from discovery results.
- `result.Deleted == 1`.
- `result.Resources` has length 1.

---

### T5.3 — Cluster-scoped resources deleted alongside namespaced

When a module manages both a ClusterRole (cluster-scoped) and a Deployment
(namespaced), Delete should find and remove both.

**File:** `internal/kubernetes/delete_test.go`

**Setup:**

1. Fake dynamic client with:
   - ClusterRole "my-app-role" (no namespace, labeled with OPM labels).
   - Deployment "my-deploy" in namespace "production" (labeled with OPM labels).
2. Fake discovery returns both `clusterroles` (Namespaced: false)
   and `deployments` (Namespaced: true).

**Action:**

```go
result, err := Delete(ctx, client, DeleteOptions{
    ModuleName: "my-app",
    Namespace:  "production",
})
```

**Assertions:**

- `err` is nil.
- `result.Deleted == 2`.
- Both resources are gone.

**Discussion:**
This is the correct behavior, but it has a side effect: if two modules
share a ClusterRole with the same labels, deleting module A removes the
ClusterRole that module B also needs. This is a design-level concern, not
a code bug, but the test documents the behavior.

---

### T5.4 — Empty namespace string causes cross-namespace discovery

**File:** `internal/kubernetes/delete_test.go`

**Setup:**

1. Fake dynamic client with ConfigMap "cm-1" in namespace "alpha" and
   ConfigMap "cm-2" in namespace "beta", both with OPM labels.

**Action:**

```go
result, err := Delete(ctx, client, DeleteOptions{
    ModuleName: "my-app",
    Namespace:  "", // empty!
})
```

**Assertions:**

- Document what happens: does `.Namespace("").List()` return resources
  from all namespaces? (In real K8s, yes. In fake client, depends on setup.)
- If both ConfigMaps are discovered, both are deleted — potentially
  deleting resources in namespaces the user didn't intend.

**Discussion:**
The `--namespace` flag is marked required in cobra (`mod_delete.go:79`),
so this can only happen if `Delete()` is called programmatically with
empty Namespace. The test documents defensive behavior.

---

## Category 6: deleteResource Edge Cases

### T6.1 — Foreground propagation policy is set

**File:** `internal/kubernetes/delete_test.go`

**Setup:**

1. Fake dynamic client with a Deployment.
2. Reactor captures the `DeleteOptions` passed to the API.

**Action:**
Call `deleteResource(ctx, client, obj)`.

**Assertions:**

- The captured `DeleteOptions.PropagationPolicy` is
  `metav1.DeletePropagationForeground`.

---

### T6.2 — Namespaced vs cluster-scoped branching

**File:** `internal/kubernetes/delete_test.go`

**Setup:**
Two sub-tests:

1. Resource with namespace set ("default").
2. Resource with empty namespace (cluster-scoped).
Use a reactor to capture which API path was called.

**Action:**
Call `deleteResource(ctx, client, obj)` for each.

**Assertions:**

- Namespaced: called `.Namespace("default").Delete(...)`.
- Cluster-scoped: called `.Delete(...)` directly (no namespace).

---

### T6.3 — Nil unstructured object

**File:** `internal/kubernetes/delete_test.go`

**Setup:**
Pass `nil` to `deleteResource`.

**Action:**
Call `deleteResource(ctx, client, nil)`.

**Assertions:**

- Does NOT panic.
- Returns an error (or if it panics, this test documents the need
  for a nil guard).

---

## Category 7: Command Layer (mod_delete.go)

### T7.1 — --ignore-not-found suppresses no-resources error

**File:** `internal/cmd/mod_delete_test.go`

This requires mocking the Kubernetes client at the command level. The
simplest approach is to test the error-handling logic directly if
`runDelete` can be refactored to accept a client, or use the cobra
command with a test server.

**Setup:**

1. Configure the command so that `kubernetes.Delete` returns a
   `noResourcesFoundError`.
2. Set flags: `--name nonexistent -n default --force --ignore-not-found`.

**Action:**
Execute the command.

**Assertions:**

- Returns nil (exit code 0).
- No error printed.

---

### T7.2 — --ignore-not-found does NOT suppress per-resource errors

The `--ignore-not-found` flag only checks `IsNoResourcesFound(err)` on the
top-level error from `Delete()`. Per-resource NotFound errors in
`result.Errors` are still reported.

**File:** `internal/cmd/mod_delete_test.go`

**Setup:**

1. `kubernetes.Delete` returns a `*deleteResult` with `Errors` containing
   a NotFound resource error, and `err == nil`.
2. Set `--ignore-not-found`.

**Action:**
Execute the command.

**Assertions:**

- Returns `ExitGeneralError` (because `len(result.Errors) > 0`).
- This documents the semantic gap: `--ignore-not-found` only handles the
  "zero resources discovered" case, not "some resources vanished during
  deletion".

---

### T7.3 — Confirmation prompt: non-TTY stdin

**File:** `internal/cmd/mod_delete_test.go`

**Setup:**

1. Replace `os.Stdin` with a closed pipe (no input available):

   ```go
   r, w, _ := os.Pipe()
   w.Close()
   oldStdin := os.Stdin
   os.Stdin = r
   defer func() { os.Stdin = oldStdin }()
   ```

**Action:**

```go
result := confirmDelete("my-app", "", "production")
```

**Assertions:**

- Returns `false` (safe default when stdin has no input).

---

### T7.4 — Confirmation prompt: various inputs

**File:** `internal/cmd/mod_delete_test.go`

**Setup:**
Table-driven test with inputs piped to stdin:

| Input          | Expected |
|----------------|----------|
| `"y\n"`        | true     |
| `"yes\n"`      | true     |
| `"Y\n"`        | true     |
| `"YES\n"`      | true     |
| `"Yes\n"`      | true     |
| `"n\n"`        | false    |
| `"no\n"`       | false    |
| `"yep\n"`      | false    |
| `"\n"`         | false    |
| `"  y  \n"`    | true (after TrimSpace)  |

For each case:

```go
r, w, _ := os.Pipe()
w.WriteString(input)
w.Close()
os.Stdin = r
```

**Action:**
Call `confirmDelete(name, releaseID, namespace)`.

**Assertions:**
Return value matches the Expected column.

---

### T7.5 — Confirmation prompt: shows module name vs release ID

**File:** `internal/cmd/mod_delete_test.go`

**Setup:**

1. Capture stderr output.
2. Call `confirmDelete("my-app", "", "production")`.
3. Call `confirmDelete("", "a1b2c3d4", "production")`.

**Assertions:**

- First call: prompt contains `module "my-app"`.
- Second call: prompt contains `release-id "a1b2c3d4"`.
- Both contain `namespace "production"`.

---

### T7.6 — resolveFlag prefers local over global

**File:** `internal/cmd/mod_delete_test.go`

**Setup:**
Table-driven:

| Local Flag | Global Fallback | Expected |
|------------|-----------------|----------|
| `"local"`  | `"global"`      | `"local"` |
| `""`       | `"global"`      | `"global"` |
| `"local"`  | `""`            | `"local"` |
| `""`       | `""`            | `""` |

**Action:**
Call `resolveFlag(local, global)`.

**Assertions:**
Return matches Expected.

---

## Category 8: Wait Flag (Unimplemented)

### T8.1 — --wait flag is accepted but has no effect

**File:** `internal/kubernetes/delete_test.go`

**Setup:**

1. Fake dynamic client with 1 resource.

**Action:**

```go
result, err := Delete(ctx, client, DeleteOptions{
    ModuleName: "my-app",
    Namespace:  "production",
    Wait:       true,
})
```

**Assertions:**

- `err` is nil.
- `result.Deleted == 1`.
- The call returns immediately (does not block).
- This test documents that Wait is a no-op. When Wait is implemented,
  change this test to verify blocking behavior.

---

## Category 9: Log Name Formatting

### T9.1 — Log name uses module name when available

**File:** `internal/kubernetes/delete_test.go`

**Setup:**
`DeleteOptions{ModuleName: "my-app", Namespace: "production"}`.

**Assertions:**

- Verify (through log capture or by checking that the function doesn't
  panic) that `logName` is `"my-app"` (not `"release-id:"`).

---

### T9.2 — Log name falls back to release ID

**File:** `internal/kubernetes/delete_test.go`

**Setup:**
`DeleteOptions{ReleaseID: "a1b2c3d4-e5f6", Namespace: "production"}`.

**Assertions:**

- `logName` is `"release-id:a1b2c3d4-e5f6"` (the full release ID, no
  truncation — note that `delete.go:55` does NOT truncate, unlike
  `mod_delete.go:109` which does).

---

## Category 10: Mixed Resource Types

### T10.1 — Delete with both namespaced and cluster-scoped resources, different weights

**File:** `internal/kubernetes/delete_test.go`

**Setup:**

1. Fake dynamic client with:
   - ValidatingWebhookConfiguration "my-webhook" (cluster-scoped, weight 500)
   - Deployment "my-deploy" (namespaced, weight 100)
   - Service "my-svc" (namespaced, weight 50)
   - ConfigMap "my-cm" (namespaced, weight 15)
   - CustomResourceDefinition "my-crd" (cluster-scoped, weight 1)
2. Record delete order via reactor.

**Action:**
Call `Delete(...)`.

**Assertions:**

- All 5 deleted.
- Order: webhook -> deployment -> service -> configmap -> CRD.
- Namespaced resources use `.Namespace(ns).Delete(...)`.
- Cluster-scoped use `.Delete(...)` directly.

---

### T10.2 — Resources of same kind, different names

**File:** `internal/kubernetes/delete_test.go`

**Setup:**
3 ConfigMaps with different names, all with OPM labels.

**Action:**
Call `Delete(...)`.

**Assertions:**

- All 3 deleted.
- `result.Deleted == 3`.

---

## Category 11: Error Reporting Structure

### T11.1 — resourceError formats with namespace

**File:** `internal/kubernetes/delete_test.go`

**Setup:**

```go
e := resourceError{
    Kind: "Deployment", Name: "my-deploy", Namespace: "production",
    Err: fmt.Errorf("forbidden"),
}
```

**Assertions:**

- `e.Error() == "Deployment/my-deploy in production: forbidden"`.

---

### T11.2 — resourceError formats without namespace

**Setup:**

```go
e := resourceError{
    Kind: "ClusterRole", Name: "my-role", Namespace: "",
    Err: fmt.Errorf("not found"),
}
```

**Assertions:**

- `e.Error() == "ClusterRole/my-role: not found"`.

---

## Test Dependency Graph

Tests that require specific mocking infrastructure. Build these first.

```
Helper: newFakeDeleteClient(resources...) -> *Client
  Used by: T2.*, T3.*, T4.*, T5.*, T6.*, T8.*, T9.*, T10.*

Helper: makeOPMUnstructured(apiVersion, kind, name, ns, moduleName) -> *unstructured.Unstructured
  Like makeUnstructured but also sets OPM labels. Used by all fake client tests.

Helper: recordingReactor(order *[]string) -> reactor
  Records the (kind, name) of each delete call. Used by T2.6, T10.1.
```

Build the helpers first, then tests can be written independently in any order.

---

## Summary Checklist

| ID    | Category                    | Difficulty | Priority |
|-------|-----------------------------|------------|----------|
| T1.1  | Panic: short release ID     | Easy       | Critical |
| T1.2  | Panic: nil opmConfig        | Easy       | Critical |
| T1.3  | Flag: mutual exclusivity    | Easy       | High     |
| T1.4  | Flag: neither provided      | Easy       | High     |
| T1.5  | Flag: missing namespace     | Easy       | High     |
| T1.6  | Panic: empty GVK            | Easy       | Medium   |
| T2.1  | Happy: delete by name       | Medium     | Critical |
| T2.2  | Happy: delete by release ID | Medium     | High     |
| T2.3  | Happy: dry run              | Medium     | High     |
| T2.4  | Happy: no resources         | Medium     | High     |
| T2.5  | Happy: cluster-scoped       | Medium     | Medium   |
| T2.6  | Happy: weight ordering      | Medium     | Medium   |
| T3.1  | Race: external delete       | Medium     | High     |
| T3.2  | Race: concurrent conflict   | Medium     | Medium   |
| T3.3  | Race: context cancelled     | Medium     | High     |
| T3.4  | Race: terminating resource  | Medium     | Medium   |
| T4.1  | API: discovery failure      | Medium     | High     |
| T4.2  | API: partial RBAC failure   | Medium     | High     |
| T4.3  | API: all deletes fail       | Easy       | Medium   |
| T4.4  | API: server unavailable     | Medium     | Medium   |
| T4.5  | API: webhook rejection      | Easy       | Low      |
| T5.1  | Discovery: RBAC skip        | Medium     | High     |
| T5.2  | Discovery: owned excluded   | Medium     | Medium   |
| T5.3  | Discovery: mixed scope      | Medium     | Medium   |
| T5.4  | Discovery: empty namespace  | Easy       | Medium   |
| T6.1  | Propagation policy          | Easy       | Low      |
| T6.2  | Scope branching             | Easy       | Medium   |
| T6.3  | Nil object                  | Easy       | Low      |
| T7.1  | Cmd: ignore-not-found       | Hard       | Medium   |
| T7.2  | Cmd: ignore partial errors  | Hard       | Medium   |
| T7.3  | Cmd: non-TTY prompt         | Easy       | Medium   |
| T7.4  | Cmd: prompt inputs          | Easy       | Medium   |
| T7.5  | Cmd: prompt messages        | Easy       | Low      |
| T7.6  | Cmd: resolveFlag            | Easy       | Low      |
| T8.1  | Wait: no-op documented      | Easy       | Medium   |
| T9.1  | Log: module name            | Easy       | Low      |
| T9.2  | Log: release ID fallback    | Easy       | Low      |
| T10.1 | Mixed: types + ordering     | Medium     | Medium   |
| T10.2 | Mixed: same kind diff names | Easy       | Low      |
| T11.1 | Error: with namespace       | Easy       | Low      |
| T11.2 | Error: without namespace    | Easy       | Low      |
