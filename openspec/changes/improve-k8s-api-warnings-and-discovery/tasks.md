## 1. Config Schema and Types

- [ ] 1.1 Add `LogKubernetesConfig` struct to `internal/config/config.go` with `APIWarnings` field
- [ ] 1.2 Update `LogConfig` struct to include `Kubernetes LogKubernetesConfig` (non-pointer, always present)
- [ ] 1.3 Update `DefaultConfigTemplate` in `internal/config/templates.go` with `log.kubernetes.apiWarnings` field
- [ ] 1.4 Update config loader in `internal/config/loader.go` to extract `log.kubernetes.apiWarnings` value

## 2. Warning Handler Implementation

- [ ] 2.1 Create `internal/kubernetes/warnings.go` with `opmWarningHandler` struct
- [ ] 2.2 Implement `rest.WarningHandler` interface on `opmWarningHandler`
- [ ] 2.3 Add routing logic: warn → `output.Warn()`, debug → `output.Debug()`, suppress → no-op
- [ ] 2.4 Update `internal/kubernetes/client.go` to set `restConfig.WarningHandler` before creating clientset
- [ ] 2.5 Wire config value through to warning handler (pass APIWarnings level to client creation)

## 3. Discovery Improvements

- [ ] 3.1 Add `ExcludeOwned bool` field to `DiscoveryOptions` in `internal/kubernetes/discovery.go`
- [ ] 3.2 Replace `ServerGroupsAndResources()` with `ServerPreferredResources()` in `discoverAPIResources()`
- [ ] 3.3 Add ownerReference filtering in `discoverWithSelector()` when `ExcludeOwned` is true
- [ ] 3.4 Update `DiscoverResources()` to pass through the `ExcludeOwned` option

## 4. Command Integration

- [ ] 4.1 Update `Delete()` in `internal/kubernetes/delete.go` to set `ExcludeOwned: true`
- [ ] 4.2 Update diff discovery in `internal/kubernetes/diff.go` to set `ExcludeOwned: true`
- [ ] 4.3 Verify `status` command continues to work (uses `ExcludeOwned: false` by default)

## 5. Testing

- [ ] 5.1 Add unit tests for `opmWarningHandler` with warn/debug/suppress modes
- [ ] 5.2 Add unit tests for `LogKubernetesConfig` extraction in config loader
- [ ] 5.3 Add unit tests for `ExcludeOwned` filtering in discovery
- [ ] 5.4 Add unit test for `ServerPreferredResources()` usage (mock clientset)
- [ ] 5.5 Update existing discovery tests to account for new option
- [ ] 5.6 Integration test: delete with Service that has auto-created Endpoints (no 404 errors)

## 6. Documentation and Cleanup

- [ ] 6.1 Update TODO.md to mark SDK version task as done and reference this change
- [ ] 6.2 Add inline documentation to new config fields
- [ ] 6.3 Run `task check` (fmt + vet + test) to verify all changes pass validation gates
