## Why

The `internal/build/` package has grown to 21 files in a flat structure, making it difficult for developers to navigate and understand the codebase. Files mix concerns (pipeline orchestration, AST manipulation, validation, matching, execution) without clear boundaries. Large files (649 lines for release_builder.go, 356 for pipeline.go) combine multiple responsibilities. This reorganization will improve developer productivity by creating focused subpackages with single responsibilities.

## What Changes

- Restructure `internal/build/` from flat 21-file package into 4 focused subpackages
- Create `module/` subpackage for module loading and AST inspection
- Create `release/` subpackage for release building (overlay, values, validation)
- Create `transform/` subpackage for provider loading, matching, and execution
- Create `orchestration/` subpackage for pipeline coordination
- Extract shared types (`LoadedComponent`, `LoadedTransformer`) to root `internal/build/` for cross-package use
- Keep public API unchanged (backward compatible)
- Move test files into their respective subpackages
- Split large files into focused single-responsibility modules
- Update documentation (AGENTS.md) to reflect new structure

## Capabilities

### New Capabilities
- `refactoring-requirements`: Structural and organizational requirements for reorganizing internal/build package into focused subpackages

### Modified Capabilities
<!-- No existing requirements are changing - this is a refactor -->

## Impact

- **Code organization**: 4 new subpackages under `internal/build/`
- **Files affected**: All 21 files in `internal/build/` (moved/split/reorganized)
- **Import paths**: Internal imports within build package change (e.g., `import "github.com/opmodel/cli/internal/build/module"`)
- **Public API**: No breaking changes - `build.NewPipeline()`, `build.Pipeline`, `build.RenderResult` stay unchanged
- **Tests**: All test files move into subpackages, coverage maintained
- **Documentation**: AGENTS.md project structure section updated
- **SemVer impact**: PATCH (internal refactor, no API changes)

## Recent Pre-Refactor Changes

**Note:** The following changes were made to the build package before this refactoring work:

1. **API Rename (Commit 83a240d "Rename"):**
   - `ModuleMetadata` → `ModuleReleaseMetadata` 
   - `RenderResult.Module` → `RenderResult.Release`
   - Added `ModuleName` field to distinguish canonical module name from release name

2. **Deterministic Sorting (Commit 204ebb6 "Implement inventory"):**
   - Changed from simple weight-based sorting to 5-key total ordering
   - Sort keys: weight → group → kind → namespace → name
   - Uses `sort.SliceStable` for deterministic output

3. **Updated Line Counts:**
   - pipeline.go: 356 → 377 lines
   - release_builder.go: 649 → 648 lines
   - errors.go: 562 → 561 lines
   - Total package size: ~5,308 lines (21 files)

These changes are now part of the baseline that will be refactored into subpackages.
