## Context

`internal/cmd` is a single flat Go package (`package cmd`) with 24 files. All commands share six package-level mutable variables (`opmConfig`, `resolvedBaseConfig`, `verboseFlag`, `registryFlag`, `configFlag`, `timestampsFlag`) that are populated by `PersistentPreRunE` and read via accessor functions (`GetOPMConfig()`, `GetRegistry()`, etc.). Three additional per-command vars (`modInitTemplate`, `modInitDir`, `configInitForce`) live at package scope unnecessarily. The `verboseFlag` global is also consumed inside `cmdutil.ShowRenderOutput` through the calling package's scope, creating hidden coupling. Two helper functions in `mod_status.go` (`runStatusOnce`, `displayStatus`) duplicate ~20 lines of logic.

## Goals / Non-Goals

**Goals:**

- Split `internal/cmd` into sub-packages `internal/cmd/mod/` and `internal/cmd/config/`
- Introduce `GlobalConfig` struct to replace package-level vars and accessor functions
- Pass `*GlobalConfig` explicitly into every sub-command constructor (dependency injection)
- Eliminate per-command package-level flag vars; make them function-local
- Capitalise `exitCodeFromK8sError` → `ExitCodeFromK8sError` for cross-package access
- Merge `runStatusOnce` / `displayStatus` duplication in `mod_status.go`
- Move test files into the sub-package alongside their source
- Update `AGENTS.md` project structure tree

**Non-Goals:**

- Changes to `internal/cmdutil/` — no structural changes there
- Changes to `cmd/opm/main.go` — stays identical
- Any behaviour, flag, or API changes
- Adding new commands or functionality

## Decisions

### D1: `GlobalConfig` lives in `root.go`, pointer threaded at wire-up time

`NewRootCmd()` allocates one `*GlobalConfig`, `PersistentPreRunE` fills it, and it is passed into `mod.NewModCmd(cfg)` and `configcmd.NewConfigCmd(cfg)` at command-tree construction. Each sub-command constructor receives `cfg` and captures it in its `RunE` closure.

**Alternative considered**: inject config via `cobra.Command.SetContext` (storing in `context.Context`). Rejected — adds indirection with no benefit; type-safe struct field access is cleaner and idiomatic.

**Alternative considered**: keep accessors but move them to a separate `internal/cmd/globals` package. Rejected — perpetuates the global-state anti-pattern; `GlobalConfig` injection solves the root cause.

### D2: `exit.go` and its symbols stay in `package cmd`

`ExitError`, exit code constants, and `ExitCodeFromK8sError` remain in `internal/cmd/exit.go`. Sub-packages import `internal/cmd` for these. This avoids moving them to `cmdutil` (which would widen `cmdutil`'s responsibility) and keeps `main.go` unchanged.

**Circular import check**: `internal/cmd/mod` imports `internal/cmd` for `ExitError` and `ExitCodeFromK8sError`. `internal/cmd` does **not** import `internal/cmd/mod`. No cycle.

### D3: `specs` artifact skipped — no spec-level behaviour changes

This is a pure structural refactor. No new capabilities, no requirement changes. The `specs/` directory will not be created.

### D4: `runStatusOnce` / `displayStatus` merged into single helper

Both functions call `kubernetes.GetModuleStatus`, handle `ignoreNotFound`, and print output. The only difference is exit-code mapping and output format selection (watch mode always uses table). A single `fetchAndPrintStatus(ctx, client, opts, logName, ignoreNotFound, forWatch bool)` covers both cases cleanly.

## Risks / Trade-offs

- [Risk: import cycle `internal/cmd/mod` → `internal/cmd`] → Verified safe: `internal/cmd` root files (`root.go`, `exit.go`, `version.go`) never import sub-packages. Dependency is one-way.
- [Risk: test package names break] → Tests move into their sub-package (`package mod`, `package config`). The one test using `package cmd` (`exit_test.go`) stays in `internal/cmd/`.
- [Risk: `verbose_output_test.go` has no `package cmd` dependency] → Confirmed: it only uses `cmdutil` and `build` directly. Moves cleanly to `internal/cmd/mod/`.
