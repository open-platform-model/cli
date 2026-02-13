## Why

The `improve-k8s-api-warnings-and-discovery` change shipped with several deferred items: untestable warning handler routing logic, a spec-violating silent swallow of unavailable API groups during discovery, and a missing UX hint in `opm config init`. These are small gaps that erode test confidence and user experience.

## What Changes

- **Warning handler testability**: Refactor `opmWarningHandler` to accept an injected logger interface, enabling tests to assert on actual routing behavior (warn/debug/suppress) instead of only checking no-panic
- **Discovery group failure logging**: Add `output.Warn()` when `ServerPreferredResources()` returns `IsGroupDiscoveryFailedError`, fulfilling the existing spec requirement that unavailable groups are logged
- **Config init UX**: Add a `cue mod tidy` hint to the `opm config init` success output, since the generated config imports `opmodel.dev/providers@v0` which requires dependency resolution

## Capabilities

### New Capabilities

<!-- None — this change closes gaps in existing capabilities -->

### Modified Capabilities

- `discovery-owned-filter`: Add missing warning log for unavailable API groups (spec scenario compliance)
- `k8s-warning-routing`: No spec change — implementation refactor for testability only

## Impact

- **`internal/kubernetes/warnings.go`**: Extract `warningLogger` interface, inject into `opmWarningHandler`, add default `outputWarningLogger` adapter
- **`internal/kubernetes/warnings_test.go`**: Replace no-panic assertions with mock-based routing verification using the existing `wantLevel` test table field
- **`internal/kubernetes/client.go`**: Pass default logger when constructing `opmWarningHandler`
- **`internal/kubernetes/discovery.go`**: Add `output.Warn()` in `IsGroupDiscoveryFailedError` branch (~line 200)
- **`internal/cmd/config_init.go`**: Add `cue mod tidy` hint line to success output

SemVer: **PATCH** (bug fixes and test improvements, no new user-facing features)
