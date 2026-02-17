## 1. Create shared resource utilities file

- [x] 1.1 Create `internal/kubernetes/resource.go` with `gvrFromUnstructured` (from `delete.go:137-145`), `knownKindResources` map (from `apply.go:175-213`), `kindToResource` (from `apply.go:215-222`), `heuristicPluralize` (from `apply.go:224-238`), `isVowel` (from `apply.go:240-242`). Required imports: `strings`, `k8s.io/apimachinery/pkg/apis/meta/v1/unstructured`, `k8s.io/apimachinery/pkg/runtime/schema`, `k8s.io/client-go/dynamic`
- [x] 1.2 Add `ResourceClient` method on `*Client` returning `dynamic.ResourceInterface` — if ns non-empty return `c.Dynamic.Resource(gvr).Namespace(ns)`, else return `c.Dynamic.Resource(gvr)`
- [x] 1.3 Create `internal/kubernetes/resource_test.go`: move `TestKindToResource`, `TestHeuristicPluralize`, `TestGvrFromObject` (rename to `TestGvrFromUnstructured`) from `apply_test.go`; add `TestResourceClient` for namespace/cluster-scoped dispatch

## 2. Update callers to use shared helpers

- [x] 2.1 Update `apply.go`: remove `gvrFromObject`, replace calls with `gvrFromUnstructured`; replace 2 namespaced-vs-cluster-scoped branches in `applyResource` with `client.ResourceClient(gvr, ns)`
- [x] 2.2 Update `delete.go`: remove `gvrFromUnstructured` (now in `resource.go`); replace namespaced branch in `deleteResource` with `client.ResourceClient(gvr, ns)`
- [x] 2.3 Update `diff.go`: replace namespaced branch in `fetchLiveState` with `client.ResourceClient(gvr, ns)`

## 3. Consolidate expandTilde

- [x] 3.1 Remove `expandTilde` function from `kubernetes/client.go` (lines 186-209)
- [x] 3.2 Add `import "github.com/opmodel/cli/internal/config"` to `kubernetes/client.go`
- [x] 3.3 Replace `expandTilde(path)` call in `resolveKubeconfig` with `config.ExpandTilde(path)`
- [x] 3.4 Remove `expandTilde` tests from `kubernetes/client_test.go` (already covered by `config/paths_test.go`)

## 4. Export consistency fixes

- [x] 4.1 Rename `deleteResult` to `DeleteResult` in `delete.go`, update return type of `Delete()` function and all references in `internal/cmd/mod_delete.go`
- [x] 4.2 Rename `statusResult` to `StatusResult` in `status.go`, update return types of `GetModuleStatus()`, `FormatStatus()`, `FormatStatusTable()`, and all references in `internal/cmd/mod_status.go`
- [x] 4.3 Export `resourceState` type as `ResourceState` in `diff.go`; export `resourceUnchanged` as `ResourceUnchanged` (the other three constants `ResourceModified`, `ResourceAdded`, `ResourceOrphaned` are already exported)

## 5. Remove YAGNI option fields

- [x] 5.1 Remove `Wait` (line 25) and `Timeout` (line 28) fields from `ApplyOptions` in `apply.go`; update `mod_apply.go` line 151-155 to stop passing `Wait` and `Timeout` to `ApplyOptions`
- [x] 5.2 Remove `Wait` (line 31) field from `DeleteOptions` in `delete.go`; update `mod_delete.go` line 135-141 to stop passing `Wait` to `DeleteOptions`
- [x] 5.3 Remove `Watch` (line 33), `Kubeconfig` (line 36), and `Context` (line 39) fields from `StatusOptions` in `status.go`; update `mod_status.go` lines 134-140 to stop passing `Watch` to `StatusOptions`
- [x] 5.4 Update any test files that reference removed fields

## 6. Remove redundant cmdutil.K8sClientOpts

- [x] 6.1 Change `NewK8sClient` in `cmdutil/k8s.go` to accept `kubernetes.ClientOptions` directly instead of `K8sClientOpts`
- [x] 6.2 Remove `K8sClientOpts` struct from `cmdutil/k8s.go`
- [x] 6.3 Update all call sites in `internal/cmd/` (`mod_apply.go`, `mod_delete.go`, `mod_diff.go`, `mod_status.go`) to pass `kubernetes.ClientOptions{...}` instead of `cmdutil.K8sClientOpts{...}`
- [x] 6.4 Update `cmdutil/k8s_test.go` to use `kubernetes.ClientOptions`

## 7. Remove unused config.ResolvedValue

- [x] 7.1 Remove `ResolvedValue` struct from `config/config.go` (lines 70-84)
- [x] 7.2 Remove any tests for `ResolvedValue` in `config/config_test.go`

## 8. Replace containsSlash with strings.Contains

- [x] 8.1 Add `"strings"` to imports in `discovery.go`; replace `containsSlash(r.Name)` call at line 225 with `strings.Contains(r.Name, "/")`
- [x] 8.2 Remove `containsSlash` function from `discovery.go` (lines 249-257)
- [x] 8.3 Remove `containsSlash` tests from `discovery_test.go`

## 9. Fix stale documentation

- [x] 9.1 Update AGENTS.md line 31: change `go test ./pkg/loader` to `go test ./internal/build`; lines 123-124: change `pkg/loader/` to `internal/build/` and `pkg/version/` to `internal/version/`

## 10. Fix stale integration tests

- [x] 10.1 In `diff_integration_test.go`: replace `DeleteOptions{ModuleName:` with `DeleteOptions{ReleaseName:` (lines 69-72, 121-124); replace `StatusOptions{Name:` with `StatusOptions{ReleaseName:` (lines 112-115, 176-179); replace `HealthReady` with `healthReady` (line 118); replace `HealthUnknown` with `healthUnknown` (line 182)

## 11. Validation

- [x] 11.1 Run `task fmt` and fix any formatting issues
- [x] 11.2 Run `task vet` and fix any issues
- [x] 11.3 Run `task test` and ensure all unit tests pass
- [x] 11.4 Run `task lint` and fix any linter warnings
