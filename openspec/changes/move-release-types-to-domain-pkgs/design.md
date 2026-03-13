## Context

`ModuleRelease` and `BundleRelease` types are currently defined in `pkg/render/`, alongside rendering logic (matching, execution, validation). This couples domain model types to a specific processing pipeline. The existing specs (`pkg-types`, `core-modulerelease`) planned separate `pkg/modulerelease/` and `pkg/bundlerelease/` packages, but the codebase never followed through — types landed in `pkg/render/` instead.

The current layout:
- `pkg/module/module.go` — `Module`, `ModuleMetadata`
- `pkg/bundle/bundle.go` — `Bundle`, `BundleMetadata`
- `pkg/render/modulerelease.go` — `ModuleRelease`, `ModuleReleaseMetadata`, `NewModuleRelease()`, accessors
- `pkg/render/bundlerelease.go` — `BundleRelease`, `BundleReleaseMetadata`

Consumers must import `pkg/render` just to reference a `ModuleRelease` type, even if they have nothing to do with rendering.

## Goals / Non-Goals

**Goals:**
- Colocate release types with their parent domain packages (`pkg/module/`, `pkg/bundle/`)
- Eliminate the need to import `pkg/render` for domain types
- Update specs to reflect the simpler package structure (no `pkg/modulerelease/` or `pkg/bundlerelease/`)

**Non-Goals:**
- Refactoring rendering logic within `pkg/render/`
- Changing type names, signatures, or behavior
- Moving any other types out of `pkg/render/`

## Decisions

### Decision 1: Colocate in parent domain packages, not separate release packages

**Choice**: `pkg/module/release.go` and `pkg/bundle/release.go`
**Alternative**: `pkg/modulerelease/` and `pkg/bundlerelease/` (as current specs describe)

**Rationale**: A release is a deployment instance of a module or bundle — it's inherently part of that domain. Separate packages for 1-2 types each create unnecessary indirection. `pkg/module/` already has `Module` and `ModuleMetadata`; adding `ModuleRelease` and `ModuleReleaseMetadata` keeps the domain cohesive. Same for `pkg/bundle/`.

### Decision 2: Accept bundle → module dependency

**Choice**: `pkg/bundle/` imports `pkg/module/` for `BundleRelease.Releases map[string]*module.ModuleRelease`
**Alternative**: Use an interface or keep both in a shared package

**Rationale**: A bundle release *contains* module releases — this is a fundamental domain relationship, not accidental coupling. The dependency is one-way (`bundle → module`), no cycle risk. Using an interface would add indirection for no real benefit.

### Decision 3: Rename types idiomatically

**Choice**: Rename to `module.Release`, `module.ReleaseMetadata`, `bundle.Release`, `bundle.ReleaseMetadata`
**Alternative**: Keep names identical (`module.ModuleRelease`, `bundle.BundleRelease`)

**Rationale**: In Go, it is considered bad practice for type names to repeat the package name (the "stutter" rule). Since we are moving these types into `pkg/module` and `pkg/bundle`, renaming them to simply `Release` and `ReleaseMetadata` makes them idiomatic (e.g., `rel *module.Release`). This eliminates the need for linter suppressions (`//nolint:revive`).

## Risks / Trade-offs

- [Package size growth] `pkg/module/` and `pkg/bundle/` each gain ~50-75 lines → Acceptable; still focused single-responsibility packages.
- [Import churn] ~8 files need import updates → Mechanical change, low risk. `task check` validates.
- [Spec divergence] Existing specs describe `pkg/modulerelease/` and `pkg/bundlerelease/` → Delta specs update these requirements as part of this change.
