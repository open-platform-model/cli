## 1. Refactor Discovery Package

- [x] 1.1 Update `internal/kubernetes/discovery.go`: modify `DiscoverResources` to accept only ONE selector type (remove union logic)
- [x] 1.2 Add new error type or sentinel for "no resources found" in discovery
- [x] 1.3 Update `internal/kubernetes/discovery_test.go`: add tests for single-selector behavior

## 2. Update Delete Command

- [x] 2.1 Update `internal/cmd/mod_delete.go`: add mutual exclusivity validation for `--name` and `--release-id` flags
- [x] 2.2 Update `internal/cmd/mod_delete.go`: update help text and examples to reflect new behavior
- [x] 2.3 Update `internal/kubernetes/delete.go`: return error when no resources found (not success)
- [x] 2.4 Update error message format: `"no resources found for module \"<name>\" in namespace \"<namespace>\""` or `"no resources found for release-id \"<uuid>\" in namespace \"<namespace>\""`

## 3. Update Status Command

- [x] 3.1 Update `internal/cmd/mod_status.go`: add `--release-id` flag
- [x] 3.2 Update `internal/cmd/mod_status.go`: add mutual exclusivity validation for `--name` and `--release-id` flags
- [x] 3.3 Update `internal/cmd/mod_status.go`: update help text and examples
- [x] 3.4 Update `internal/kubernetes/status.go`: return error when no resources found

## 4. Documentation

- [x] 4.1 Add `--ignore-not-found` to TODO.md as planned future feature
- [x] 4.2 Update CHANGELOG with breaking changes

## 5. Testing

- [x] 5.1 Run `task test` to verify all changes pass
- [x] 5.2 Manual test: `opm mod delete --name x --release-id y -n z` should error with mutual exclusivity message
- [x] 5.3 Manual test: `opm mod delete --name nonexistent -n default` should error with "no resources found"
- [x] 5.4 Manual test: `opm mod status --release-id <uuid> -n default` should work
