## Why

The `opm mod delete` command has confusing behavior when both `--name` and `--release-id` flags are provided—it takes the **union** of both selectors, meaning a wrong release-id combined with a correct name still deletes resources. Additionally, all discovery-based commands (`delete`, `status`) silently succeed when no resources match, making typos and mistakes hard to detect.

## What Changes

- **BREAKING**: `--name` and `--release-id` flags become mutually exclusive on `delete` and `status` commands
- **BREAKING**: Commands fail with error when no resources match selector (previously returned success with "no resources found" message)
- Add `--release-id` flag to `status` command (currently only has `--name`)
- Document `--ignore-not-found` flag as future TODO for idempotent delete operations

## Capabilities

### New Capabilities

- `resource-discovery`: Defines how OPM discovers resources in a cluster using label selectors, including validation rules and error handling

### Modified Capabilities

_None—this is a new specification for behavior that was previously undocumented._

## Impact

**Commands affected:**

- `internal/cmd/mod_delete.go` — flag validation, help text, mutual exclusivity
- `internal/cmd/mod_status.go` — add `--release-id` flag, flag validation

**Packages affected:**

- `internal/kubernetes/discovery.go` — remove union logic, single selector per invocation
- `internal/kubernetes/delete.go` — return error on no match
- `internal/kubernetes/status.go` — return error on no match

**Tests affected:**

- `internal/kubernetes/discovery_test.go`

**SemVer**: MINOR (breaking changes are acceptable during v0.x development per SemVer spec)
