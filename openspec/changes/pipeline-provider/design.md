## Context

Provider loading currently lives in `internal/build/transform/provider.go` as `LoadProvider()`, which returns both a `*core.Provider` (for matching) and `[]*LoadedTransformer` (for execution). This dual-return exists because the old executor needed its own `LoadedTransformer` type alongside the core type. Now that `core-transformer-match-plan-execute` is complete and `TransformerMatchPlan.Execute()` operates directly on `*core.Transformer`, the `LoadedTransformer` parallel track is no longer needed.

The new `internal/provider/` package isolates this one responsibility: given a provider name and a map of provider CUE values from `GlobalConfig`, parse transformer definitions and return a single structured `*LoadedProvider`. The legacy code remains in `internal/legacy/` during the transition; the new package is additive.

## Goals / Non-Goals

**Goals:**
- Create `internal/provider/` with a clean `Load()` function signature (no dual return)
- Expose `*LoadedProvider` with a `Transformers []*core.Transformer` field and a `Requirements()` helper
- Support auto-selection when exactly one provider is configured
- Return meaningful errors for missing providers, empty transformer lists, and parse failures
- Be independently unit-testable without a real CUE module on disk

**Non-Goals:**
- Replacing the legacy `LoadProvider` function — it stays in `internal/legacy/` until the full pipeline migration is complete
- Component matching — that is `internal/transformer/`'s job
- Transformer execution — owned by `core.TransformerMatchPlan.Execute()`
- Any CLI command or flag changes

## Decisions

### D1: Single return value — `*LoadedProvider` wrapping `[]*core.Transformer`

**Decision**: `Load()` returns `(*LoadedProvider, error)` only, no parallel `[]*LoadedTransformer` slice.

**Rationale**: The dual return in the legacy code was a migration shim. `core.Transformer` now carries everything needed for both matching (criteria fields) and execution (`Transform cue.Value`, `Metadata`). A single `*LoadedProvider` struct with a `Transformers []*core.Transformer` field is sufficient and cleaner.

**Alternative considered**: Keep `LoadedTransformer` in `internal/provider/` as a separate type. Rejected — it would duplicate `core.Transformer` for no gain now that execution is on `core.TransformerMatchPlan`.

### D2: `cueCtx *cue.Context` is a parameter to `Load()`, not stored on the struct

**Decision**: The caller passes a `*cue.Context`; `LoadedProvider` does not cache it.

**Rationale**: Consistent with the project convention of fresh CUE contexts per command (AGENTS.md). Storing the context on a struct risks context leakage across commands. The context is only needed during the `Load()` call to extract transformer criteria from the CUE value; `core.Provider.CueCtx` (set by the caller after loading) remains the place to store it for execution.

**Alternative considered**: Store `cueCtx` on `LoadedProvider` and set `core.Provider.CueCtx` automatically. Rejected — blurs the boundary between loading (provider's job) and execution setup (pipeline's job).

### D3: `Requirements()` returns FQNs, not transformer names

**Decision**: `LoadedProvider.Requirements()` returns a `[]string` of FQNs (`provider#transformer`).

**Rationale**: FQNs are the identifier used in error messages and in `TransformerMatchPlan` throughout the codebase. Returning short names would require callers to reconstruct FQNs for display. The `Requirements()` method exists to support readable error messages in the matcher ("no components matched; available transformers: kubernetes#deployment, kubernetes#service").

### D4: Error when provider has zero transformers

**Decision**: Return an error if the `transformers` field is absent or empty, rather than returning an empty `LoadedProvider`.

**Rationale**: A provider with no transformers is always a misconfiguration — there is nothing useful the pipeline can do with it. Failing loudly at load time surfaces the problem before any matching or execution begins, consistent with Principle I (validate early).

**Alternative considered**: Return an empty `LoadedProvider` and let the matcher fail later. Rejected — deferred errors are harder to diagnose.

## Risks / Trade-offs

- **Risk**: Callers in `internal/transformer/` and `internal/pipeline/` must be written against the new signature rather than the legacy one.
  → **Mitigation**: Those packages don't exist yet; this change is purely additive and does not modify any existing callers.

- **Risk**: `core.Transformer` fields (`RequiredResources`, `RequiredTraits`, etc.) are `map[string]cue.Value`, which is harder to test than `[]string`.
  → **Mitigation**: Unit tests can construct `cue.Value` test fixtures using a real `cue.Context` with string literal injection. The spec scenarios that reference slice-style criteria (`requiredResources`, `requiredTraits`) map to the map keys of these fields.

- **Trade-off**: Duplicating `extractLabelsField`, `extractMapKeys`, `extractCueValueMap` from the legacy package vs. importing from legacy.
  → New package copies the helpers (they are small and stable). Importing from `internal/legacy/` would create a dependency in the wrong direction and block future deletion of legacy code.

## Open Questions

_None — all decisions above are resolved._
