## 1. Extend config/resolver with ResolvedConfig and ResolveAll

- [x] 1.[1-6] Add `ResolvedField` struct to `config/resolver.go` with `Value string`, `Source Source`, `Shadowed map[Source]string`
- [x] 1.[1-6] Add `SourceConfigAuto Source = "config-auto"` constant for provider auto-resolution
- [x] 1.[1-6] Add `ResolvedConfig` struct to `config/resolver.go` with fields: ConfigPath, Registry, Kubeconfig, Context, Namespace, Provider (all `ResolvedField`), plus Output `string`
- [x] 1.[1-6] Add `ResolveAllOptions` struct with flag values for all fields, config values from `*Config`, and provider names from `map[string]cue.Value`
- [x] 1.[1-6] Implement `ResolveAll(opts ResolveAllOptions) (*ResolvedConfig, error)` — resolve each field using Flag > Env > Config > Default precedence, reusing existing `ResolveConfigPath` and `ResolveRegistry` internally
- [x] 1.[1-6] Implement provider auto-resolution within `ResolveAll`: if `--provider` flag is empty and exactly one provider exists, auto-select it with source `config-auto`
- [x] 1.7 Write tests for `ResolveAll`: flag overrides env, env overrides config, config overrides default, default used when nothing set, provider auto-resolution with 0/1/N providers, flag overrides auto-resolution

## 2. Store resolved state in cmd/root.go

- [x] 2.[1-6] Add package-level `opmConfig *config.OPMConfig` and `resolvedConfig *config.ResolvedConfig` variables
- [x] 2.[1-6] Add `GetOPMConfig() *config.OPMConfig` and `GetResolvedConfig() *config.ResolvedConfig` accessor functions
- [x] 2.[1-6] Update `initializeGlobals` to call `config.ResolveAll()` after `LoadOPMConfig`, passing flag values, config values from `opmConfig.Config`, and provider names from `opmConfig.Providers`
- [x] 2.[1-6] Store results in the new package-level variables
- [x] 2.[1-6] Update existing `Get*()` functions (`GetKubeconfig`, `GetContext`, `GetNamespace`, `GetRegistry`, `GetProvider`) to delegate to `resolvedConfig` fields
- [x] 2.[1-6] Remove the standalone `resolvedRegistry` variable (replaced by `resolvedConfig.Registry`)

## 3. Update "initializing CLI" log

- [x] 3.[12] Replace raw flag values in the "initializing CLI" debug log with resolved values from `resolvedConfig` (kubeconfig, context, namespace, config path, registry, provider)
- [x] 3.[12] Remove `registry_flag` from the log output (redundant with resolved registry)

## 4. Remove redundant debug logs from config/loader.go

- [x] 4.[1-4] Remove `output.Debug("bootstrap: extracted registry from config", ...)` at loader.go:61
- [x] 4.[1-4] Remove `output.Debug("setting CUE_REGISTRY for config load", ...)` at loader.go:171
- [x] 4.[1-4] Remove `output.Debug("extracted provider from config", ...)` at loader.go:254
- [x] 4.[1-4] Remove `output.Debug("extracted providers from config", ...)` at loader.go:261

## 5. Remove redundant debug logs from build packages

- [x] 5.[1-4] Remove `output.Debug("release built successfully", ...)` at build/release_builder.go:191 (Build method)
- [x] 5.[1-4] Remove `output.Debug("release built successfully", ...)` at build/release_builder.go:253 (BuildFromValue method)
- [x] 5.[1-4] Remove `output.Debug("loading provider", ...)` at build/provider.go:62
- [x] 5.[1-4] Update `output.Debug("extracted transformer", ...)` at build/provider.go:141 to use FQN as `name` field and remove the separate `fqn` field

## 6. Remove duplicate LoadOPMConfig calls from subcommands

- [x] 6.[1-6] Update `cmd/mod_apply.go` to use `GetOPMConfig()` instead of calling `config.LoadOPMConfig()`, and use `GetResolvedConfig()` for resolved field values
- [x] 6.[1-6] Update `cmd/mod_build.go` to use `GetOPMConfig()` instead of calling `config.LoadOPMConfig()`, and use `GetResolvedConfig()` for resolved field values
- [x] 6.[1-6] Update `cmd/mod_diff.go` to use `GetOPMConfig()` instead of calling `config.LoadOPMConfig()`, and use `GetResolvedConfig()` for resolved field values
- [x] 6.[1-6] Update `cmd/config_vet.go` to use `GetOPMConfig()` instead of re-loading config
- [x] 6.[1-6] Update `resolveFlag()` helper in `cmd/mod_apply.go` to accept `config.ResolvedField` as the fallback instead of a raw string
- [x] 6.[1-6] Update all `resolveFlag` call sites in apply, delete, diff, status to use `GetResolvedConfig()` fields

## 7. Update "rendering module" log to show resolved provider

- [x] 7.[12] In `cmd/mod_apply.go`, ensure the `provider` variable used in `output.Debug("rendering module", ...)` comes from the resolved config (not empty flag)
- [x] 7.[12] In `cmd/mod_diff.go`, same fix for the "rendering module for diff" log if applicable

## 8. Validation

- [x] 8.1 Run `task fmt` and fix any formatting issues
- [x] 8.2 Run `task test` and fix any failures
- [x] 8.3 Run `task check` (fmt + vet + test) and confirm all pass
- [x] 8.4 Manual smoke test: run `opm mod apply . --name jellyfin --namespace default --verbose` and verify the debug output shows resolved values, no redundant lines, and provider is populated
