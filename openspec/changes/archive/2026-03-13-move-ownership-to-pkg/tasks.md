## 1. Create pkg/ownership package

- [x] 1.1 Create `pkg/ownership/ownership.go` with: `CreatedByCLI = "cli"` and `CreatedByController = "controller"` constants, refactored `EnsureCLIMutable(createdBy, releaseName, releaseNamespace string) error` function that checks `createdBy` against `CreatedByController`
- [x] 1.2 Create `pkg/ownership/ownership_test.go` adapted from `internal/workflow/ownership/ownership_test.go`: test with plain string parameters instead of `*inventory.ReleaseInventoryRecord`

## 2. Update callers

- [x] 2.1 Find all callers of `ownership.EnsureCLIMutable` in the codebase and update them to: import `pkg/ownership`, extract `createdBy`, `releaseName`, `releaseNamespace` from the inventory record, pass as plain strings
- [x] 2.2 Update `internal/inventory` package: if it defines `CreatedByController`/`CreatedByCLI` constants, update them to reference `pkg/ownership` constants (or keep both and ensure they match)

## 3. Remove old package

- [x] 3.1 Delete `internal/workflow/ownership/` directory

## 4. Validation

- [x] 4.1 Run `task build` — confirm compilation succeeds
- [x] 4.2 Run `task test` — confirm all tests pass
- [x] 4.3 Run `task lint` — confirm linter passes

## 5. Commits

- [x] 5.1 Commit tasks 1.1–1.2: `refactor(ownership): create pkg/ownership with refactored EnsureCLIMutable`
- [x] 5.2 Commit tasks 2.1–2.2, 3.1: `refactor(ownership): update callers to pkg/ownership and remove internal package`
