## Why

With all phase packages in place (`loader`, `builder`, `provider`, `transformer`), the final step is a new orchestrator that ties them together behind a `Pipeline` interface and replaces `internal/legacy/`. This change creates `internal/pipeline/`, updates `cmdutil` to use it, and deletes `internal/legacy/` — completing the pipeline redesign.

## What Changes

- Create `internal/pipeline/pipeline.go` implementing the `Pipeline` interface: sequences PREPARATION → BUILD → MATCHING → GENERATE, collects results and errors
- Create `internal/pipeline/types.go` with `Pipeline` interface, `RenderOptions`, `RenderResult` (replacing equivalents in `internal/legacy/`)
- Update `internal/cmdutil/render.go` to import `internal/pipeline` instead of `internal/legacy`
- Update `internal/cmdutil/output.go` import path
- Update `internal/cmd/mod/verbose_output_test.go` import path
- Delete `internal/legacy/` entirely

## Capabilities

### New Capabilities

- `render-pipeline`: The `Pipeline` interface and full orchestration of PREPARATION → BUILD → MATCHING → GENERATE phases, replacing the legacy monolithic pipeline with the new phase-structured implementation

### Modified Capabilities

_None._

## Impact

- New package `internal/pipeline/`
- `internal/cmdutil/render.go` — import updated, `Pipeline` type reference updated
- `internal/cmdutil/output.go` — import updated
- `internal/cmd/mod/verbose_output_test.go` — import updated
- `internal/legacy/` — deleted
- Depends on: `internal/loader/`, `internal/builder/`, `internal/provider/`, `internal/transformer/`, `internal/core/`
- SemVer: **MINOR** — internal restructure, no CLI behavior or flag changes
