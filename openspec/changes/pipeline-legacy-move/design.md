## Context

`internal/build/` is the current monolithic pipeline package containing module loading, release building, matching, and generation. It will be replaced incrementally by focused phase packages (`internal/loader/`, `internal/builder/`, `internal/provider/`, `internal/transformer/`, `internal/pipeline/`). During that transition, the existing code must remain operational. Moving it to `internal/legacy/` makes the intent explicit without requiring any logic changes upfront.

Six files outside the package import `internal/build`:
- `internal/cmdutil/render.go`
- `internal/cmdutil/render_test.go`
- `internal/cmdutil/output.go`
- `internal/cmdutil/output_test.go`
- `internal/cmd/mod/verbose_output_test.go`
- `experiments/module-full-load/single_load_test.go`

## Goals / Non-Goals

**Goals:**
- Move `internal/build/` → `internal/legacy/` with zero behavior changes
- Update all 6 external import paths in one commit
- Leave all package declarations (`package build`) untouched inside the moved files

**Non-Goals:**
- Renaming package declarations inside `internal/legacy/`
- Any refactoring of the code being moved
- Updating `AGENTS.md` project structure tree (done separately)

## Decisions

**Keep `package build` declarations inside `legacy/`**

Alternatives considered:
- Rename to `package legacy` throughout — adds churn with no benefit during transition; callers still use the directory-qualified import anyway.
- Keep as `package build` — Go allows the directory name (`legacy`) to differ from the package name (`build`). Callers reference it as `build.Pipeline`, `build.RenderOptions`, etc., unchanged except for the import path. No source changes needed inside the moved files.

**Single commit for directory move + import path updates**

The move and the 6 import path updates must land together. A half-applied state (directory moved but imports not updated, or vice versa) breaks the build. One atomic commit keeps `task test` green throughout.

## Risks / Trade-offs

- **`git mv` vs manual copy** → Use `git mv internal/build internal/legacy` to preserve file history. A copy-delete loses history in `git log --follow`.
- **Subpackage imports** → `internal/build/module`, `internal/build/release`, `internal/build/transform` also move. Check for any direct imports of subpackages beyond the 6 known files.
