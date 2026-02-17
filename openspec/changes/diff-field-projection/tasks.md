## 1. Server-managed metadata stripping

- [ ] 1.1 Add `stripServerManagedFields(obj map[string]interface{})` function to `internal/kubernetes/diff.go` that removes `metadata.managedFields`, `metadata.uid`, `metadata.resourceVersion`, `metadata.creationTimestamp`, `metadata.generation`, and top-level `status` from the object
- [ ] 1.2 Add unit tests for `stripServerManagedFields` — verify each field is removed, verify non-server fields are preserved, verify function is safe on objects missing these fields

## 2. Field projection

- [ ] 2.1 Add `projectLiveToRendered(rendered, live map[string]interface{}) map[string]interface{}` function that recursively walks the rendered object and retains only matching paths in a deep copy of the live object
- [ ] 2.2 Handle map values: for each key in rendered that exists in live, if both values are maps then recurse; otherwise keep the live value
- [ ] 2.3 Handle list values: for lists of maps, match elements by `name` field; fall back to index-based matching when no `name` field is present; for scalar lists, keep the live list as-is
- [ ] 2.4 Handle empty map cleanup: after projection, remove maps that became empty (all keys stripped) to prevent spurious diffs
- [ ] 2.5 Add unit tests for `projectLiveToRendered` — table-driven tests covering: identical objects, server defaults stripped, nested map projection, list matching by name, list fallback to index, empty map removal, scalar list preservation, missing keys in live

## 3. Integration into Diff flow

- [ ] 3.1 In `Diff()` function, after `fetchLiveState()` and before `comparer.Compare()`, call `stripServerManagedFields` then `projectLiveToRendered` on the live object
- [ ] 3.2 Update existing diff unit tests to verify that server-managed fields no longer produce diffs
- [ ] 3.3 Update integration tests to verify apply-then-diff with no changes reports "No differences found"

## 4. Validation

- [ ] 4.1 Run `task test` and fix any failures
- [ ] 4.2 Run `task check` (fmt + vet + test) to verify all checks pass
