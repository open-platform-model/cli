## Context

`internal/core` currently holds 11 Go files spanning pure type definitions, CUE extraction logic, matching algorithms, and transformer execution. The package doc claims it has no CUE dependency, but CUE appears in 7 of those files. This makes the package's purpose opaque and its boundaries undefined.

The CUE catalog at `catalog/v0/core` is already organized by concept: `module.cue`, `module_release.cue`, `transformer.cue`, `provider.cue`, `component.cue`. The Go layer should mirror this — each CUE type gets a corresponding Go package holding its struct definitions, receiver methods, and closely related helpers.

The existing phase packages (`internal/loader`, `internal/builder`, `internal/pipeline`) already implement clean separation. This change completes that pattern at the domain type level.

## Goals / Non-Goals

**Goals:**

- Split `internal/core` into focused subpackages, one per CUE catalog concept
- Establish a clean linear import chain with no circular dependencies
- Absorb `internal/transformer` (warnings only) into `internal/core/transformer`
- Consolidate `TransformError`/`ValidationError` into `internal/errors` alongside existing CLI errors
- Keep all tests passing, all behavior identical

**Non-Goals:**

- Changing any logic, algorithm, or behavior
- Completing the legacy `MatchPlan` / `ToLegacyMatchPlan` migration (deferred)
- Renaming types or methods
- Changing the public CLI interface

## Decisions

### Decision 1: Subpackages nested under `internal/core/` rather than top-level

`internal/core/component`, `internal/core/module`, etc. rather than `internal/component`, `internal/module`.

**Rationale**: The subpackages are domain *types*, not pipeline phases. Nesting under `core/` signals "these are the core data model" and groups them visually. Top-level packages like `internal/loader` and `internal/builder` are phase packages — a different concern.

**Alternative considered**: Flat top-level packages. Rejected because it conflates domain types with operational phases in the package listing.

### Decision 2: `Component` gets its own subpackage (`internal/core/component`)

Rather than placing `Component` under `internal/core/module`.

**Rationale**: `Component` is used by `module`, `modulerelease`, `transformer`, and `provider`. If it lived in `module`, all those packages would import `module` just for the `Component` type, creating unnecessary coupling. Its own package mirrors `component.cue` in the catalog.

**Alternative considered**: Under `module`. Rejected due to transitive coupling — every consumer of `Component` would pull in `Module` as well.

### Decision 3: Linear import chain, bottom-up

```
core ← component ← module ← modulerelease ← transformer ← provider
```

Each package only imports packages earlier in the chain. No cycles are possible.

**Rationale**: Enforces a clear layering that mirrors the conceptual dependency: a Provider matches Components from a ModuleRelease built from a Module.

### Decision 4: `validation.go` stays unexported in `modulerelease`

`validateFieldsRecursive` and `pathRewrittenError` are CUE internals called only by `ModuleRelease.ValidateValues()`. They move to `internal/core/modulerelease/validation.go` as unexported symbols.

**Rationale**: No other package needs these helpers. Exposing them would be YAGNI.

**Alternative considered**: Separate `internal/core/cueutil` package. Rejected — only one caller, no justification for a new package.

### Decision 5: `CollectWarnings` absorbed into `internal/core/transformer`

`internal/transformer/warnings.go` operates on `*TransformerMatchPlan`, which moves to `internal/core/transformer`. Moving `CollectWarnings` with its subject eliminates a package.

**Rationale**: The function is conceptually part of the transformer matching domain. Keeping it in a separate `internal/transformer` package was only necessary when `TransformerMatchPlan` lived in `internal/core` (a different package).

### Decision 6: `TransformError` and `ValidationError` move to `internal/errors/domain.go`

Rather than staying in a `internal/core/errors.go`.

**Rationale**: Centralizes all error types in one package. The `internal/errors` package is split into three files to keep CLI infrastructure errors (`errors.go`, `sentinel.go`) visually separate from domain errors (`domain.go`).

**Risk**: `ErrValidation` (sentinel) and `ValidationError` (concrete type) live in the same package with similar names. Mitigated by file-level separation and the semantic difference being clear from the type names.

### Decision 7: Migration order is bottom-up along the import chain

Create and migrate packages in dependency order: `component` → `module` → `modulerelease` → `transformer` → `provider`. Each step compiles before the next begins.

**Rationale**: Ensures the codebase compiles at each step of the migration, making it easier to validate and revert if needed.

## Risks / Trade-offs

- **[Wide import churn]** ~30 files update their import paths. → Mitigated by the mechanical nature of the change (find-and-replace) and full test coverage verifying correctness.
- **[Legacy MatchPlan]** `ToLegacyMatchPlan`, `MatchPlan`, `TransformerMatchOld` move with `TransformerMatchPlan` to `internal/core/transformer` but remain as technical debt. → Accepted; deferred cleanup is tracked separately.
- **[Package name `modulerelease`]** Compound word without separator is Go-conventional but slightly verbose at call sites (`modulerelease.ModuleRelease{}`). → Accepted; mirrors the CUE filename convention and is unambiguous.

## Migration Plan

Execute in this order to keep the codebase compiling at each step:

1. Create `internal/errors/domain.go` — move `TransformError`, `ValidationError`; split `errors.go` into `errors.go` + `sentinel.go`; update all consumers
2. Create `internal/core/component/` — move `component.go` + tests; update `loader` and `builder`
3. Create `internal/core/module/` — move `module.go` + tests; update `loader` and `builder`
4. Create `internal/core/modulerelease/` — move `module_release.go` + `validation.go` + tests; update `builder` and `pipeline`
5. Create `internal/core/transformer/` — move transformer types, context, match plan, execution; absorb `internal/transformer/warnings.go`; delete `internal/transformer/`; update `loader`, `pipeline`, `cmdutil`
6. Create `internal/core/provider/` — move provider types and matching logic; update `loader` and `pipeline`
7. Delete emptied files from `internal/core/`; verify only `resource.go`, `labels.go`, `weights.go` remain
8. Update `AGENTS.md` package tree
9. Run `task test` — all tests must pass

**Rollback**: Each step is independently revertible via git. No database migrations or external state changes.

## Open Questions

- None. All decisions were settled during exploration.
