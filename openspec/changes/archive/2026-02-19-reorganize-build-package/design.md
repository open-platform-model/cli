## Context

The `internal/build/` package currently contains 21 files in a flat structure with mixed concerns:

**Current Pain Points:**
- Navigation difficulty: Developers must scan 21 files to find relevant code
- Mixed responsibilities: Single files handle multiple distinct concerns (e.g., release_builder.go does overlay generation, validation, metadata extraction, component extraction)
- Large files: release_builder.go (649 lines), pipeline.go (356 lines), errors.go (562 lines)
- Unclear dependencies: Import relationships between files not visible from structure
- Test organization: 9 test files mixed with implementation

**Current Structure:**
```
internal/build/
├── pipeline.go (377 lines) - Pipeline implementation + helpers + deterministic sorting
├── release_builder.go (648 lines) - Release building + overlay + validation + metadata
├── provider.go (208 lines) - Provider loading
├── matcher.go (218 lines) - Component matching
├── executor.go (260 lines) - Transformer execution
├── module.go (80 lines) - AST metadata extraction
├── context.go (90 lines) - Transformer context
├── types.go (237 lines) - Shared types (ModuleReleaseMetadata)
├── errors.go (561 lines) - Errors + validation logic
└── 9 test files + testdata/
```

**Recent API Changes (Pre-Refactor):**
- `ModuleMetadata` renamed to `ModuleReleaseMetadata` with added `ModuleName` field
- `RenderResult.Module` renamed to `RenderResult.Release`
- Resource sorting changed to 5-key deterministic ordering (weight → group → kind → namespace → name)

**Key Dependencies:**
- Commands (`internal/cmd/mod/`) depend on public API: `build.NewPipeline()`, `build.RenderResult`, `build.ModuleReleaseMetadata`
- Pipeline orchestrates: module loading → release building → transformation
- Release building uses: module inspection (AST) + validation
- Transformation uses: provider loading → matching → execution
- Kubernetes package depends on `ModuleReleaseMetadata` for apply/diff operations

**Constraints:**
- Public API must remain unchanged (backward compatibility)
- All tests must continue to pass
- No behavior changes (pure refactor)
- Shared types need visibility across subpackages

## Goals / Non-Goals

**Goals:**
1. Improve developer navigation: Find relevant code by package name
2. Enforce separation of concerns: Each subpackage has single responsibility
3. Reduce file sizes: No file over 300 lines
4. Make dependencies explicit: Import paths show relationships
5. Improve testability: Tests live with implementation
6. Maintain backward compatibility: Public API unchanged

**Non-Goals:**
1. Change behavior or fix bugs (pure refactor)
2. Modify public API or command interfaces
3. Change test coverage requirements
4. Introduce new features or capabilities
5. Change build or runtime performance characteristics

## Decisions

### Decision 1: Four-Subpackage Organization

**Choice:** Create `module/`, `release/`, `transform/`, `orchestration/` subpackages

**Rationale:**
- Maps to natural pipeline phases: module → release → transform → orchestration
- Each subpackage has clear input/output contracts
- Matches developer mental model of the rendering process
- Aligns with Principle II (Separation of Concerns)

**Alternatives Considered:**
- **Three packages** (merge module into release): Rejected because module inspection is used independently by InspectModule()
- **Five packages** (split transform into provider/matcher/executor): Rejected as over-engineering for current codebase size
- **Functional organization** (loaders/, validators/, executors/): Rejected because it cuts across natural workflow boundaries

### Decision 2: Shared Types Stay in Root

**Choice:** Keep `LoadedComponent`, `LoadedTransformer`, `ModuleReleaseMetadata`, and other public API types in root `internal/build/`

**Rationale:**
- These types are shared across multiple subpackages
- Avoids circular dependencies (module imports transform, transform imports module)
- Keeps public API clean (callers import `build`, not `build/module`)
- Creates `component.go` and `transformer.go` in root as explicit shared type files
- `ModuleReleaseMetadata` (previously `ModuleMetadata`) stays in `types.go` as it's part of public API

**Alternatives Considered:**
- **Move to subpackages and re-export**: Adds indirection and confusing import paths
- **Create separate shared/ package**: Overkill for ~100 lines of shared types
- **Keep everything in types.go**: Doesn't clarify what's public vs shared-internal

**Note on Recent API Evolution:**
- `ModuleMetadata` was recently renamed to `ModuleReleaseMetadata` to better distinguish between module definition metadata (from `module.metadata.name`) and release instance metadata (from `--release-name`)
- New `ModuleName` field added to preserve canonical module name when release name is overridden

### Decision 3: Phased Migration Strategy

**Choice:** Implement in 5 sequential phases (module → release → transform → orchestration → cleanup)

**Rationale:**
- Each phase is independently testable
- Allows rollback at any phase boundary
- Reduces risk of breaking changes
- Makes code review easier (review one subpackage at a time)

**Alternatives Considered:**
- **Big bang migration**: Too risky, hard to debug failures
- **Feature branch for months**: Conflicts with main branch changes
- **Per-file migration**: Doesn't create clean package boundaries

### Decision 4: Split Large Files by Responsibility

**Choice:** 
- Split release_builder.go → builder.go, overlay.go, validation.go, metadata.go
- Split errors.go → keep errors, move validation to release/validation.go
- Split pipeline.go → orchestration/pipeline.go, orchestration/helpers.go

**Rationale:**
- Each file has single responsibility (Single Responsibility Principle)
- Easier to find specific functionality
- Reduces merge conflicts
- Improves code navigation

**Alternatives Considered:**
- **Keep large files**: Rejected because it doesn't address core pain point
- **Split by feature**: Rejected because responsibilities are clearer than features

### Decision 5: Tests Move With Implementation

**Choice:** Move test files into subpackages alongside implementation

**Rationale:**
- Tests document the package they're testing
- Easier to run subset of tests (`go test ./internal/build/release`)
- Follows Go convention (tests live in same package)
- Shared testdata/ stays at root (used by multiple packages)

**Alternatives Considered:**
- **Separate tests/ directory**: Rejected because Go idiom is tests alongside code
- **Keep all tests in root**: Rejected because it doesn't improve organization

## Risks / Trade-offs

### Risk: Import Cycle Between Subpackages
**Mitigation:** Shared types stay in root. Dependency flow is unidirectional: orchestration → transform/release/module (no cycles possible)

### Risk: Breaking Internal Callers
**Mitigation:** Public API unchanged. Commands only import `build`, not subpackages. Verify with `task test` after each phase.

### Risk: Test Failures During Migration
**Mitigation:** Run tests after each file move. Keep original files until tests pass. Use Git to rollback if needed.

### Risk: Merge Conflicts During Long Refactor
**Mitigation:** Complete in single session (~3 hours). Coordinate with team to avoid parallel build/ changes.

### Trade-off: More Import Paths
**Accepted:** Subpackages add import statements, but makes dependencies explicit (good for understanding)

### Trade-off: More Directories to Navigate
**Accepted:** 4 new directories is manageable and improves navigation vs 21 flat files

## Migration Plan

### Phase 1: Module Package (30-45 min)
1. Create `internal/build/module/` directory
2. Create module/loader.go, module/inspector.go, module/types.go
3. Move `LoadedComponent` to root `component.go` (shared type)
4. Update imports in pipeline.go and release_builder.go
5. Delete module.go
6. Run `go test ./internal/build/module -v && task test`

### Phase 2: Release Package (45-60 min)
1. Create `internal/build/release/` directory
2. Create release/builder.go, release/overlay.go, release/validation.go, release/metadata.go, release/types.go
3. Move test files to release/
4. Update imports in pipeline.go, executor.go, errors.go
5. Delete release_builder.go
6. Run `go test ./internal/build/release -v && task test`

### Phase 3: Transform Package (30-45 min)
1. Create `internal/build/transform/` directory
2. Move `LoadedProvider`, `LoadedTransformer` to root `transformer.go` (shared types)
3. Create transform/provider.go, transform/matcher.go, transform/executor.go, transform/context.go, transform/types.go
4. Move test files to transform/
5. Update imports in pipeline.go
6. Delete provider.go, matcher.go, executor.go, context.go
7. Run `go test ./internal/build/transform -v && task test`

### Phase 4: Orchestration Package (20-30 min)
1. Create `internal/build/orchestration/` directory
2. Create orchestration/pipeline.go, orchestration/helpers.go
3. Move pipeline_test.go to orchestration/
4. Rewrite root pipeline.go as thin facade (exports NewPipeline)
5. Run `go test ./internal/build/orchestration -v && task test`

### Phase 5: Cleanup & Verification (15-20 min)
1. Verify all old files deleted
2. Create internal/build/README.md
3. Update AGENTS.md project structure
4. Run `task check && task build`
5. Manual end-to-end test with sample module
6. Commit: `refactor(build): reorganize into focused subpackages`

**Rollback Strategy:**
- Each phase is a Git checkpoint
- If phase fails: `git checkout -- internal/build/` and retry
- If critical bug found post-merge: Revert commit, fix in separate PR

**Testing Strategy:**
- Run full test suite after each phase
- Verify `task build` succeeds after each phase
- Manual smoke test: `./bin/opm mod apply ./tests/fixtures/simple-module`
- Check test coverage doesn't drop: `task test:coverage`

## Open Questions

None - design is complete and ready for implementation.
