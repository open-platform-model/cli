## Why

The `internal/workflow/ownership` package enforces the no-takeover policy between CLI and controller. Both tools need this policy — the CLI refuses to mutate controller-managed releases, and the controller will need the inverse check. Making the ownership check public enables shared policy enforcement.

## What Changes

- Move `EnsureCLIMutable` from `internal/workflow/ownership/` to `pkg/ownership/`
- Update callers
- The function currently depends on `internal/inventory.ReleaseInventoryRecord` — this dependency needs to be evaluated. If the function signature can use a simpler interface or `pkg/inventory` types instead, refactor accordingly.

## Capabilities

### New Capabilities

- `pkg-ownership`: Release ownership policy checks are publicly importable from `pkg/ownership/`. External consumers can call ownership validation functions.

### Modified Capabilities

- `inventory-ownership`: The ownership enforcement function moves from `internal/workflow/ownership` to `pkg/ownership`.

## Impact

- **Files moved**: `internal/workflow/ownership/ownership.go`, `internal/workflow/ownership/ownership_test.go` → `pkg/ownership/`
- **Dependency concern**: Currently imports `internal/inventory.ReleaseInventoryRecord` — may need interface extraction or refactoring to avoid pulling internal types into `pkg/`
- **SemVer**: MINOR (new public API surface)
