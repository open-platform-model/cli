## Why

The current `internal/build/` package conflates all pipeline concerns — module loading, release building, matching, and generation — in one place, making incremental replacement impossible. Moving it to `internal/legacy/` explicitly signals that it is being phased out and keeps it operational while new phase-structured packages are built alongside it.

## What Changes

- Move `internal/build/` directory → `internal/legacy/` (directory rename, no logic changes)
- Update 6 import paths from `github.com/opmodel/cli/internal/build` → `github.com/opmodel/cli/internal/legacy`
- Package declarations inside `legacy/` remain `package build` (no source changes required)

## Capabilities

### New Capabilities

_None. This is a pure structural refactor with no behavior changes._

### Modified Capabilities

_None. No spec-level behavior changes._

## Impact

- `internal/cmdutil/render.go` — import path update
- `internal/cmdutil/render_test.go` — import path update
- `internal/cmdutil/output.go` — import path update
- `internal/cmdutil/output_test.go` — import path update
- `internal/cmd/mod/verbose_output_test.go` — import path update
- `experiments/module-full-load/single_load_test.go` — import path update
- SemVer: **PATCH** — internal refactor, no public API or CLI behavior changes
