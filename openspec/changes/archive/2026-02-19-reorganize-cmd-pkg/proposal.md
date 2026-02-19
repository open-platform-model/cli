## Why

`internal/cmd` is a flat 24-file package where all commands share mutable package-level globals (`opmConfig`, `verboseFlag`, etc.) accessed via accessor functions. This makes commands hard to test in isolation and couples unrelated commands through hidden shared state. As the command surface grows, the flat layout becomes increasingly difficult to navigate.

## What Changes

- Split `internal/cmd` into sub-packages `internal/cmd/mod/` and `internal/cmd/config/` mirroring the cobra command tree
- Introduce `GlobalConfig` struct in `root.go` to replace package-level vars and accessor functions
- Thread `*GlobalConfig` explicitly into each sub-command constructor (dependency injection)
- Move per-command flag vars (`modInitTemplate`, `modInitDir`, `configInitForce`) from package scope to function-local scope
- Capitalise `exitCodeFromK8sError` → `ExitCodeFromK8sError` so sub-packages can call it
- Merge duplicate `runStatusOnce`/`displayStatus` helpers in `mod_status.go`
- Move test files into the same sub-package as their source
- Update `AGENTS.md` project structure tree

No behaviour changes. No flag or API changes. `main.go` is unchanged.

## Capabilities

### New Capabilities
<!-- none — pure refactor -->

### Modified Capabilities
<!-- none — no spec-level behaviour changes -->

## Impact

- `internal/cmd/` — restructured into sub-packages
- `internal/cmdutil/` — no changes
- `cmd/opm/main.go` — no changes
- `AGENTS.md` — project structure tree updated
- SemVer: PATCH (internal refactor, no public interface change)
