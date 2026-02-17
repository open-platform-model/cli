## Why

The codebase has grown organically and accumulated structural inconsistencies: duplicated utility functions, an overloaded `internal/output` package that creates unnecessary transitive dependencies, tight coupling between `internal/kubernetes` and `internal/build`, inconsistent export patterns, and YAGNI option fields that violate Principle VII. Addressing these now prevents compounding technical debt and makes the codebase easier to read, test, and extend.

## What Changes

- **Unify duplicated `gvrFromObject`/`gvrFromUnstructured`** into a single shared function in `internal/kubernetes/`, placed alongside the `kindToResource` helper it depends on
- **Extract a `ResourceClient` helper** on `*Client` to eliminate 4 repeated namespaced-vs-cluster-scoped branching blocks across `apply.go`, `delete.go`, and `diff.go`
- **Consolidate duplicated `expandTilde`** implementations (`config.ExpandTilde` and `kubernetes.expandTilde`) into a single shared location
- **Make result type exports consistent** — export `deleteResult` and `statusResult` (currently unexported) to match `ApplyResult` and `DiffResult`
- **Export diff state type to match its constants** — `ResourceModified`, `ResourceAdded`, `ResourceOrphaned` are exported but their type `resourceState` is not; export the type as `ResourceState` and export `resourceUnchanged` as `ResourceUnchanged` for consistency
- **Remove YAGNI option fields** — `ApplyOptions.Wait/Timeout`, `DeleteOptions.Wait`, `StatusOptions.Watch/Kubeconfig/Context` are defined but never read
- **Remove redundant `cmdutil.K8sClientOpts`** mirror struct — have `NewK8sClient` accept `kubernetes.ClientOptions` directly
- **Remove unused `config.ResolvedValue`** type that duplicates `config.ResolvedField`
- **Replace `containsSlash` loop** with `strings.Contains` for readability
- **Fix stale AGENTS.md** references to non-existent `pkg/loader/` and `pkg/version/`
- **Fix stale integration tests** in `diff_integration_test.go` that reference old type/field names

## Capabilities

### New Capabilities

- `k8s-helpers`: Shared Kubernetes utility functions (unified GVR resolution, `ResourceClient` method, consolidated `expandTilde`)

### Modified Capabilities

_(No spec-level behavioral changes. All modifications are internal implementation details — type exports, YAGNI cleanup, and consistency fixes that do not alter user-observable behavior.)_

## Impact

- **Packages modified**: `internal/kubernetes/`, `internal/cmdutil/`, `internal/config/`, `internal/cmd/`
- **APIs changed**: `cmdutil.NewK8sClient` signature changes (takes `kubernetes.ClientOptions` instead of `K8sClientOpts`), `kubernetes.DeleteResult`/`StatusResult` become exported types
- **No user-facing CLI changes**: All changes are internal refactoring
- **SemVer**: PATCH — no public API or CLI behavior changes
- **Risk**: Low — all changes are mechanical refactoring with existing test coverage
