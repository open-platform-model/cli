## Context

In v0, the CUE `#Module` definition had a `values` field for author-provided concrete defaults, and `#config` was constraints-only. The CLI loader supported two patterns for populating `Module.Values`:

- **Pattern A**: `values.cue` exists alongside `module.cue` — loader reads it separately via `CompileBytes`, extracts the `values` field, stores it in `mod.Values`.
- **Pattern B**: No `values.cue` — loader falls back to an inline `values` field in `module.cue` (from `mod.Raw`).

In v1alpha1, `#Module` no longer has a `values` field (replaced by `debugValues` for test/debug). Defaults now live inside `#config` itself (e.g., `port: int | *8096`). The `values.cue` file still provides concrete author defaults, but as an external file — not as part of the `#Module` definition.

The loader still filters `values*.cue` from the package load because `#Module` is a closed CUE definition — including `values.cue` in `load.Instances` would cause a CUE error since `values` is not a field on `#Module`.

## Goals / Non-Goals

**Goals:**

- Remove `Module.Values`, `Module.HasValuesCue`, and `Module.SkippedValuesFiles` from the Go `Module` struct
- Move all values resolution logic from loader to builder
- Simplify the loader to only filter `values*.cue` files and extract metadata/config/components
- Simplify `pipeline.prepare()` by removing values-related debug logging
- Maintain the existing `--values` flag behavior

**Non-Goals:**

- Changing `ModuleRelease.Values` (the release still carries concrete values)
- Handling `debugValues` (out of scope for this change)
- Changing the `--values` CLI flag interface
- Supporting building without any values source (values.cue or --values is always required)

## Decisions

### Decision 1: Builder owns all values resolution

**Choice**: Move values.cue discovery and loading from the loader into the builder's `selectValues` function.

**Rationale**: The loader's job is "parse the module definition" — extracting metadata, `#config`, and `#components` from the CUE evaluation. Values are a build-time concern: the builder already handles `--values` files and validation against `#config`. Having the builder also discover `values.cue` from `mod.ModulePath` eliminates the state-passing via `Module.Values` and makes the loader simpler.

**Alternative considered**: Keep values loading in the loader but just remove Pattern B. Rejected because it leaves the loader doing work that conceptually belongs to the builder, and keeps the `Module.Values` field as a pass-through.

### Decision 2: Loader keeps values*.cue filtering

**Choice**: The loader continues to filter all `values*.cue` files from the `load.Instances` file list.

**Rationale**: `#Module` is a closed CUE definition. If `values.cue` (which defines a `values` field) is included in the package load, CUE will reject it because `values` is not a field on `#Module`. This is a CUE package load constraint, not a values logic concern — so it stays in the loader.

### Decision 3: values.cue or --values is a hard gate

**Choice**: The builder errors if neither `--values` files nor a `values.cue` in the module directory are available. No fallback to `#config` defaults.

**Rationale**: Even though `#config` now carries defaults, requiring explicit values keeps the build deterministic and auditable. The author's `values.cue` is the concrete "this is what I intend" declaration. Building without it would be implicit — violating Principle VII (explicit over magic).

### Decision 4: --values replaces values.cue entirely

**Choice**: When `--values` files are provided, `values.cue` is completely ignored. The `--values` files are the sole values source.

**Rationale**: CUE unification is commutative — there's no "override" semantic. If we unified `--values` with `values.cue`, the result would be the CUE unification of both, which could produce unexpected constraint tightening. Clean replacement avoids this.

### Decision 5: Remove Module.Values without replacement

**Choice**: Remove the field entirely rather than renaming or repurposing it.

**Rationale**: No consumer reads `rel.Module.Values` — only `rel.Values` (on `ModuleRelease`). The builder sets `ModuleRelease.Values` directly from `selectValues` output. The field on `Module` was pure pass-through.

## Risks / Trade-offs

- **[Risk] Integration test `values-flow` directly asserts `mod.Values`** → Update the test to only assert on `rel.Values` (the release-level values). The test's purpose is to verify values flow through the pipeline, which `rel.Values` still covers.
- **[Risk] External consumers of `Module.Values`** → The struct is in `internal/`, so no external consumers exist. Grep confirmed no reads of `rel.Module.Values` anywhere.
- **[Risk] Builder now needs `mod.ModulePath` to find `values.cue`** → `ModulePath` is already a required field on `Module` and is always set by the loader. No new dependency.
- **[Trade-off] Debug logging for filtered values files moves from pipeline to builder** → Acceptable. The builder is the right place for values-related logging since it now owns the full values pipeline.
