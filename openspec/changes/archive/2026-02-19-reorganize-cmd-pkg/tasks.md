## 1. Prepare `internal/cmd` root

- [x] 1.1 Add `GlobalConfig` struct to `root.go` with fields: `OPMConfig`, `ConfigPath`, `Registry`, `RegistryFlag`, `Verbose`
- [x] 1.2 Replace package-level vars (`opmConfig`, `resolvedBaseConfig`, `verboseFlag`, `registryFlag`, `configFlag`, `timestampsFlag`) with a single `var cfg GlobalConfig` in `NewRootCmd`
- [x] 1.3 Rewrite `initializeGlobals` to populate `&cfg` instead of the old package vars
- [x] 1.4 Remove accessor functions (`GetOPMConfig`, `GetConfigPath`, `GetRegistry`, `GetRegistryFlag`)
- [x] 1.5 Capitalise `exitCodeFromK8sError` → `ExitCodeFromK8sError` in `exit.go`
- [x] 1.6 Update `exit_test.go` to call `ExitCodeFromK8sError`

## 2. Create `internal/cmd/mod/` sub-package

- [x] 2.1 Create `internal/cmd/mod/mod.go` (`package mod`) — `NewModCmd(cfg *cmd.GlobalConfig)` wires all sub-commands
- [x] 2.2 Create `internal/cmd/mod/init.go` from `mod_init.go` — move `modInitTemplate`/`modInitDir` to function-local, accept `cfg`
- [x] 2.3 Create `internal/cmd/mod/build.go` from `mod_build.go` — replace `GetOPMConfig()`/`GetRegistry()`/`verboseFlag` with `cfg`
- [x] 2.4 Create `internal/cmd/mod/vet.go` from `mod_vet.go` — same substitutions
- [x] 2.5 Create `internal/cmd/mod/apply.go` from `mod_apply.go` — same substitutions
- [x] 2.6 Create `internal/cmd/mod/diff.go` from `mod_diff.go` — same substitutions
- [x] 2.7 Create `internal/cmd/mod/delete.go` from `mod_delete.go` — replace accessors, use `cmd.ExitCodeFromK8sError`
- [x] 2.8 Create `internal/cmd/mod/status.go` from `mod_status.go` — merge `runStatusOnce`/`displayStatus` into `fetchAndPrintStatus(ctx, client, opts, logName, ignoreNotFound, forWatch bool)`; replace accessors, use `cmd.ExitCodeFromK8sError`

## 3. Create `internal/cmd/config/` sub-package

- [x] 3.1 Create `internal/cmd/config/config.go` (`package config`) — `NewConfigCmd(cfg *cmd.GlobalConfig)` wires sub-commands
- [x] 3.2 Create `internal/cmd/config/init.go` from `config_init.go` — move `configInitForce` to function-local, accept `cfg`
- [x] 3.3 Create `internal/cmd/config/vet.go` from `config_vet.go` — replace `GetConfigPath()`/`GetRegistryFlag()` with `cfg`

## 4. Wire sub-packages into root and update `root.go`

- [x] 4.1 Update `NewRootCmd` to call `mod.NewModCmd(&cfg)` and `configcmd.NewConfigCmd(&cfg)` and `NewVersionCmd(&cfg)`
- [x] 4.2 Update `version.go` to accept `cfg *GlobalConfig` (unused but consistent interface)
- [x] 4.3 Remove old flat source files: `mod.go`, `mod_init.go`, `mod_build.go`, `mod_vet.go`, `mod_apply.go`, `mod_diff.go`, `mod_delete.go`, `mod_status.go`, `config.go`, `config_init.go`, `config_vet.go`

## 5. Move test files

- [x] 5.1 Move `mod_apply_test.go` → `internal/cmd/mod/apply_test.go`, update package to `package mod`
- [x] 5.2 Move `mod_build_test.go` → `internal/cmd/mod/build_test.go`, update package to `package mod`
- [x] 5.3 Move `mod_init_test.go` → `internal/cmd/mod/init_test.go`, update package, update refs to former pkg-level vars
- [x] 5.4 Move `mod_status_test.go` → `internal/cmd/mod/status_test.go`, update package to `package mod`
- [x] 5.5 Move `mod_vet_test.go` → `internal/cmd/mod/vet_test.go`, update package to `package mod`
- [x] 5.6 Move `verbose_output_test.go` → `internal/cmd/mod/verbose_output_test.go`, update package to `package mod`
- [x] 5.7 Move `config_init_test.go` → `internal/cmd/config/init_test.go`, update package to `package config`
- [x] 5.8 Move `config_vet_test.go` → `internal/cmd/config/vet_test.go`, update package to `package config`
- [x] 5.9 Move `version_test.go` → `internal/cmd/version_test.go`, update signature if needed

## 6. Update AGENTS.md and validate

- [x] 6.1 Update `AGENTS.md` project structure tree to reflect new sub-packages
- [x] 6.2 Run `task build` — ensure it compiles
- [x] 6.3 Run `task test` — ensure all tests pass
- [x] 6.4 Run `task check` — fmt + vet + test all green
