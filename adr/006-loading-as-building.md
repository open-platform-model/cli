# ADR-006: Loading as Building

## Status

Accepted

## Context

The CLI needs to produce a concrete, validated `*ModuleRelease` from user-provided CUE source files and values. An earlier design used a separate `builder.Build()` phase after loading — the loader produced a raw module, and the builder transformed it into a release. This separation created duplication: both phases performed validation, both manipulated CUE values, and the boundary between "loaded" and "built" was unclear. CUE evaluation naturally handles what the builder did imperatively — unification, defaults, constraint checking — so the separate phase added ceremony without value. Two distinct entry points exist: module-directory loading (used by `opm mod` commands) and standalone release-file loading (used by `opm release` commands).

## Decision

Eliminate the separate builder phase. Loading IS building — the loader validates consumer values and produces a concrete `*ModuleRelease` directly. Both entry points converge on `LoadModuleReleaseFromValue()`, which runs the module gate, concreteness check, metadata extraction, and value finalization in a single pass.

- Module-directory path: `LoadReleasePackage` -> `LoadModuleReleaseFromValue`
- Standalone release file path: `LoadReleaseFile` -> `LoadModuleReleaseFromValue`

Values are selected with a fallback chain: explicit `--values` files -> `values.cue` -> `debugValues` -> error. When `--values` is provided, both `values.cue` and `debugValues` are ignored. Auto-secrets handling is delegated entirely to CUE (`#AutoSecrets` in the v1alpha1 core schema), not Go-side injection. Release metadata and labels are derived by CUE evaluation, not Go code. UUIDs are deterministic and repeatable — identical inputs produce identical UUIDs across builds.

Remove `builder.Build()` function; move values file resolution to `internal/cmdutil/` as a CLI-layer concern. See also ADR-007 for the overlay construction and validation approach used during loading.

## Consequences

**Positive:** Single loading path eliminates confusion about where validation happens — there is one place, not two.

**Positive:** Two entry points share one validation contract, preventing divergence between module and release workflows.

**Positive:** CUE-driven auto-secrets and metadata keep Go code declarative rather than imperative.

**Positive:** Deterministic UUIDs enable reproducible builds and reliable change detection.

**Negative:** The loader becomes a larger function with more responsibilities than a pure "load from disk" operation.

**Trade-off:** Values fallback chain (--values > values.cue > debugValues) is convenient but may surprise users who expect explicit values to merge with defaults rather than replace them.
