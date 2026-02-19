## Context

The `internal/kubernetes/discovery.go` file contains two discovery approaches:
1. **Label-scan discovery** (`DiscoverResources`): Scans all API types on the cluster with label selectors
2. **Inventory-based discovery** (constants/errors used by `inventory.DiscoverResourcesFromInventory`): Performs targeted GETs per inventory entry

All production commands have migrated to inventory-based discovery. The label-scan code is unused, but the file also hosts several constants, error types, and a utility function that active code depends on.

## Goals / Non-Goals

**Goals:**
- Remove all dead label-scan discovery code (~150 lines)
- Preserve all actively-used symbols by moving them to appropriate files
- Maintain zero functional changes to any command behavior
- Ensure all tests pass after refactor

**Non-Goals:**
- Changing discovery behavior or adding new features
- Modifying command APIs or flags
- Optimizing the inventory-based discovery path

## Decisions

### Decision 1: Extract survivors before deletion

**Choice:** Create new files (`labels.go`, `errors.go`) and move functions into existing files (`delete.go`) before deleting `discovery.go`.

**Rationale:** Minimizes risk. Each move can be verified independently. If we deleted first and tried to remember what to preserve, we'd risk missing something.

**Alternatives considered:**
- Delete and fix compile errors: Too risky, harder to review, easy to miss unused-but-still-valuable constants

### Decision 2: Group symbols by cohesion, not by origin

**Choice:**
- Label constants → `labels.go` (used by apply, status, inventory)
- Error types → `errors.go` (used by delete, status, commands)
- `sortByWeightDescending` → `delete.go` (only caller)

**Rationale:** Follows "Separation of Concerns" principle. Each file has a clear purpose. Future readers won't need to know these came from `discovery.go`.

**Alternatives considered:**
- Keep everything in `discovery.go` and just delete dead functions: Misleading file name, still implies label-scan discovery exists
- Create `inventory_support.go` for survivors: Vague name, doesn't clarify what's inside

### Decision 3: Delete stale integration tests entirely

**Choice:** Delete `integration_test.go` without replacement.

**Rationale:** The file is already broken (calls non-existent function signatures, uses removed struct fields). It tests the label-scan path we're removing. No value in fixing it.

**Alternatives considered:**
- Fix the integration test to use inventory discovery: Not needed — `tests/integration/deploy/` and `tests/integration/inventory-ops/` already test the inventory path end-to-end

### Decision 4: Move tests with their functions

**Choice:**
- `TestSortByWeightDescending` → `delete_test.go` (function moved to `delete.go`)
- `TestNoResourcesFoundError` → `errors_test.go` (error type moved to `errors.go`)
- Delete tests for dead functions (`TestBuildReleaseNameSelector`, etc.)

**Rationale:** Tests should live near the code they test. Makes it obvious what's covered.

## Risks / Trade-offs

**Risk: Miss an active reference to a "dead" symbol**
→ Mitigation: The thorough exploration already identified all references. Compile will catch any mistakes (symbols would be undefined).

**Risk: Break an import in external code (if this were a library)**
→ Mitigation: N/A — this is `internal/` package code, not exposed as a public API.

**Trade-off: More files in `internal/kubernetes/`**
→ Acceptable — `labels.go` and `errors.go` are small, focused files. Better than keeping everything in a misleadingly-named `discovery.go`.

**Trade-off: `delete.go` gains a utility function**
→ Acceptable — `sortByWeightDescending` is only called from `Delete()`, so colocation makes sense. If a second caller appears later, we can extract to a shared utility file.
