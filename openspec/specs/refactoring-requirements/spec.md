# Refactoring Requirements

## Purpose

Structural requirements for internal codebase organization. These requirements govern how code is organized into packages, how dependencies flow between packages, how files are sized, and how tests are co-located with implementation. They are enforced as part of refactoring changes, not end-user features.

---

## Requirements

### Requirement: Package organization by responsibility

The build package SHALL be organized into subpackages with single, focused responsibilities.

#### Scenario: Module loading subpackage

- **WHEN** module-related code is needed
- **THEN** it SHALL be located in `internal/build/module/` package

#### Scenario: Release building subpackage

- **WHEN** release building code is needed
- **THEN** it SHALL be located in `internal/build/release/` package

#### Scenario: Transformation subpackage

- **WHEN** transformer-related code is needed
- **THEN** it SHALL be located in `internal/build/transform/` package

#### Scenario: Orchestration in root

- **WHEN** pipeline coordination code is needed
- **THEN** it SHALL be in `internal/build/pipeline.go` (see: Root pipeline contains orchestration logic)

### Requirement: File size limits

Files in the reorganized build package SHALL NOT exceed 300 lines.

#### Scenario: Large file split

- **WHEN** a file from the old structure exceeds 300 lines
- **THEN** it SHALL be split into multiple files by responsibility

#### Scenario: Validation after refactor

- **WHEN** reorganization is complete
- **THEN** all files SHALL be verified to be under 300 lines

### Requirement: Shared type accessibility

Types shared across multiple subpackages SHALL be defined at root `internal/build/` level.

#### Scenario: LoadedComponent shared type

- **WHEN** LoadedComponent is needed by module and release packages
- **THEN** it SHALL be accessible via `internal/build/component.go` at root level

> **Note**: `LoadedComponent` is defined in `internal/build/module/types.go` (the package that owns it) and re-exported at root via a type alias in `component.go`. This achieves the same goal — all callers import `build.LoadedComponent` — while keeping the struct definition close to its owning package.

#### Scenario: LoadedTransformer shared type

- **WHEN** LoadedTransformer is needed by transform and error packages
- **THEN** it SHALL be defined in `internal/build/transformer_adapter.go` at root level

#### Scenario: Public API types

- **WHEN** types are part of the public API (Pipeline, RenderResult, RenderOptions, ModuleReleaseMetadata)
- **THEN** they SHALL remain in `internal/build/types.go` at root level

> **Note**: `ModuleReleaseMetadata` (previously named `ModuleMetadata`) is part of the public API and must stay in types.go. This type was renamed to better distinguish module definition metadata from release instance metadata.

### Requirement: No circular dependencies

Subpackages SHALL have unidirectional dependency flow without circular imports.

#### Scenario: Dependency direction

- **WHEN** subpackages import each other
- **THEN** the dependency flow SHALL be: `pipeline.go` → (module, release, transform) with no cycles

#### Scenario: Shared type usage prevents cycles

- **WHEN** two subpackages need the same type
- **THEN** that type SHALL be at root level to prevent circular imports

### Requirement: Public API backward compatibility

The public API of the build package SHALL remain unchanged.

#### Scenario: Pipeline interface unchanged

- **WHEN** commands import the build package
- **THEN** `build.Pipeline` interface SHALL remain identical

#### Scenario: NewPipeline factory unchanged

- **WHEN** commands create a pipeline
- **THEN** `build.NewPipeline(cfg)` SHALL continue to work with same signature

#### Scenario: RenderResult structure unchanged

- **WHEN** commands process render results
- **THEN** `build.RenderResult` type and all its fields SHALL remain unchanged

> **Note**: `RenderResult.Release` (type `ModuleReleaseMetadata`) is the current field name. Any references to the old name `Module` in documentation are historical and should be updated to `Release`.

#### Scenario: Error types unchanged

- **WHEN** commands handle errors
- **THEN** all public error types (UnmatchedComponentError, TransformError, etc.) SHALL remain unchanged

### Requirement: Test organization

Test files SHALL be located in the same package as the code they test.

#### Scenario: Module tests

- **WHEN** module package code exists
- **THEN** module tests SHALL be in `internal/build/module/*_test.go`

#### Scenario: Release tests

- **WHEN** release package code exists
- **THEN** release tests SHALL be in `internal/build/release/*_test.go`

#### Scenario: Transform tests

- **WHEN** transform package code exists
- **THEN** transform tests SHALL be in `internal/build/transform/*_test.go`

#### Scenario: Shared test fixtures

- **WHEN** test fixtures are used by multiple packages
- **THEN** testdata/ SHALL remain at `internal/build/testdata/`

### Requirement: Test coverage maintained

The reorganization SHALL NOT reduce test coverage.

#### Scenario: All existing tests pass

- **WHEN** reorganization is complete
- **THEN** all existing tests SHALL pass without modification to test logic

#### Scenario: Coverage verification

- **WHEN** reorganization is complete
- **THEN** `task test:coverage` SHALL show coverage equal to or greater than pre-refactor

### Requirement: No code duplication

Code SHALL NOT be duplicated between root and subpackages.

#### Scenario: Function moved to subpackage

- **WHEN** a function is moved from root to a subpackage
- **THEN** the original function SHALL be deleted from root

#### Scenario: Type moved to subpackage

- **WHEN** a type is moved from root to a subpackage
- **THEN** the original type definition SHALL be deleted from root

### Requirement: Import path correctness

Subpackages SHALL use correct import paths for internal and external dependencies.

#### Scenario: Subpackage imports shared types

- **WHEN** a subpackage needs LoadedComponent
- **THEN** it SHALL import `"github.com/opmodel/cli/internal/build"`

#### Scenario: Import order maintained

- **WHEN** files have imports
- **THEN** imports SHALL be ordered: stdlib, external, internal (per Go conventions)

### Requirement: Behavior preservation

The reorganization SHALL NOT change runtime behavior of the build pipeline.

#### Scenario: Same render output

- **WHEN** a module is rendered before and after reorganization
- **THEN** the RenderResult SHALL be identical

#### Scenario: Same error handling

- **WHEN** errors occur during rendering
- **THEN** error types and messages SHALL be identical

#### Scenario: Same resource ordering

- **WHEN** resources are returned
- **THEN** they SHALL be in the same weight-based order

### Requirement: Documentation updates

Documentation SHALL reflect the new package structure.

#### Scenario: AGENTS.md updated

- **WHEN** reorganization is complete
- **THEN** AGENTS.md project structure section SHALL document new subpackages

#### Scenario: Package README created

- **WHEN** reorganization is complete
- **THEN** `internal/build/README.md` SHALL document package organization and usage

### Requirement: Phased implementation

The reorganization SHALL be implemented in sequential, independently testable phases.

#### Scenario: Phase completion

- **WHEN** a phase is complete
- **THEN** all tests SHALL pass before proceeding to next phase

#### Scenario: Phase independence

- **WHEN** a phase fails
- **THEN** it SHALL be possible to rollback that phase without affecting previous phases

#### Scenario: Phase order

- **WHEN** implementing phases
- **THEN** they SHALL proceed in order: module → release → transform → cleanup

### Requirement: Build success

The CLI binary SHALL build successfully after reorganization.

#### Scenario: Task build passes

- **WHEN** reorganization is complete
- **THEN** `task build` SHALL complete without errors

#### Scenario: Binary functionality

- **WHEN** binary is built
- **THEN** `./bin/opm mod apply` SHALL work on test fixtures

### Requirement: Root pipeline contains orchestration logic

> **Note**: The original design anticipated a thin facade delegating to a separate `orchestration/` subpackage. This was not implemented because creating that subpackage introduced import cycles. The accepted design keeps full orchestration logic in `pipeline.go`.

The root `pipeline.go` SHALL contain the full pipeline orchestration logic, directly calling into `module/`, `release/`, and `transform/` subpackages.

#### Scenario: Pipeline orchestration in root

- **WHEN** a module is rendered
- **THEN** `pipeline.go` in the root `build` package SHALL orchestrate all phases (module loading, release building, transformation)

#### Scenario: Public API entry point

- **WHEN** commands create a pipeline
- **THEN** `build.NewPipeline(cfg)` SHALL remain the sole public entry point

#### Scenario: File size

- **WHEN** pipeline.go is implemented
- **THEN** it SHALL not exceed 300 lines

### Requirement: Module and ModuleRelease identifiers are distinct

Internal Go identifiers (function names, type names, field names, variable names) SHALL accurately reflect whether they operate on a Module (a CUE definition directory) or a ModuleRelease (a concrete deployed instance). Identifiers that represent release concepts SHALL use "Release" in their name, not "Module".

This requirement applies to internal packages only. User-facing terminal output (success messages, log prefixes, command descriptions) is governed by the UX principle that users think in terms of modules and is explicitly excluded.

#### Scenario: Release-scoped function names

- **WHEN** a function operates on or returns ModuleRelease data (release name, release ID, release namespace, release status)
- **THEN** its Go identifier SHALL include "Release" rather than "Module"

#### Scenario: Release-scoped type names

- **WHEN** a type's primary role is to represent or track a ModuleRelease concept
- **THEN** its Go identifier SHALL include "Release" or another accurate descriptor rather than "Module"

#### Scenario: Inventory types unambiguous

- **WHEN** an inventory type has fields for both the source module name and the release name
- **THEN** the field names SHALL be distinct and unambiguous (e.g., `ModuleName` vs `ReleaseName`), not both named `Name` or `Namespace`

#### Scenario: Dead types removed

- **WHEN** a type is defined but never instantiated or type-asserted anywhere in the codebase
- **THEN** it SHALL be removed to prevent false expectations for future contributors

#### Scenario: JSON backward compatibility preserved

- **WHEN** a Go field is renamed for clarity
- **THEN** its JSON struct tag SHALL remain unchanged to preserve deserialization of existing serialized data (e.g., existing inventory Secrets on clusters)
