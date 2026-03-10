# Refactor Checklist

## Goals

- [x] Make `opm module` the canonical command name
- [x] Keep `opm mod` as a compatibility alias
- [x] Rename `internal/cmd/mod` to `internal/cmd/module`
- [x] Reduce `internal/cmdutil` to a small CLI-helper package
- [x] Consolidate duplicated `module` / `release` workflows
- [x] Update `AGENTS.md` after the refactor lands

## Phase 1: Command Rename

- [x] Rename `internal/cmd/mod` to `internal/cmd/module`
- [x] Rename the package to `modulecmd` to avoid confusion with `pkg/module`
- [x] Make `opm module` the canonical Cobra command
- [x] Keep `mod` as an alias for `module`
- [x] Update root command wiring in `internal/cmd/root.go`
- [x] Update command tests and help text to prefer `opm module`
- [x] Keep deprecated compatibility messaging only where still intended

## Phase 2: Render Workflow Extraction

- [x] Extract shared render orchestration out of `internal/cmdutil`
- [x] Keep source-specific loading separate for module paths vs release files
- [x] Normalize both sources behind one shared render workflow result
- [x] Migrate `build` commands to the shared render workflow
- [x] Migrate `vet` commands to the shared render workflow
- [x] Migrate `diff`-style release rendering to the shared render workflow

## Phase 3: Apply Workflow Consolidation

- [x] Extract shared apply orchestration from `internal/cmd/module/apply.go`
- [x] Extract shared apply orchestration from `internal/cmd/release/apply.go`
- [x] Share namespace creation, inventory loading, safety checks, apply, prune, and inventory write flow
- [x] Keep source-specific change metadata injectable
- [x] Make both apply commands thin wrappers over the shared workflow

## Phase 4: Query Workflow Consolidation

- [x] Consolidate shared status workflow
- [x] Consolidate shared tree workflow
- [x] Consolidate shared events workflow
- [x] Consolidate shared delete workflow
- [x] Consolidate shared list workflow
- [x] Move inventory resolution helpers out of catch-all `cmdutil`

## Phase 5: Shrink `internal/cmdutil`

- [x] Keep only CLI-specific helpers in `internal/cmdutil`
- [x] Keep flag groups in `internal/cmdutil/flags.go`
- [x] Keep annotations and tiny release arg helpers in `internal/cmdutil`
- [x] Move workflow orchestration to dedicated internal packages
- [x] Re-home output composition helpers if they are not truly CLI-only

## Phase 6: Documentation

- [x] Update `AGENTS.md` project structure
- [x] Update `AGENTS.md` key packages
- [x] Replace stale references to old internal package layout
- [x] Document any new workflow packages introduced by the refactor

## Validation

- [x] Run targeted command package tests during each phase
- [x] Run shared utility tests as files move
- [x] Run a broader test pass after the refactor stabilizes
- [x] Verify `opm module ...` works
- [x] Verify `opm mod ...` still works as an alias

## Test Relocation Cleanup

- [x] Move render-owned tests to `internal/workflow/render`
- [x] Move query-owned tests to `internal/workflow/query`
- [x] Move apply-owned tests to `internal/workflow/apply`
- [x] Move output-formatting tests to `internal/output`
- [x] Move exit-code constant tests to `pkg/errors`
- [x] Keep only true `cmdutil` tests in `internal/cmdutil`

## Final Polish

- [x] Review remaining `cmdutil` comments for stale wording
- [x] Fix remaining `internal/cmdutil/output_test.go` and `internal/cmdutil/render_test.go` issues
- [x] Relocate workflow-owned tests out of `internal/cmdutil`
- [x] Trim deprecated compatibility messaging only where it still adds value
