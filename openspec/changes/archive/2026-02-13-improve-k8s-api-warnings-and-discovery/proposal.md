## Why

The TODO item "Update the CLI kubernetes SDK to 1.34+" is partially stale—the SDK is already at v0.35.0 (K8s 1.35). However, two underlying problems remain:

1. **Ugly klog warnings**: K8s API deprecation warnings (e.g., "v1 ComponentStatus is deprecated") bypass charmbracelet/log and print raw klog output to stderr, breaking the CLI's consistent output formatting.

2. **Delete 404 errors**: Discovery finds auto-managed resources (Endpoints, EndpointSlice) that inherit OPM labels from their parent Services. When delete runs, these resources are already garbage-collected by the time OPM tries to delete them, causing spurious "not found" errors.

## What Changes

- **K8s API warning routing**: Implement a custom `rest.WarningHandler` that routes Kubernetes API deprecation warnings through charmbracelet/log instead of klog
- **Configurable warning behavior**: Add `log.kubernetes.apiWarnings` config option with values `"warn"` (default), `"debug"`, or `"suppress"`
- **Preferred API discovery**: Switch from `ServerGroupsAndResources()` to `ServerPreferredResources()` to reduce API calls and naturally avoid deprecated API versions
- **Owner-reference filtering**: Add `ExcludeOwned` option to discovery that skips resources with `ownerReferences`, preventing attempts to delete controller-managed children

## Capabilities

### New Capabilities

- `k8s-warning-routing`: Custom warning handler that routes K8s API warnings through charmbracelet/log with config-driven behavior (warn/debug/suppress)
- `discovery-owned-filter`: Filter to exclude controller-managed resources (those with ownerReferences) during delete and diff operations

### Modified Capabilities

<!-- No existing spec-level requirements are changing -->

## Impact

- **Config schema**: Extend `log` section in config.cue with `kubernetes.apiWarnings` field
- **Config Go types**: Add `LogKubernetesConfig` struct to `internal/config/config.go`
- **Config loader**: Extract new field in `internal/config/loader.go`
- **K8s client**: Set custom `WarningHandler` on `rest.Config` in `internal/kubernetes/client.go`
- **New file**: `internal/kubernetes/warnings.go` implementing `rest.WarningHandler`
- **Discovery**: Switch to `ServerPreferredResources()` and add `ExcludeOwned` filtering in `internal/kubernetes/discovery.go`
- **Delete/Diff**: Pass `ExcludeOwned: true` in `internal/kubernetes/delete.go` and `internal/kubernetes/diff.go`
- **TODO.md**: Update to mark SDK version task done and document these improvements

SemVer: **MINOR** (new config option with default, no breaking changes)
