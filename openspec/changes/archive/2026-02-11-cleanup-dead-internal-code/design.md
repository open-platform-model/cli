## Context

The `internal/` codebase has grown organically during active development. A thorough audit identified ~10 completely dead items (files, packages, functions), ~30+ exported symbols that should be unexported, and 1 stale integration test. All are implementation-internal — no user-facing behavior is affected.

The codebase is under heavy development; cleaning now prevents further accumulation and makes the real API surface of each package clear to contributors.

## Goals / Non-Goals

**Goals:**

- Remove entirely dead packages (`testutil`, `identity`) and the dead stub file (`mod_stubs.go`)
- Remove dead exported functions that are never called outside their package
- Unexport symbols that are only used within their own package (reducing public API surface)
- Fix the stale integration test `DiscoverResources` call signature
- Ensure all tests pass after cleanup

**Non-Goals:**

- Refactoring live code or changing behavior
- Restructuring package layout
- Addressing the `experiments/ast-pipeline/` directory (separate concern)
- Modifying any command flags, output formats, or user-facing behavior

## Decisions

### 1. Delete vs. Unexport

**Decision**: Delete code that has zero callers (including within its own package). Unexport code that is only used internally within its package.

**Rationale**: Deleted code can be recovered from git history if ever needed. Unexporting keeps internal helpers available to the package but removes them from the public API surface, which is the Go-idiomatic approach.

### 2. Handle `identity` package UUID duplication

**Decision**: Delete the `internal/identity` package entirely. Inline the UUID constant directly in `build/release_builder.go` (where it's already hardcoded as a string literal) as an unexported constant. Update the test that imports `identity` to reference the constant from `build` or use the literal directly.

**Rationale**: A whole package for a single constant that duplicates a hardcoded value adds confusion. The constant belongs where it's used.

### 3. Batch changes by package

**Decision**: Group all changes by package and make one commit per package to keep the diff reviewable and bisectable.

**Rationale**: If any single package cleanup introduces a regression, it can be isolated and reverted independently.

### 4. Test-only functions

**Decision**: Functions that are only called from their own test files (e.g., `NewConnectivityError`, `NewNotFoundError`, `NewPermissionError`) will be deleted along with their test cases. The tests were only validating the constructors themselves — the underlying error types and `Wrap()` function remain well-tested.

**Rationale**: Tests that only exercise dead code provide no value. The production error paths use `Wrap()` and `DetailError{}` directly, and those remain tested.

## Risks / Trade-offs

- **Risk**: Removing exported symbols could break out-of-tree consumers → **Mitigation**: This is `internal/`, so Go enforces that only code within this module can import it. No external consumers are possible.
- **Risk**: Unexporting symbols could break tests in the same package → **Mitigation**: Tests within the same package can still access unexported symbols. Only `_test.go` files with a different package name (external test packages) would break, and we'll fix those.
- **Risk**: Missing a usage site → **Mitigation**: `go build ./...` and `task test` will catch any reference to deleted/renamed symbols at compile time.
