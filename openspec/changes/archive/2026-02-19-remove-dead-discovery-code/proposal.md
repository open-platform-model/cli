## Why

The `internal/kubernetes/discovery.go` module contains approximately 150 lines of label-scan discovery code that is no longer used by any production command. All `opm mod` commands (status, delete, diff, apply) have been migrated to inventory-based discovery, which performs targeted GETs per inventory entry rather than scanning all API types on the cluster. The dead code creates maintenance burden and confusion about which discovery path is canonical.

## What Changes

- Delete unused label-scan discovery functions and types from `internal/kubernetes/discovery.go`
- Extract still-needed constants (label names), error types, and utility functions to appropriate files
- Delete stale integration tests that reference removed code
- No changes to any command behavior or public APIs

This is a **PATCH** change (refactor/cleanup, no user-facing changes).

## Capabilities

### New Capabilities
<!-- None - this is pure code cleanup -->

### Modified Capabilities
- `resource-discovery`: Removing label-scan `DiscoverResources()` function requirements (inventory-based discovery is the only supported path)
- `identity-based-discovery`: Removing `BuildReleaseIDSelector` and dual-strategy discovery requirements (inventory handles this)
- `discovery-owned-filter`: Removing `ExcludeOwned` filter requirements from `DiscoverResources()` (no longer applicable without label-scan)

## Impact

**Code removed:**
- `internal/kubernetes/discovery.go` (entire file)
- `internal/kubernetes/discovery_test.go` (entire file)
- `internal/kubernetes/integration_test.go` (entire file, already broken/stale)

**Code created:**
- `internal/kubernetes/labels.go` (label constants extracted from discovery.go)
- `internal/kubernetes/errors.go` (error types extracted from discovery.go)
- `internal/kubernetes/errors_test.go` (tests moved from discovery_test.go)

**Code modified:**
- `internal/kubernetes/delete.go` (add `sortByWeightDescending` moved from discovery.go)
- `internal/kubernetes/delete_test.go` (add test moved from discovery_test.go)

**No impact on:**
- User-facing commands or flags
- External APIs or contracts
- Dependencies
